package cmd

import (
	"errors"
	"fmt"

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
