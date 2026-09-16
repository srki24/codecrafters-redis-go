package resp

import (
	"bytes"
	"fmt"
	"slices"
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
	data = parseQuotes(data)
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

func parseQuotes(input []byte) []byte {
	var quoteChars []byte = []byte{'\'', '"'}
	var spaces []byte = []byte{' '}
	var quoteStack = []byte{}

	var out []byte = []byte{}

	var firstSpace = false

	for _, c := range input {

		// Deal with open/ close quotes
		if slices.Contains(quoteChars, c) {
			if len(quoteStack) > 0 && quoteStack[len(quoteStack)-1] == c {
				quoteStack = quoteStack[:len(quoteStack)-1]
			} else {
				quoteStack = append(quoteStack, c)
				continue
			}
		}
		// outside of quotes
		if len(quoteStack) == 0 {
			if slices.Contains(spaces, c) {
				if !firstSpace {
					firstSpace = true
					out = append(out, ' ')
					continue
				} else {
					// already has space
					continue
				}
			} else {
				// not a space char
				firstSpace = false
				out = append(out, c)
				continue

			}
		}
		// inside of quotes
		out = append(out, c)

	}
	return out
}
