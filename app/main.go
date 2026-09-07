package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

func hanleConn(conn net.Conn) {
	defer conn.Close()

	for {

		var buff []byte = make([]byte, 1024)
		n, err := conn.Read(buff)

		if err != nil {
			return
		}

		for _, ln := range strings.Split(string(buff[:n]), "\r\n") {
			if ln == "PING" {
				conn.Write([]byte("+PONG\r\n"))
			}

		}
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
