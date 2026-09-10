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

func parseBulkString(data []byte) (BulkString, int) {
	lenIdx := bytes.Index(data, []byte("\r\n"))

	dataLen, _ := strconv.Atoi(string(data[:lenIdx]))

	if dataLen == -1 {
		return BulkString{}, lenIdx + 2
	}
	strStart := lenIdx + 2
	strEnd := strStart + dataLen
	return BulkString{Data: data[strStart:strEnd]}, strEnd + 2

}

func parseArray(data []byte) (Array, int) {
	lenIdx := bytes.Index(data, []byte("\r\n"))

	var arrData []RESPValue
	nrElements, _ := strconv.Atoi(string(data[:lenIdx]))

	if nrElements == -1 {
		return Array{}, lenIdx + 2
	}
	consumeIdx := lenIdx + 2
	for range nrElements {
		dp, consumed := Parse(data[consumeIdx:])
		arrData = append(arrData, dp)
		consumeIdx += consumed + 1
	}

	return Array{Data: arrData}, consumeIdx
}

func Parse(data []byte) (out RESPValue, consumed int) {
	switch data[0] {
	case '+':
		out, consumed = parseSimpleString(data[1:])
	case '$':
		out, consumed = parseBulkString(data[1:])
	case '*':
		out, consumed = parseArray(data[1:])
	default:
		panic(fmt.Sprintf("Unknown prefix: %s", string(data[0])))
	}
	return out, consumed
}
