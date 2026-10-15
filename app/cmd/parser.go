package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/db"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

type Command struct {
	Name string
	Args []string
}

func parseStringCommand(input resp.RESPValue) (Command, error) {
	cmd := input.String()
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
		cmd = inputCmd.String()
	default:
		return Command{}, fmt.Errorf("First element of a command must be either BulkString or SimpleString, got %T", inputCmd)
	}

	for _, v := range input.Data[1:] {
		args = append(args, v.String())
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

func GenerateResponse(command Command, database *db.Database, connState *ConnStatus) (response resp.RESPValue) {

	if ok := connState.QueueCommand(command); ok {
		response = resp.SimpleString{Data: []byte("QUEUED")}
		return
	}

	switch strings.ToUpper(command.Name) {
	case "ECHO":
		if len(command.Args) < 1 {
			response = resp.SimpleError{Data: "No args provided"}
		} else {
			response = resp.BulkString{Data: []byte(command.Args[0])}
		}
	case "PING":
		response = resp.SimpleString{Data: []byte("PONG")}
	case "SET":
		err := Set(command, database)
		if err != nil {
			response = resp.SimpleError{Data: err.Error()}
		} else {
			response = resp.SimpleString{Data: []byte("OK")}
		}

	case "GET":
		if v, err := Get(command, database); err == nil {
			response = resp.BulkString{Data: []byte(v)}
		} else {
			fmt.Println(err)
			response = resp.BulkString{}
		}

	case "RPUSH":
		n, err := ListPush(command, database, true)
		if err != nil {
			response = resp.SimpleError{Data: err.Error()}
		} else {
			response = resp.Integer{Data: n}
		}

	case "LPUSH":
		n, err := ListPush(command, database, false)
		if err != nil {
			response = resp.SimpleError{Data: err.Error()}
		} else {
			response = resp.Integer{Data: n}
		}

	case "LRANGE":
		data, err := LRange(command, database)
		if err != nil {
			fmt.Println(err)
		}
		response = resp.NewArray(data)
	case "LLEN":
		response = resp.Integer{Data: GetNumberOfElments(command, database)}
	case "LPOP":
		var data []string
		data, err := ListPop(command, database)
		if err != nil {
			response = resp.SimpleError{Data: err.Error()}
		} else {
			if len(data) == 1 {
				response = resp.BulkString{Data: []byte(data[0])}
			} else {
				response = resp.NewArray(data)
			}
		}
	case "BLPOP":
		data, err := ListBLPop(command, database)
		if err != nil {
			fmt.Println(err)
			response = resp.NewNullArray()
		} else {
			response = resp.NewArray(data)
		}
	case "TYPE":
		data, err := Type(command, database)
		if err != nil {
			response = resp.SimpleError{Data: err.Error()}
		} else {
			response = resp.SimpleString{Data: []byte(data)}
		}
	case "XADD":
		data, err := Xadd(command, database)
		if err != nil {
			response = resp.SimpleError{Data: err.Error()}
		} else {
			response = resp.BulkString{Data: []byte(data)}
		}
	case "XRANGE":
		data, err := Xrange(command, database)
		if err != nil {
			response = resp.SimpleError{Data: err.Error()}
		} else {
			response = resp.NewArrayFromStream(data)
		}

	case "XREAD":
		{
			keys, streams, err := Xread(command, database)
			if err != nil {
				fmt.Println(err)
				response = resp.SimpleError{Data: err.Error()}
			} else {
				streamResponses := []resp.RESPValue{}
				for i := 0; i < len(keys); i++ {
					key := keys[i]
					stream := streams[i]
					streamKey := resp.BulkString{Data: []byte(key)}
					streamValue := resp.NewArrayFromStream(stream)

					if streamValue.IsNull {
						continue
					}
					streamResponses = append(
						streamResponses,
						resp.Array{Data: []resp.RESPValue{streamKey, streamValue}})
				}
				if len(streamResponses) == 0 {
					response = resp.NewNullArray()
				} else {
					response = resp.Array{Data: streamResponses}
				}
			}
		}
	case "INCR":
		{
			val, err := Incr(command, database)
			if err != nil {
				fmt.Println(err)
				response = resp.SimpleError{Data: err.Error()}
			} else {
				response = resp.Integer{Data: val}
			}
		}
	case "MULTI":
		{
			connState.BeginTransaction()
			response = resp.SimpleString{Data: []byte("OK")}
		}
	case "EXEC":
		{
			if connState.IsWatchingKeyUpdated(database) {
				response = resp.NewNullArray()
				connState.EndTransaction()
			} else {

				err, queue := connState.Exec(database)
				fmt.Println(queue)
				if err != nil {
					response = resp.SimpleError{Data: err.Error()}
				} else {

					var responses []resp.RESPValue
					for _, cmd := range queue {
						r := GenerateResponse(cmd, database, connState)
						responses = append(responses, r)
					}
					response = resp.Array{Data: responses}
				}
			}

		}
	case "DISCARD":
		{
			if err := connState.Discard(); err != nil {
				response = resp.SimpleError{Data: err.Error()}
			} else {
				response = resp.SimpleString{Data: []byte("OK")}
			}
		}
	case "WATCH":
		{
			if err := connState.Watch(command); err != nil {
				response = resp.SimpleError{Data: err.Error()}
			} else {
				response = resp.SimpleString{Data: []byte("OK")}
			}
		}
	case "UNWATCH":
		{
			connState.Unwatch()
			response = resp.SimpleString{Data: []byte("OK")}
		}

	default:
		response = resp.SimpleError{Data: fmt.Sprintf("Unknown command: %s", command.Name)}

	}
	fmt.Println(response)
	return
}
