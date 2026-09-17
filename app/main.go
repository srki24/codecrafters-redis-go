package main

import (
	"bufio"
	"fmt"
	"net"
	"os"

	"github.com/codecrafters-io/redis-starter-go/app/cmd"
	"github.com/codecrafters-io/redis-starter-go/app/db"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

func handleConn(conn net.Conn, db db.Database) {
	defer conn.Close()

read:
	for {

		reader := bufio.NewReader(conn)
		var buff []byte = make([]byte, 2048)

		n, err := reader.Read(buff)

		if err != nil {
			fmt.Println(err)
			continue read
		}
		request, _ := resp.Parse(buff[:n])
		command, err := cmd.ParseCommand(request)

		if err != nil {
			fmt.Println(err)
		}
		response := cmd.GenerateResponse(command, db)

		conn.Write(response.Serialize())
	}

}

func main() {

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}

	db := db.InitializeDb()
	for {

		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleConn(conn, db)
	}

}
