package resp

import (
	"bytes"
	"fmt"
	"strconv"
)

func parseSimpleString(data []byte) (SimpleString, int) {
	lenIdx := bytes.Index(data, []byte("\r\n"))
	return SimpleString{Data: data[:lenIdx]}, lenIdx + 2
}

func parseBulkString(data []byte) (BulkString, int, error) {
	lenIdx := bytes.Index(data, []byte("\r\n"))

	dataLen, err := strconv.Atoi(string(data[:lenIdx]))

	if err != nil {
		return BulkString{}, 0, err
	}

	if dataLen == -1 {
		return BulkString{}, lenIdx + 2, nil
	}
	strStart := lenIdx + 2
	strEnd := strStart + dataLen
	return BulkString{Data: data[strStart:strEnd]}, strEnd + 2, nil

}

func parseArray(data []byte) (Array, int, error) {

	lenIdx := bytes.Index(data, []byte("\r\n"))
	var arrData []RESPValue
	nrElements, _ := strconv.Atoi(string(data[:lenIdx]))

	if nrElements == -1 {
		return Array{}, lenIdx + 2, nil
	}
	consumeIdx := lenIdx + 2
	for range nrElements {
		dp, consumed, err := Parse(data[consumeIdx:])

		if err != nil {
			return Array{}, 0, err
		}
		arrData = append(arrData, dp)
		consumeIdx += consumed + 1
	}

	return Array{Data: arrData}, consumeIdx, nil
}

func Parse(data []byte) (out RESPValue, consumed int, err error) {
	switch data[0] {
	case '+':
		out, consumed = parseSimpleString(data[1:])
	case '$':
		out, consumed, err = parseBulkString(data[1:])
	case '*':
		out, consumed, err = parseArray(data[1:])
	default:
		err = fmt.Errorf("Unknown prefix: %s", string(data[0]))
	}
	return out, consumed, err
}
