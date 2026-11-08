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
	"sync"

	"github.com/codecrafters-io/redis-starter-go/app/cmd"
	"github.com/codecrafters-io/redis-starter-go/app/config"
	"github.com/codecrafters-io/redis-starter-go/app/db"
	"github.com/codecrafters-io/redis-starter-go/app/rdb"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

var WRITE_COMMANDS = []string{"SET", "DEL", "INCR", "RPUSH", "LPUSH", "LPOP", "RPOP", "BLPOP", "BRPOP", "XADD", "EXEC", "MULTI"}

type replicaHub struct {
	mu       sync.Mutex
	channels []chan resp.RESPValue
}

func (rh *replicaHub) addChannel(ch chan resp.RESPValue) {
	rh.mu.Lock()
	defer rh.mu.Unlock()
	rh.channels = append(rh.channels, ch)
}

func parseRequest(connection net.Conn) (resp.RESPValue, error) {
	reader := bufio.NewReader(connection)
	var buff []byte = make([]byte, 2048)
	n, err := reader.Read(buff)
	if errors.Is(err, io.EOF) {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	request, _, err := resp.Parse(buff[:n])
	if err != nil {
		return nil, err
	}
	return request, nil
}

func clientLoop(connection net.Conn, connStatus *cmd.ConnStatus, db *db.Database, rh *replicaHub) chan resp.RESPValue {
	var err error
	for {
		if err != nil {
			connection.Write([]byte(err.Error()))
			err = nil
			continue
		}
		request, err := parseRequest(connection)
		if err != nil {
			continue
		}
		command, err := cmd.ParseCommand(request)
		if err != nil {
			continue
		}
		response := cmd.GenerateResponse(command, db, connStatus)
		connection.Write(response.Serialize())

		// send command to all of the replicas
		for _, ch := range rh.channels {
			if slices.Contains(WRITE_COMMANDS, command.Name) {
				ch <- request
			}
		}
		// client loop exiting
		if connStatus.ShouldRepeplicate() {
			ch := make(chan resp.RESPValue)
			rh.addChannel(ch)
			return ch // break client loop - connection becomes replica connection
		}
	}
}

func replicaLoop(connection net.Conn, connStatus *cmd.ConnStatus, ch chan resp.RESPValue) {
	defer close(ch)
	data, err := rdb.Serialize()
	if err != nil {
		log.Printf("failed serialize rdb: %v", err)
	}
	_, err = connection.Write(data)
	if err == nil {
		log.Printf("failed write rdb: %v", err)
	}
	connStatus.MarkReplicated()
	for data := range ch {
		connection.Write(data.Serialize())
	}
}

func handlClientConn(connection net.Conn, db *db.Database, rh *replicaHub) {
	defer connection.Close()

	connStatus := cmd.NewConnStatus()
	ch := clientLoop(connection, &connStatus, db, rh)
	if connStatus.ShouldRepeplicate() {
		replicaLoop(connection, &connStatus, ch)
	}

}

func OpenClient(appConfig *config.Config, database *db.Database) error {
	l, err := net.Listen(
		"tcp", fmt.Sprintf("0.0.0.0:%d", appConfig.Port))
	replicaHub := replicaHub{}
	if err != nil {
		return err
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			log.Printf("accept %v", err)
			continue
		}
		go handlClientConn(conn, database, &replicaHub)
	}
}

func handleReplicaConn(connection net.Conn, db *db.Database) {
	defer connection.Close()
	connStatus := cmd.NewConnStatus()
	var err error
	for {
		if err != nil {
			connection.Write([]byte(err.Error()))
			err = nil
			continue
		}
		request, err := parseRequest(connection)
		if err != nil {
			continue
		}
		command, err := cmd.ParseCommand(request)
		if err != nil {
			continue
		}
		_ = cmd.GenerateResponse(command, db, &connStatus)
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
	go handleReplicaConn(conn, database)
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
		fmt.Println(msg)
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
		fmt.Println(buf[:n])
	}
	return nil
}
