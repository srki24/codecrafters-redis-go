package cmd

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/db"
)

func xadd(cmd Command, database *db.Database) (string, error) {
	args := cmd.Args

	if len(args) < 4 {
		return "", errors.New("Failed to get add stream, not enough args")
	}

	key := args[0]
	entryId := args[1]
	streamMapping := make(map[string]string)

	for i := 2; i < len(args); i = i + 2 {
		k := args[i]
		v := args[i+1]
		streamMapping[k] = v
	}
	data, keyExists := database.Get(key)

	if !keyExists {
		streamData := make(map[db.EntryId]map[string]string)
		streamData[db.EntryId(entryId)] = streamMapping

		dbValue := db.Data{Value: db.Stream{Data: streamData}, Time: time.Now(), Exp: -1}
		database.Set(key, dbValue)
		return entryId, nil

	}

	stream, isCorrectType := data.Value.(db.Stream)
	if !isCorrectType {
		return "", fmt.Errorf("Key exists but it's not a stream. It's of a type: %s", reflect.TypeOf(data.Value))
	}

	entity, entityExists := stream.Data[db.EntryId(entryId)]
	if entityExists {
		for k, v := range streamMapping {
			entity[k] = v
		}
	} else {
		stream.Data[db.EntryId(entryId)] = streamMapping
	}

	return entryId, nil
}
