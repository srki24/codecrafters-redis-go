package main

import (
	"fmt"
	"net"
	"os"

	"strings"

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

		var response resp.RESPValue
		switch strings.ToUpper(command.Name) {
		case "ECHO":
			response = resp.BulkString{Data: []byte(command.Args[0])}
		case "PING":
			response = resp.SimpleString{Data: []byte("PONG")}
		case "SET":
			err = cmd.SetMapping(command, mapping)
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
			err = cmd.ListPush(command, listMapping, true)
			if err != nil {
				fmt.Println(err)
			}
			response = resp.Integer{Data: cmd.GetNrElems(command, listMapping)}

		case "LPUSH":
			err = cmd.ListPush(command, listMapping, false)
			if err != nil {
				fmt.Println(err)
			}
			response = resp.Integer{Data: cmd.GetNrElems(command, listMapping)}

		case "LRANGE":
			data, err := cmd.LRange(command, listMapping)
			if err != nil {
				fmt.Println(err)
			}
			response = resp.NewArray(data)
		case "LLEN":
			response = resp.Integer{Data: cmd.GetNrElems(command, listMapping)}
		case "LPOP":
			var data []string
			data, err = cmd.ListPop(command, listMapping)
			if err != nil {
				fmt.Println(err)
			}
			if len(data) == 1 {
				response = resp.BulkString{Data: []byte(data[0])}
			} else {
				response = resp.NewArray(data)
			}
		case "BLPOP":
			data, err := cmd.ListBLPop(command, listMapping)
			if err != nil {
				fmt.Println(err)
			}
			response = resp.NewArray(data)
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
