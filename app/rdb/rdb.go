package rdb

import (
	"encoding/base64"
	"fmt"
)

const EMPTY_FILE = "UkVESVMwMDEx+glyZWRpcy12ZXIFNy4yLjD6CnJlZGlzLWJpdHPAQPoFY3RpbWXCbQi8ZfoIdXNlZC1tZW3CsMQQAPoIYW9mLWJhc2XAAP/wbjv+wP9aog=="

type RDB struct{}

func (rdb RDB) Serialize() []byte {
	data, _ := base64.StdEncoding.DecodeString(EMPTY_FILE)

	return fmt.Appendf(nil, "$%d\r\n%s", len(data), data)
}
