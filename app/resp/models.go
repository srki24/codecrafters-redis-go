package resp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/db"
)

type RESPValue interface {
	Serialize() []byte
	fmt.Stringer
}

type SimpleString struct {
	Data []byte
}

func (v SimpleString) Serialize() []byte {
	return fmt.Appendf(nil, "+%s\r\n", v.Data)
}

func (v SimpleString) String() string {
	return string(v.Data)
}

type BulkString struct {
	Data []byte
}

func (v BulkString) String() string {
	return string(v.Data)
}

func (v BulkString) Serialize() []byte {
	if v.Data == nil {
		return fmt.Appendf(nil, "$-1\r\n")
	}
	return fmt.Appendf(nil, "$%d\r\n%s\r\n", len(v.Data), v.Data)
}

type Array struct {
	Data   []RESPValue
	IsNull bool
}

func (v Array) Serialize() []byte {

	if v.IsNull {
		return []byte("*-1\r\n")
	}

	out := fmt.Appendf(nil, "*%d\r\n", len(v.Data))

	for _, respV := range v.Data {
		out = append(out, respV.Serialize()...)
	}
	return out
}

func (v Array) String() string {
	var data []string
	for _, rv := range v.Data {
		data = append(data, rv.String())
	}
	return strings.Join(data, ",")
}

func NewArray(data []string) Array {
	arr := Array{}

	for _, dp := range data {
		arr.Data = append(arr.Data, BulkString{[]byte(dp)})
	}
	return arr
}

func NewRespArray(data ...RESPValue) Array {
	return Array{Data: data}

}
func NewNullArray() Array {
	return Array{IsNull: true}
}

func NewArrayFromStream(stream db.StreamData) Array {
	respEntries := []string{}

	entryId := BulkString{Data: []byte(stream.Id.String())}
	for _, e := range stream.Data {
		respEntries = append(respEntries, e.Key, e.Value)

	}
	entryValue := NewArray(respEntries)
	entry := Array{Data: []RESPValue{entryId, entryValue}}
	return entry
}

// func NewArrayFromStream(data []db.StreamData) Array {
// 	respEntries := []RESPValue{}

// 	for _, entry := range data {
// 		entryId := BulkString{Data: []byte(entry.Id.String())}
// 		for _, e := range entry.Data {
// 			entryValue := NewArray([]string{e.Key, e.Value})
// 			entry := Array{Data: []RESPValue{entryId, entryValue}}
// 			respEntries = append(respEntries, entry)

// 		}
// 	}
// 	return Array{Data: respEntries}
// }

type Integer struct {
	Data int
}

func (v Integer) Serialize() []byte {
	return fmt.Appendf(nil, ":%d\r\n", v.Data)

}

func (v Integer) String() string {
	return strconv.Itoa(v.Data)
}

type SimpleError struct {
	Data string
}

func (v SimpleError) Serialize() []byte {
	return fmt.Appendf(nil, "-%s\r\n", v.Data)
}

func (v SimpleError) String() string {
	return v.Data
}
