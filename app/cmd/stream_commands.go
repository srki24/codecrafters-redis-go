package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/db"
)

func parseParts(id string, streamMs int, streamSeq int) (msTime, seqNr int, err error) {
	parts := strings.Split(id, "-")
	if len(parts) != 2 {
		return msTime, seqNr, fmt.Errorf("ERR Invalid entry. Should be in millisecondsTime-sequenceNumber format, got %s", id)
	}

	msTime, err = strconv.Atoi(parts[0])

	if parts[1] == "*" {
		if msTime == streamMs {
			seqNr = streamSeq + 1
		}
	} else {
		seqNr, err = strconv.Atoi(parts[1])
	}

	if (msTime < streamMs) || (msTime == streamMs) && (seqNr <= streamSeq) {
		err = errors.New("ERR The ID specified in XADD is equal or smaller than the target stream top item")
	}
	return msTime, seqNr, err
}

func parseId(id string, value db.DbValue) (entryId db.EntryId, err error) {
	if value == nil {
		// New stream defaults to
		value = db.Stream{}
	}

	stream, isCorrectType := value.(db.Stream)
	if !isCorrectType {
		return entryId, fmt.Errorf(
			"ERR Key exists but it's not a stream. It's of a type: %T", value)

	}

	if id == "0-0" {
		return entryId, errors.New("ERR The ID specified in XADD must be greater than 0-0")
	}

	if id == "*" {
		entryId.MillisecondsTime = int(time.Now().UnixMilli())
		if entryId.MillisecondsTime <= stream.MillisecondsTime {
			entryId.SequenceNumber = stream.SequenceNumber + 1
		}
	} else {
		entryId.MillisecondsTime, entryId.SequenceNumber, err = parseParts(id, stream.MillisecondsTime, stream.SequenceNumber)

	}

	return entryId, err
}

func xadd(cmd Command, database *db.Database) (string, error) {
	args := cmd.Args

	if len(args) < 4 {
		return "", errors.New("Err Failed to get add stream, not enough args")
	}

	key := args[0]
	id := args[1]

	streamMapping := make(map[string]string)

	for i := 2; i < len(args); i = i + 2 {
		k := args[i]
		v := args[i+1]
		streamMapping[k] = v
	}
	data, keyExists := database.Get(key)

	entryId, err := parseId(id, data.Value)

	if err != nil {
		return "", err
	}
	// create new stream
	if !keyExists {
		streamData := make(map[db.EntryId]map[string]string)
		streamData[entryId] = streamMapping

		stream := db.Stream{
			Data:             streamData,
			MillisecondsTime: entryId.MillisecondsTime,
			SequenceNumber:   entryId.SequenceNumber,
		}

		data := db.Data{
			Value: stream,
			Time:  time.Now(),
			Exp:   -1,
		}

		database.Set(key, data)
		return entryId.Id(), nil

	}

	// Existing stream
	stream := data.Value.(db.Stream)

	stream.MillisecondsTime = entryId.MillisecondsTime
	stream.SequenceNumber = entryId.SequenceNumber
	stream.Data[entryId] = streamMapping

	data.Value = stream
	database.Set(key, data)

	return entryId.Id(), nil
}
