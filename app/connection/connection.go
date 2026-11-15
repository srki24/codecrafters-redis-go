package connection

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/codecrafters-io/redis-starter-go/app/cmd"
	"github.com/codecrafters-io/redis-starter-go/app/config"
	"github.com/codecrafters-io/redis-starter-go/app/db"
	"github.com/codecrafters-io/redis-starter-go/app/rdb"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

var WRITE_COMMANDS = []string{"SET", "DEL", "INCR", "RPUSH", "LPUSH", "LPOP", "RPOP", "BLPOP", "BRPOP", "XADD", "EXEC", "MULTI"}
var REPLICA_COMMANDS = []string{"INFO", "REPLCONF", "PSYNC"}

type replicaHub struct {
	mu       sync.Mutex
	channels []chan Serializeable
}

type Serializeable interface {
	Serialize() []byte
}

func (rh *replicaHub) addChannel(ch chan Serializeable) {
	rh.mu.Lock()
	defer rh.mu.Unlock()
	rh.channels = append(rh.channels, ch)
}

func (rh *replicaHub) removeChannel(ch chan Serializeable) {
	rh.mu.Lock()
	defer rh.mu.Unlock()
	rh.channels = slices.DeleteFunc(rh.channels, func(c chan Serializeable) bool {
		return c == ch
	})
}

func (rh *replicaHub) publish(v Serializeable) {
	rh.mu.Lock()
	defer rh.mu.Unlock()

	for _, ch := range rh.channels {
		ch <- v
	}
}

func parseRequest(reader *ConnRW) chan resp.RESPValue {
	out := make(chan resp.RESPValue)
	go func() {
		defer close(out)
		var pending []byte
		tmp := make([]byte, 4096)

		for {
			n, err := reader.Read(tmp)
			if err != nil {
				return
			}
			if n > 0 {
				pending = append(pending, tmp[:n]...)
			}
			for len(pending) > 0 {
				request, consumed, perr := resp.Parse(pending)
				if errors.Is(perr, resp.ErrIncomplete) {
					break // partial read
				}
				if perr != nil {
					return
				}
				out <- request
				pending = pending[consumed:]
			}
		}
	}()
	return out
}

func clientLoop(rw *ConnRW, connStatus *cmd.ConnStatus, db *db.Database, rh *replicaHub) {
	var err error
	for respRequest := range parseRequest(rw) {
		if err != nil {
			rw.Write([]byte(err.Error()))
			err = nil
			continue
		}
		command, err := cmd.ParseCommand(respRequest)
		if err != nil {
			continue
		}

		response := cmd.GenerateResponse(command, db, connStatus)
		if !connStatus.IsReplica() || slices.Contains(REPLICA_COMMANDS, strings.ToUpper(command.Name)) {
			_, err = rw.Write(response.Serialize())
			if err != nil {
				fmt.Printf("err: %v\n", err)
				continue
			}
		}
		// send command to all of the replicas channels
		if db.IsMaster() {
			if slices.Contains(WRITE_COMMANDS, strings.ToUpper(command.Name)) {
				rh.publish(respRequest)
			}
		}
		// client loop exiting
		if connStatus.ShouldRepeplicate() {
			return // break client loop - connection becomes replica connection
		}
	}
}

func replicaLoop(w *ConnRW, connStatus *cmd.ConnStatus, rh *replicaHub) {
	if _, err := w.Write(rdb.RDB{}.Serialize()); err != nil {
		return
	}
	ch := make(chan Serializeable, 16)
	rh.addChannel(ch)
	defer rh.removeChannel(ch)
	defer close(ch)
	for data := range ch {
		// write channel commands to appropriate replica
		if _, err := w.Write(data.Serialize()); err != nil {
			log.Printf("failed to write to replica: %v", err)
			return
		}
	}
}

func handlClientConn(connection net.Conn, db *db.Database, connStatus *cmd.ConnStatus, rh *replicaHub) {
	defer connection.Close()
	rw := NewConnRW(connection)

	clientLoop(&rw, connStatus, db, rh)
	if connStatus.ShouldRepeplicate() {
		replicaLoop(&rw, connStatus, rh)
	}

}

func OpenClient(appConfig *config.Config, database *db.Database) error {
	l, err := net.Listen(
		"tcp", fmt.Sprintf("0.0.0.0:%d", appConfig.Port))
	if err != nil {
		return err
	}
	replicaHub := replicaHub{}
	for {
		conn, err := l.Accept()
		connStatus := cmd.NewConnStatus()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			log.Printf("accept %v", err)
			continue
		}
		go handlClientConn(conn, database, &connStatus, &replicaHub)
	}
}

func OpenReplica(appConfig *config.Config, database *db.Database) error {
	host, port, err := appConfig.ParseReplica()
	if err != nil {
		return err
	}
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		return err
	}
	rw := NewConnRW(conn)
	err = handshake(&rw, appConfig)
	if err != nil {
		return err
	}

	err = readRDB(&rw)
	if err != nil {
		return err
	}

	connStatus := cmd.NewConnStatus()
	connStatus.MarkReplicated() // replica cannot be replicated further down
	connStatus.MarkAsReplica()

	replicaHub := replicaHub{}
	go clientLoop(&rw, &connStatus, database, &replicaHub)
	return nil
}

func handshake(rw *ConnRW, appConfig *config.Config) error {
	messages := [][]string{
		{"PING"},
		{"REPLCONF", "listening-port", strconv.Itoa(appConfig.Port)},
		{"REPLCONF", "capa", "psync"},
		{"PSYNC", "?", "-1"},
	}
	buf := make([]byte, 2048)
	for _, msg := range messages {
		respMsg := resp.NewArray(msg)
		_, err := rw.Write(respMsg.Serialize())

		if err != nil {
			return err
		}
		// waiting for response
		_, err = rw.Read(buf)
		if err != nil {
			return err
		}
	}
	return nil
}

func readRDB(crw *ConnRW) error {
	header, err := crw.ReadBytes('\n') // $88/r/n
	if err != nil {
		return err
	}
	n, err := strconv.Atoi(string(header[1 : len(header)-2]))
	buf := make([]byte, n)
	_, err = crw.ReadFull(buf)
	if errors.Is(err, io.EOF) {
		log.Println("EOF")
		return nil
	}
	return err
}
