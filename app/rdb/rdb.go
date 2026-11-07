package rdb

import (
	"encoding/base64"
	"fmt"
)

const EMPTY_FILE = "UkVESVMwMDEx+glyZWRpcy12ZXIFNy4yLjD6CnJlZGlzLWJpdHPAQPoFY3RpbWXCbQi8ZfoIdXNlZC1tZW3CsMQQAPoIYW9mLWJhc2XAAP/wbjv+wP9aog=="

func Serialize() ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(EMPTY_FILE)
	if err != nil {
		return nil, err
	}
	return fmt.Appendf(nil, "$%d\r\n%s", len(data), data), nil
}
