package cmd

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/db"
)

func xadd(cmd Command, database *db.Database) (string, error) {
	args := cmd.Args

	if len(args) < 4 {
		return "", errors.New("Err Failed to get add stream, not enough args")
	}

	key := args[0]
	id := args[1]

	if id == "0-0" {
		return "", errors.New("ERR The ID specified in XADD must be greater than 0-0")
	}

	parts := strings.Split(id, "-")

	if len(parts) != 2 {
		return "", fmt.Errorf("ERR Invalid entry. Should be in millisecondsTime-sequenceNumber format, got %s", id)
	}
	msTime, err := strconv.Atoi(parts[0])

	if err != nil {
		return "", err
	}
	seqNr, err := strconv.Atoi(parts[1])

	if err != nil {
		return "", err
	}

	entryId := db.EntryId{MillisecondsTime: msTime, SequenceNumber: seqNr}

	streamMapping := make(map[string]string)

	for i := 2; i < len(args); i = i + 2 {
		k := args[i]
		v := args[i+1]
		streamMapping[k] = v
	}
	data, keyExists := database.Get(key)

	// create new stream
	if !keyExists {
		streamData := make(map[db.EntryId]map[string]string)
		streamData[entryId] = streamMapping

		stream := db.Stream{
			Data:             streamData,
			MillisecondsTime: entryId.MillisecondsTime,
			SequenceNumber:   entryId.SequenceNumber,
		}

		dbValue := db.Data{Value: stream, Time: time.Now(), Exp: -1}
		database.Set(key, dbValue)
		return id, nil

	}

	// Existing stream
	stream, isCorrectType := data.Value.(db.Stream)
	if !isCorrectType {
		return "", fmt.Errorf(
			"ERR Key exists but it's not a stream. It's of a type: %s", reflect.TypeOf(data.Value))
	}

	fmt.Println(stream.MillisecondsTime)
	fmt.Println(stream.SequenceNumber)

	fmt.Println(entryId.MillisecondsTime)
	fmt.Println(entryId.SequenceNumber)

	if (entryId.MillisecondsTime < stream.MillisecondsTime) ||
		((entryId.MillisecondsTime == stream.MillisecondsTime) &&
			(entryId.SequenceNumber <= stream.SequenceNumber)) {
		return "", errors.New("ERR The ID specified in XADD is equal or smaller than the target stream top item")
	}

	// OK
	stream.MillisecondsTime = entryId.MillisecondsTime
	stream.SequenceNumber = entryId.SequenceNumber
	stream.Data[entryId] = streamMapping

	data.Value = stream
	database.Set(key, data)

	return id, nil
}
