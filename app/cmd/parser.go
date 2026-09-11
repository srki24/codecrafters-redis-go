package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

type Command struct {
	Name string
	Args []string
}

func parseStringCommand(input resp.RESPValue) (Command, error) {
	cmd := input.GetStringData()
	if len(cmd) == 0 {
		return Command{}, errors.New("missing input data")
	}

	return Command{Name: cmd}, nil
}

func parseArrayCommand(input resp.Array) (Command, error) {

	if len(input.Data) < 1 {
		return Command{}, errors.New("Array with no data given")
	}

	var cmd string
	var args []string

	inputCmd := input.Data[0]
	switch inputCmd := inputCmd.(type) {
	case resp.SimpleString, resp.BulkString:
		cmd = inputCmd.GetStringData()
	default:
		return Command{}, fmt.Errorf("First element of a command must be either BulkString or SimpleString, got %T", inputCmd)
	}

	for _, v := range input.Data[1:] {
		args = append(args, v.GetStringData())
	}

	return Command{cmd, args}, nil

}

func ParseCommand(input resp.RESPValue) (Command, error) {
	switch input := input.(type) {
	case resp.SimpleString, resp.BulkString:
		return parseStringCommand(input)
	case resp.Array:
		return parseArrayCommand(input)
	default:
		return Command{}, errors.New("failed to parse, unknown input value")
	}
}

func GenerateResponse(command Command, mapping map[string]Mapping, listMapping ListMapping) resp.RESPValue {

	var response resp.RESPValue
	switch strings.ToUpper(command.Name) {
	case "ECHO":
		response = resp.BulkString{Data: []byte(command.Args[0])}
	case "PING":
		response = resp.SimpleString{Data: []byte("PONG")}
	case "SET":
		err := SetMapping(command, mapping)
		if err != nil {
			fmt.Println(err)
			break
		}
		response = resp.SimpleString{Data: []byte("OK")}

	case "GET":
		if v, err := GetMapping(command, mapping); err == nil {
			response = resp.BulkString{Data: []byte(v)}
		} else {
			fmt.Println(err)
			response = resp.BulkString{}
		}

	case "RPUSH":
		err := ListPush(command, listMapping, true)
		if err != nil {
			fmt.Println(err)
		}
		response = resp.Integer{Data: GetNrElems(command, listMapping)}

	case "LPUSH":
		err := ListPush(command, listMapping, false)
		if err != nil {
			fmt.Println(err)
		}
		response = resp.Integer{Data: GetNrElems(command, listMapping)}

	case "LRANGE":
		data, err := LRange(command, listMapping)
		if err != nil {
			fmt.Println(err)
		}
		response = resp.NewArray(data)
	case "LLEN":
		response = resp.Integer{Data: GetNrElems(command, listMapping)}
	case "LPOP":
		var data []string
		data, err := ListPop(command, listMapping)
		if err != nil {
			fmt.Println(err)
		}
		if len(data) == 1 {
			response = resp.BulkString{Data: []byte(data[0])}
		} else {
			response = resp.NewArray(data)
		}
	case "BLPOP":
		data, err := ListBLPop(command, listMapping)
		if err != nil {
			fmt.Println(err)
			response = resp.NewNullArray()
		} else {
			response = resp.NewArray(data)

		}
	}
	return response
}
