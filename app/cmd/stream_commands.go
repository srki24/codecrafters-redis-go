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
		if entryId.MillisecondsTime <= stream.LatestMs {
			entryId.SequenceNumber = stream.LatestSeqNr + 1
		}
	} else {
		entryId.MillisecondsTime, entryId.SequenceNumber, err = parseParts(id, stream.LatestMs, stream.LatestSeqNr)

	}

	return entryId, err
}

func Xadd(cmd Command, database *db.Database) (string, error) {
	args := cmd.Args

	if len(args) < 4 {
		return "", errors.New("Err Failed to get add stream, not enough args")
	}

	key := args[0]
	id := args[1]

	entries := []db.Entry{}

	for i := 2; i < len(args); i = i + 2 {
		entry := db.Entry{Key: args[i], Value: args[i+1]}
		entries = append(entries, entry)
	}

	data, keyExists := database.Get(key)

	entryId, err := parseId(id, data.Value)

	if err != nil {
		return "", err
	}
	// create new stream
	if !keyExists {
		streamData := make(map[db.EntryId][]db.Entry)
		streamData[entryId] = entries

		stream := db.Stream{
			Data:        streamData,
			LatestMs:    entryId.MillisecondsTime,
			LatestSeqNr: entryId.SequenceNumber,
		}

		data := db.Data{
			Value: stream,
			Time:  time.Now(),
			Exp:   -1,
		}

		database.Set(key, data)
		return entryId.String(), nil

	}

	// Existing stream
	stream := data.Value.(db.Stream)

	stream.LatestMs = entryId.MillisecondsTime
	stream.LatestSeqNr = entryId.SequenceNumber
	stream.Data[entryId] = entries

	data.Value = stream
	database.Set(key, data)

	return entryId.String(), nil
}

func parseKey(key string, isStart bool) string {
	var msTimeStr, seqNrStr string
	parts := strings.Split(key, "-")
	if len(parts) == 2 {
		msTimeStr = parts[0]
		seqNrStr = parts[1]
	} else {
		msTimeStr = key

		if isStart {
			seqNrStr = "0"
		} else {
			seqNrStr = strconv.Itoa(^int(0))
		}
	}

	return fmt.Sprintf("%s-%s", msTimeStr, seqNrStr)

}
func Xrange(cmd Command, database *db.Database) (out []map[string][]string, err error) {

	args := cmd.Args
	if len(args) != 3 {
		return
	}

	key := args[0]
	fromId := parseKey(args[1], true)
	toId := parseKey(args[2], false)

	data, keyExists := database.Get(key)
	if !keyExists {
		err = errors.New("Err key doesn't exist")
		return out, err
	}

	stream, isStream := data.Value.(db.Stream)
	if !isStream {
		err = errors.New("Err key exist but it's not a stream")
		return out, err
	}

	for entryId, dbEntries := range stream.Data {

		if entryId.String() >= fromId && entryId.String() <= toId {
			outEntries := []string{}
			for _, dbEntry := range dbEntries {
				outEntries = append(outEntries, dbEntry.Key, dbEntry.Value)
			}

			out = append(out, map[string][]string{entryId.String(): outEntries})
		}
	}
	return out, err
}
