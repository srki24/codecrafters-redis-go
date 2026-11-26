package connection

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"slices"
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/aof"
	"github.com/codecrafters-io/redis-starter-go/app/cmd"
	"github.com/codecrafters-io/redis-starter-go/app/config"
	"github.com/codecrafters-io/redis-starter-go/app/db"
	"github.com/codecrafters-io/redis-starter-go/app/rdb"
	"github.com/codecrafters-io/redis-starter-go/app/repl"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
	"github.com/codecrafters-io/redis-starter-go/app/sub"
)

var WRITE_COMMANDS = []string{"SET", "DEL", "INCR", "RPUSH", "LPUSH", "LPOP", "RPOP", "BLPOP", "BRPOP", "XADD", "EXEC", "MULTI"}
var REPLICA_COMMANDS = []string{"INFO", "REPLCONF", "PSYNC"}
var SUBSCRIBE_COMMANDS = []string{"SUBSCRIBE", "PUBLISH"}

func clientLoop(
	rw *ConnRW,
	connStatus *cmd.ConnStatus,
	db *db.Database,
	rh *repl.Hub,
	sh *sub.Hub,
) {
	var err error
	var writting bool = false
	var response resp.RESPValue
	for respRequest := range resp.ParseRequest(rw) {
		if err != nil {
			rw.Write([]byte(err.Error()))
			err = nil
			continue
		}

		command, err := cmd.ParseCommand(respRequest)
		if err != nil {
			continue
		}

		response = nil
		if connStatus.IsSub() || slices.Contains(SUBSCRIBE_COMMANDS, strings.ToUpper(command.Name)) {
			response = cmd.GenerateSubResponse(command, connStatus, sh)
		} else {
			response = cmd.GenerateResponse(command, db, connStatus, rh)
		}

		// replica should reply to master only in specific case
		if response != nil &&
			(!connStatus.IsReplica() ||
				slices.Contains(REPLICA_COMMANDS, strings.ToUpper(command.Name))) {
			_, err = rw.Write(response.Serialize())
			if err != nil {
				fmt.Printf("err: %v\n", err)
				continue
			}
		}
		// replica connection with master
		if connStatus.IsReplica() && connStatus.IsReplicated() {
			db.AddReplicaOffset(respRequest.Serialize())

		}

		// write commands
		if slices.Contains(WRITE_COMMANDS, strings.ToUpper(command.Name)) {
			err = aof.WriteCommanand(respRequest.Serialize())

			// master only
			// send command to all of the replicas channels
			if db.IsMaster() {
				db.AddReplicaOffset(respRequest.Serialize())
				rh.Publish(respRequest)
			}
		}
		// Master exit the client loop and replicate
		if connStatus.ShouldRepeplicate() && !writting {
			writting = true
			go replicaWriter(rw, rh)
		}

	}
}

func subWriter(ctx context.Context, w *ConnRW, connStatus *cmd.ConnStatus) {
	for {
		select {
		case <-ctx.Done():
			return
		case data, ok := <-connStatus.Outbound():
			if !ok {
				return
			}
			if _, err := w.Write(data); err != nil {
				log.Printf("faile to write to channel %v", err)
				return
			}
		}
	}
}

func replicaWriter(w *ConnRW, rh *repl.Hub) {
	if _, err := w.Write(rdb.RDB{}.Serialize()); err != nil {
		return
	}
	ch := make(chan repl.Serializeable, 16)
	rh.AddChannel(ch)
	defer rh.RemoveChannel(ch)
	defer close(ch)
	for data := range ch {
		// write channel commands to appropriate replica
		if _, err := w.Write(data.Serialize()); err != nil {
			log.Printf("failed to write to replica: %v", err)
			return
		}
	}
}

func handlClientConn(
	connection net.Conn,
	db *db.Database,
	connStatus *cmd.ConnStatus,
	rh *repl.Hub,
	subHub *sub.Hub) {
	defer connection.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rw := NewConnRW(connection)
	go subWriter(ctx, &rw, connStatus)
	clientLoop(&rw, connStatus, db, rh, subHub)

}

func OpenClient(appConfig *config.Config, database *db.Database) error {
	l, err := net.Listen(
		"tcp", fmt.Sprintf("0.0.0.0:%d", appConfig.Port))
	if err != nil {
		return err
	}
	replicaHub := repl.NewHub()
	subHub := sub.NewHub()
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
		go handlClientConn(conn, database, &connStatus, &replicaHub, &subHub)
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

	replicaHub := repl.NewHub()
	subHub := sub.NewHub()
	go clientLoop(&rw, &connStatus, database, &replicaHub, &subHub)
	return nil
}

func handshake(rw *ConnRW, appConfig *config.Config) error {
	messages := [][]string{
		{"PING"},
		{"REPLCONF", "listening-port", strconv.Itoa(appConfig.Port)},
		{"REPLCONF", "capa", "psync"},
		{"PSYNC", "?", "-1"},
	}
	for _, msg := range messages {
		respMsg := resp.NewArray(msg)
		_, err := rw.Write(respMsg.Serialize())

		if err != nil {
			return err
		}
		// waiting for response
		if _, err = rw.ReadBytes('\n'); err != nil { // +FULLRESYNC <id> 0\r\n
			if !errors.Is(err, io.EOF) {
				return nil
			}
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
	if err != nil && !errors.Is(err, io.EOF) {
		fmt.Println("EOFF")
		return err
	}
	return err
}
