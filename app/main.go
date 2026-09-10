package main

import (
	"fmt"
	"net"
	"os"
	"strings"

	cmd "github.com/codecrafters-io/redis-starter-go/app/commands"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

func hanleConn(conn net.Conn) {
	defer conn.Close()

	for {

		var buff []byte = make([]byte, 1024)
		n, err := conn.Read(buff)

		if err != nil {
			return
		}

		request, _ := resp.Parse(buff[:n])
		command, err := cmd.ParseCommand(request)

		var response resp.RESPValue
		switch strings.ToUpper(command.Name) {
		case "ECHO":
			response = resp.BulkString{Data: []byte(command.Args[0])}
		case "PING":
			response = resp.SimpleString{Data: []byte("PONG")}
		}

		conn.Write(response.Serialize())

	}
}
func main() {

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}

	for {

		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go hanleConn(conn)
	}

}
