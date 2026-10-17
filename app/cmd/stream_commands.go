package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/db"
)

const maxDateMillis = 253402300799000
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

	if len(args) < 4 || len(args)%2 != 0 {
		return "", errors.New("Err Failed to get add stream, not enough args")
	}

	key := args[0]
	id := args[1]

	fn := func(data db.Data, exists bool) (db.Data, string, error) {

		entries := []db.EntryData{}

		for i := 2; i < len(args); i = i + 2 {
			entry := db.EntryData{Key: args[i], Value: args[i+1]}
			entries = append(entries, entry)
		}

		entryId, err := parseId(id, data.Value)

		if err != nil {
			return data, "", err
		}
		// create new stream
		if !exists {
			var streamData []db.StreamData
			streamData = append(streamData, db.StreamData{Id: entryId, Data: entries})

			stream := db.Stream{
				Data:        streamData,
				LatestMs:    entryId.MillisecondsTime,
				LatestSeqNr: entryId.SequenceNumber,
			}

			data := db.Data{
				Value:       stream,
				CreatedTime: time.Now(),
				Exp:         -1,
			}
			return data, entryId.String(), nil
		}

		// Existing stream
		stream := data.Value.(db.Stream)

		stream.LatestMs = entryId.MillisecondsTime
		stream.LatestSeqNr = entryId.SequenceNumber
		stream.Data = append(stream.Data, db.StreamData{Id: entryId, Data: entries})

		data.Value = stream

		return data, entryId.String(), nil
	}

	out, err := db.Update(database, key, fn)
	return out, err
}

func parseKey(key string, isStart bool) (db.EntryId, error) {
	var msTimeStr, seqNrStr string
	if key == "-" && isStart {
		msTimeStr = "0"
		seqNrStr = "0"
	} else if key == "+" && !isStart {
		msTimeStr = strconv.Itoa(maxDateMillis)
		seqNrStr = strconv.Itoa(maxSeq)
	} else if strings.Contains(key, "-") && len(key) >= 3 { // ms-seq format
		parts := strings.Split(key, "-")
		msTimeStr = parts[0]
		seqNrStr = parts[1]
	} else { // just key provided
		msTimeStr = key
		if isStart {
			seqNrStr = "0"
		} else {
			seqNrStr = strconv.Itoa(maxSeq)
		}
	}

	msTime, err := strconv.Atoi(msTimeStr)
	if err != nil {
		return db.EntryId{}, err
	}
	seqNr, err := strconv.Atoi(seqNrStr)
	if err != nil {
		return db.EntryId{}, err
	}
	return db.EntryId{MillisecondsTime: msTime, SequenceNumber: seqNr}, nil
}

func Xrange(cmd Command, database *db.Database) (out []db.StreamData, err error) {

	args := cmd.Args
	if len(args) != 3 {
		err = errors.New("Err XRANGE Failed to get stream, not enough args")
		return
	}

	key := args[0]
	fromId, err := parseKey(args[1], true)
	if err != nil {
		return out, err
	}
	toId, err := parseKey(args[2], false)
	if err != nil {
		return out, err
	}
	data, keyExists := database.Get(key)
	if !keyExists {
		return out, ErrNotFound
	}

	stream, isStream := data.Value.(db.Stream)
	if !isStream {
		err = errors.New("Err key exist but it's not a stream")
		return out, err
	}

	for _, streamData := range stream.Data {
		greaterOrEqualFromId := streamData.Id.Greater(fromId) || streamData.Id.Equal(fromId)
		lessOrEqualToId := streamData.Id.Less(toId) || streamData.Id.Equal(toId)
		if greaterOrEqualFromId && lessOrEqualToId {
			out = append(out, streamData)
		}
	}
	return out, err
}

func xreadParseId(key string) (string, string) {
	toId := "+"
	fromId := key
	if key != "$" {
		parts := strings.Split(key, "-")
		ms, _ := strconv.Atoi(parts[0])
		seq, _ := strconv.Atoi(parts[1])
		fromId = fmt.Sprintf("%d-%d", ms, seq+1)
	}
	return fromId, toId

}
func Xread(cmd Command, database *db.Database) (keys []string, out [][]db.StreamData, err error) {
	args := cmd.Args
	if len(args) < 3 {
		err = errors.New("Err XREAD Failed to get stream, not enough args")
		return
	}
	start := time.Now()
	xrangeArgs, timeout, err := xreadParseArgs(args)
	if err != nil {
		return keys, out, err
	}
	commands, err := xReadParseCommands(xrangeArgs, database)
	if err != nil {
		return keys, out, err
	}
	for {
		progessed := false
		for _, newCmd := range commands {
			streamOut, err := Xrange(newCmd, database)
			if err != nil {
				return keys, out, err
			}

			if len(streamOut) > 0 {
				out = append(out, streamOut)
				keys = append(keys, newCmd.Args[0])
				progessed = true
			}
		}
		end := time.Now()
		timedOut := timeout > 0 && int(end.Sub(start).Milliseconds()) > timeout
		if progessed || timeout < 0 || timedOut {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	return keys, out, err
}

func xreadParseArgs(args []string) (xrangeArgs [][]string, timeout int, err error) {
	timeout = -1
	for i := 0; i < len(args); {
		arg := strings.ToUpper(args[i])
		switch arg {
		case "BLOCK":
			timeout, err = strconv.Atoi(args[i+1])
			if err != nil {
				return xrangeArgs, timeout, err
			}
			i = i + 2
		case "STREAM":
			streamKey := args[i+1]
			fromId, toId := xreadParseId(args[i+2])
			xrangeArgs = append(xrangeArgs, []string{streamKey, fromId, toId})
			i = i + 3
		case "STREAMS":
			i++
			halfArgs := (len(args) - i) / 2
			for j := i; j < i+halfArgs; j++ {
				streamKey := args[j]
				fromId, toId := xreadParseId(args[j+halfArgs])
				xrangeArgs = append(xrangeArgs, []string{streamKey, fromId, toId})
			}
			i = len(args)
		default:
			err = fmt.Errorf("Unknown argument: %s", arg)
			return xrangeArgs, timeout, err
		}

	}
	return xrangeArgs, timeout, err
}

func xReadParseCommands(xrangeArgs [][]string, database *db.Database) (commands []Command, err error) {
	for _, xrangeArg := range xrangeArgs {
		newCmd := Command{"XRANGE", xrangeArg}
		// special case need only newly inserted data need to get max existing key
		if newCmd.Args[1] == "$" {
			newCmd.Args[1] = "-" // read the whole stream in orer to get keys
			streamOut, err := Xrange(newCmd, database)
			if err != nil {
				if errors.Is(err, ErrNotFound) { // Non existing key
					newCmd.Args[1] = "0"
				} else {
					return commands, err
				}
			} else if len(streamOut) == 0 {
				newCmd.Args[1] = "0"
			} else {
				maxId := streamOut[len(streamOut)-1].Id
				newCmd.Args[1] = fmt.Sprintf("%d-%d", maxId.MillisecondsTime, maxId.SequenceNumber+1)
			}
		}
		commands = append(commands, newCmd)
	}
	return commands, nil
}
