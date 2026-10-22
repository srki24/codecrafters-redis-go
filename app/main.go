package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"github.com/codecrafters-io/redis-starter-go/app/cmd"
	"github.com/codecrafters-io/redis-starter-go/app/db"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

type Config struct {
	Port int
}

var AppConfig Config

func parseArgs() {
	flag.IntVar(&AppConfig.Port, "port", 6379, "Port to start the server on")
	flag.Parse()
}

func handleConn(connetion net.Conn, db *db.Database) {
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

func main() {
	parseArgs()
	l, err := net.Listen(
		"tcp", fmt.Sprintf("0.0.0.0:%d", AppConfig.Port))
	if err != nil {
		log.Println("Failed to bind to port 6379")
		os.Exit(1)
	}

	db := db.NewDatabase()
	for {

		conn, err := l.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			log.Printf("accept %v", err)
			continue
		}
		go handleConn(conn, &db)
	}

}
