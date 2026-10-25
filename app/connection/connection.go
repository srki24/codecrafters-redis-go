package connection

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/codecrafters-io/redis-starter-go/app/cmd"
	"github.com/codecrafters-io/redis-starter-go/app/config"
	"github.com/codecrafters-io/redis-starter-go/app/db"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

func handleUserConn(connetion net.Conn, db *db.Database) {
	defer connetion.Close()
	connStatus := cmd.NewConnStatus()
	var err error
	for {
		if err != nil {
			connetion.Write([]byte(err.Error()))
			err = nil
			continue
		}
		reader := bufio.NewReader(connetion)
		var buff []byte = make([]byte, 2048)
		n, err := reader.Read(buff)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			continue
		}
		request, _, err := resp.Parse(buff[:n])
		if err != nil {
			continue
		}
		command, err := cmd.ParseCommand(request)
		if err != nil {
			continue
		}
		response := cmd.GenerateResponse(command, db, &connStatus)
		connetion.Write(response.Serialize())
	}

}

func Open(appConfig *config.Config, database *db.Database) {
	l, err := net.Listen(
		"tcp", fmt.Sprintf("0.0.0.0:%d", appConfig.Port))
	if err != nil {
		log.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			log.Printf("accept %v", err)
			continue
		}
		go handleUserConn(conn, database)
	}
}

func Handshake(appConfig *config.Config) error {
	host, port, err := appConfig.ParseReplica()
	if err != nil {
		return err
	}
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", host, port))
	defer conn.Close()
	if err != nil {
		return err
	}

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
		_, err = conn.Write(respMsg.Serialize())

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
