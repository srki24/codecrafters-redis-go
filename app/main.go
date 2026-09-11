package main

import (
	"fmt"
	"net"
	"os"

	"github.com/codecrafters-io/redis-starter-go/app/cmd"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

func handleConn(conn net.Conn, mapping map[string]cmd.Mapping, listMapping cmd.ListMapping) {
	defer conn.Close()

	for {

		var buff []byte = make([]byte, 1024)
		n, err := conn.Read(buff)

		if err != nil {
			return
		}

		request, _ := resp.Parse(buff[:n])
		command, err := cmd.ParseCommand(request)

		response := cmd.GenerateResponse(command, mapping, listMapping)

		conn.Write(response.Serialize())
	}

}

func main() {

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}

	mapping := cmd.NewMapping()
	listMapping := cmd.NewListMapping()

	for {

		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleConn(conn, mapping, listMapping)
	}

}
