package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}

	conn, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	for {

		var buff []byte = make([]byte, 1024)
		n, _ := conn.Read(buff)

		for _, ln := range strings.Split(string(buff[:n]), "\r\n") {
			if ln == "PING" {
				conn.Write([]byte("+PONG\r\n"))
			}

		}
	}

}
