package resp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
)

var ErrIncomplete = errors.New("incomplete resp value")

func parseSimpleString(data []byte) (SimpleString, int, error) {
	// +<data>\r\n
	lenIdx := bytes.Index(data, []byte("\r\n"))
	if lenIdx < 0 {
		return SimpleString{}, 0, ErrIncomplete
	}
	return SimpleString{Data: data[:lenIdx]}, lenIdx + 2, nil
}

func parseBulkString(data []byte) (BulkString, int, error) {
	// $<length>\r\n<data>\r\n
	lenIdx := bytes.Index(data, []byte("\r\n"))
	if lenIdx < 0 {
		return BulkString{}, 0, ErrIncomplete
	}
	dataLen, err := strconv.Atoi(string(data[:lenIdx]))

	if err != nil {
		return BulkString{}, 0, err
	}

	if dataLen == -1 {
		return BulkString{}, lenIdx + 2, nil
	}
	strStart := lenIdx + 2 // skip /r/n
	strEnd := strStart + dataLen

	if strEnd+2 > len(data) {
		return BulkString{}, 0, ErrIncomplete
	}
	return BulkString{Data: data[strStart:strEnd]}, strEnd + 2, nil

}

func parseArray(data []byte) (Array, int, error) {
	// *<number-of-elements>\r\n<element-1>...<element-n>
	lenIdx := bytes.Index(data, []byte("\r\n"))
	if lenIdx < 0 {
		return Array{}, 0, ErrIncomplete
	}

	var items []RESPValue
	nrElements, _ := strconv.Atoi(string(data[:lenIdx]))

	if nrElements == -1 {
		return Array{}, lenIdx + 2, nil
	}

	consumeIdx := lenIdx + 2
	for range nrElements {
		dp, consumed, err := parse(data[consumeIdx:])

		if err != nil {
			return Array{}, 0, err
		}
		items = append(items, dp)
		consumeIdx += consumed
	}

	return Array{Data: items}, consumeIdx, nil
}

func parse(data []byte) (out RESPValue, consumed int, err error) {
	if len(data) == 0 {
		return nil, 0, ErrIncomplete
	}
	switch data[0] {
	case '+':
		out, consumed, err = parseSimpleString(data[1:])
	case '$':
		out, consumed, err = parseBulkString(data[1:])
	case '*':
		out, consumed, err = parseArray(data[1:])
	default:
		err = fmt.Errorf("Unknown prefix: %s", string(data[0]))
	}
	if err != nil {
		return nil, 0, err
	}
	consumed++ // the type byte
	return out, consumed, nil

}

func ParseRequest(reader io.Reader) chan RESPValue {
	out := make(chan RESPValue)
	go func() {
		defer close(out)
		var pending []byte
		tmp := make([]byte, 4096)

		for {
			n, err := reader.Read(tmp)
			if err != nil {
				return
			}
			if n > 0 {
				pending = append(pending, tmp[:n]...)
			}
			for len(pending) > 0 {
				request, consumed, perr := parse(pending)
				if errors.Is(perr, ErrIncomplete) {
					break // partial read
				}
				if perr != nil {
					return
				}
				out <- request
				pending = pending[consumed:]
			}
		}
	}()
	return out
}
