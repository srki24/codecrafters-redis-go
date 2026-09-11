package resp

import (
	"fmt"
	"strconv"
	"strings"
)

type RESPValue interface {
	Serialize() []byte
	GetStringData() string
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

func (v SimpleString) GetStringData() string {
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

func (v BulkString) GetStringData() string {
	return string(v.Data)
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

func (v Array) GetStringData() string {
	var data []string
	for _, rv := range v.Data {
		data = append(data, rv.GetStringData())
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
func NewNullArray() Array {
	return Array{IsNull: true}
}

type Integer struct {
	Data int
}

func (v Integer) Serialize() []byte {
	return fmt.Appendf(nil, ":%d\r\n", v.Data)

}

func (v Integer) GetStringData() string {
	return strconv.Itoa(v.Data)
}
