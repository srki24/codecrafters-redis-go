package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	cmd "github.com/codecrafters-io/redis-starter-go/app/commands"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

type Mapping struct {
	value string
	time  time.Time
	exp   int
}

func setMapping(command cmd.Command, mapping map[string]Mapping) (map[string]Mapping, error) {

	args := command.Args
	exp := -1

	if len(args) < 2 {
		return mapping, errors.New("Failed to set mapping, not enough args")
	}
	key := args[0]
	value := args[1]

	if len(args) == 4 {
		option := strings.ToUpper(args[2])
		optionVal := args[3]
		switch option {
		case "EX", "PX":
			optionVal, err := strconv.Atoi(optionVal)
			if err != nil {
				return mapping, fmt.Errorf("Couldn't convert value to an integer: %s", optionVal)
			}
			factor := 1
			if option == "EX" {
				factor = 1000
			}
			exp = optionVal * factor
		default:
			return mapping, errors.New("Unknown option")
		}
	}
	mapping[key] = Mapping{value: value, time: time.Now(), exp: exp}

	return mapping, nil

}

func getMapping(command cmd.Command, mapping map[string]Mapping) (string, error) {
	args := command.Args
	fmt.Println(args)

	if len(args) < 1 {
		return "", errors.New("Failed to get mapping, not enough args")
	}

	k := args[0]

	if v, ok := mapping[k]; ok {

		cTime := time.Now()

		elapsed := int(cTime.Sub(v.time).Milliseconds())

		if (v.exp > 0) && elapsed > v.exp {
			return "", errors.New("Key expired")
		}
		return v.value, nil
	} else {
		return "", errors.New("Missing key")
	}
}

func hanleConn(conn net.Conn) {
	defer conn.Close()

	mapping := make(map[string]Mapping)

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
			mapping, err = setMapping(command, mapping)
			if err != nil {
				fmt.Println(err)
				break
			}
			response = resp.SimpleString{Data: []byte("OK")}

		case "GET":
			if v, err := getMapping(command, mapping); err == nil {
				response = resp.BulkString{Data: []byte(v)}
			} else {
				fmt.Println(err)
				response = resp.BulkString{}
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
