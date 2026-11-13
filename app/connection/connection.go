package connection

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

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
	channels []chan resp.RESPValue
}

func (rh *replicaHub) addChannel(ch chan resp.RESPValue) {
	rh.mu.Lock()
	defer rh.mu.Unlock()
	rh.channels = append(rh.channels, ch)
}

func (rh *replicaHub) removeChannel(ch chan resp.RESPValue) {
	rh.mu.Lock()
	defer rh.mu.Unlock()
	rh.channels = slices.DeleteFunc(rh.channels, func(c chan resp.RESPValue) bool {
		return c == ch
	})
}

func (rh *replicaHub) publish(v resp.RESPValue) {
	rh.mu.Lock()
	defer rh.mu.Unlock()

	for _, ch := range rh.channels {
		ch <- v
	}
}

func parseRequest(connection net.Conn) chan resp.RESPValue {
	out := make(chan resp.RESPValue)
	go func() {
		defer close(out)
		reader := bufio.NewReader(connection)
		for {
			var start, read int
			var buff []byte = make([]byte, 2048)
			read, err := reader.Read(buff[start:])
			if errors.Is(err, io.EOF) {
				log.Printf("eof: %v", err)
				time.Sleep(time.Millisecond * 100)
				continue
			}
			if err != nil {
				return
			}
			for {
				request, consumed, err := resp.Parse(buff[start:read])
				if err != nil {
					log.Printf("failed to parse buff %v", err)
					break // partial read
				}
				out <- request
				start += consumed
				if start == read {
					break
				}

			}
		}
	}()
	return out
}

func clientLoop(connection net.Conn, connStatus *cmd.ConnStatus, db *db.Database, rh *replicaHub) {
	var err error
	for respRequest := range parseRequest(connection) {
		if err != nil {
			connection.Write([]byte(err.Error()))
			err = nil
			continue
		}
		command, err := cmd.ParseCommand(respRequest)
		if err != nil {
			continue
		}
		response := cmd.GenerateResponse(command, db, connStatus)
		if !connStatus.IsReplica() || slices.Contains(REPLICA_COMMANDS, strings.ToUpper(command.Name)) {
			_, err = connection.Write(response.Serialize())
			if err != nil {
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

func replicaLoop(connection net.Conn, connStatus *cmd.ConnStatus, rh *replicaHub) {
	data, err := rdb.Serialize()
	if err != nil {
		log.Printf("failed serialize rdb: %v", err)
		return
	}
	if _, err = connection.Write(data); err != nil {
		log.Printf("failed write rdb: %v", err)
		return
	}
	ch := make(chan resp.RESPValue)
	rh.addChannel(ch)
	defer rh.removeChannel(ch)
	defer close(ch)

	connStatus.MarkReplicated()
	for data := range ch {
		// write channel commands to appropriate replica
		if _, err := connection.Write(data.Serialize()); err != nil {
			log.Printf("failed to write to replica: %v", err)
			return
		}
	}
}

func handlClientConn(connection net.Conn, db *db.Database, connStatus *cmd.ConnStatus, rh *replicaHub) {
	defer connection.Close()
	clientLoop(connection, connStatus, db, rh)
	if connStatus.ShouldRepeplicate() {
		replicaLoop(connection, connStatus, rh)
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
	err = handshake(conn, appConfig)
	if err != nil {
		return err
	}

	// TODO PARSE RDB FILE
	buf := make([]byte, 2048)
	conn.Read(buf)

	connStatus := cmd.NewConnStatus()
	connStatus.MarkReplicated() // replica cannot be replicated further down
	connStatus.MarkAsReplica()

	replicaHub := replicaHub{}
	go clientLoop(conn, &connStatus, database, &replicaHub)
	return nil
}

func handshake(conn net.Conn, appConfig *config.Config) error {
	messages := [][]string{
		{"PING"},
		{"REPLCONF", "listening-port", strconv.Itoa(appConfig.Port)},
		{"REPLCONF", "capa", "psync"},
		{"PSYNC", "?", "-1"},
	}
	buf := make([]byte, 2048)
	for _, msg := range messages {
		respMsg := resp.NewArray(msg)
		_, err := conn.Write(respMsg.Serialize())

		if err != nil {
			return err
		}
		// waiting for response
		n, err := conn.Read(buf)
		if err != nil {
			return err
		}
		log.Println(string(buf[:n]))
	}
	return nil
}
