package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/db"
)

const maxDateMillis = "253402300799000"
const maxSeq = int(^uint(0) >> 1)

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

	entries := []db.EntryData{}

	for i := 2; i < len(args); i = i + 2 {
		entry := db.EntryData{Key: args[i], Value: args[i+1]}
		entries = append(entries, entry)
	}
	data, keyExists := database.Get(key)

	entryId, err := parseId(id, data.Value)

	if err != nil {
		return "", err
	}
	// create new stream
	if !keyExists {
		var streamData []db.StreamData
		streamData = append(streamData, db.StreamData{Id: entryId, Data: entries})

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
	stream.Data = append(stream.Data, db.StreamData{Id: entryId, Data: entries})

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

		if key == "-" && isStart {
			msTimeStr = "0"
			seqNrStr = "0"
		} else if key == "+" && !isStart {
			msTimeStr = maxDateMillis
		} else { // just key provided
			msTimeStr = key

			if isStart {
				seqNrStr = "0"
			} else {
				seqNrStr = strconv.Itoa(maxSeq)
			}

		}

	}

	return fmt.Sprintf("%s-%s", msTimeStr, seqNrStr)

}

func Xrange(cmd Command, database *db.Database) (out []db.StreamData, err error) {

	args := cmd.Args
	if len(args) != 3 {
		err = errors.New("Err XRABGE Failed to get stream, not enough args")
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

	for _, streamData := range stream.Data {
		if streamData.Id.String() >= fromId && streamData.Id.String() <= toId {

			out = append(out, streamData)
		}
	}
	return out, err
}

func Xread(cmd Command, database *db.Database) (keys []string, out [][]db.StreamData, err error) {
	args := cmd.Args

	if len(args) < 3 {
		err = errors.New("Err XREAD Failed to get stream, not enough args")
		return
	}

	timeout := 0
	xrangeArgs := [][]string{}
	for i := 0; i < len(args); {

		arg := strings.ToUpper(args[i])
		switch arg {
		case "BLOCK":
			timeout, err = strconv.Atoi(args[i+1])
			if err != nil {
				return
			}
			i = i + 2
		case "STREAM":
			streamKey := args[i+1]
			fromId := args[i+2]
			toId := "+"
			xrangeArgs = append(xrangeArgs, []string{streamKey, fromId, toId})
			i = i + 3
		case "STREAMS":
			i++
			halfArgs := (len(args) - i) / 2
			for j := i; j < i+halfArgs; j++ {
				streamKey := args[j]
				fromId := args[j+halfArgs] + "0"
				toId := "+"
				xrangeArgs = append(xrangeArgs, []string{streamKey, fromId, toId})
			}
			i = len(args)
		default:
			err = fmt.Errorf("Unknown argument: %s", arg)
			return
		}

	}
	start := time.Now()

	for _, xrangeArg := range xrangeArgs {
		newCmd := Command{"XRANGE", xrangeArg}
	blocking:
		for {

			streamOut, err := Xrange(newCmd, database)
			if err != nil {
				return keys, out, err
			}

			if len(streamOut) == 0 {
				fmt.Println("Nout found blocking...")
				end := time.Now()
				fmt.Println(end.Sub(start).Milliseconds())
				time.Sleep(time.Second / 2)
				if (timeout != 0) && int(end.Sub(start).Milliseconds()) < timeout {
					continue blocking
				}
				fmt.Println("Timeout, data not found...")
			}
			out = append(out, streamOut)
			keys = append(keys, xrangeArg[0])
			break blocking
		}

	}
	fmt.Println(out)
	if len(out) != len(keys) {
		err = errors.New("Err XREAD, number of keys and outputs missmatch")
	}
	return keys, out, err
}
