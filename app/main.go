package main

import (
	"fmt"
	"net"
	"os"

	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/cmd"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

func hanleConn(conn net.Conn) {
	defer conn.Close()

	mapping := cmd.NewMapping()
	listMapping := cmd.NewListMapping()

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
		case "SET":
			mapping, err = cmd.SetMapping(command, mapping)
			if err != nil {
				fmt.Println(err)
				break
			}
			response = resp.SimpleString{Data: []byte("OK")}

		case "GET":
			if v, err := cmd.GetMapping(command, mapping); err == nil {
				response = resp.BulkString{Data: []byte(v)}
			} else {
				fmt.Println(err)
				response = resp.BulkString{}
			}

		case "RPUSH":
			{
				listMapping, err := cmd.AddElement(command, listMapping)
				if err != nil {
					fmt.Println(err)
				}
				response = resp.Integer{Data: cmd.GetNrElems(command, listMapping)}
			}
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
