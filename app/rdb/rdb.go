package rdb

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/db"
)

const EMPTY_FILE = "UkVESVMwMDEx+glyZWRpcy12ZXIFNy4yLjD6CnJlZGlzLWJpdHPAQPoFY3RpbWXCbQi8ZfoIdXNlZC1tZW3CsMQQAPoIYW9mLWJhc2XAAP/wbjv+wP9aog=="

const (
	RDBEOF          = 0xFF
	RDBSELECTDB     = 0xFE
	RDBEXPIRETIME   = 0xFD
	RDBEXPIRETIMEMS = 0xFC
	RDBRESIZEDB     = 0xFB
	RDBAUX          = 0xFA
)

const (
	RDBSTRING = 0x00
)

type RDB struct{}

func (rdb RDB) Serialize() []byte {
	data, _ := base64.StdEncoding.DecodeString(EMPTY_FILE)
	return fmt.Appendf(nil, "$%d\r\n%s", len(data), data)
}

func Load(db *db.Database, dir, filename string) error {
	if dir == "" || filename == "" {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(dir, filename))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	return apply(db, data)
}

func calcExpFromTime(t time.Time) int {
	return int(time.Until(t).Milliseconds())
}

func apply(database *db.Database, data []byte) error {
	var exp int = -1
	i := 0
	if err := consumeHeader(&i, data); err != nil {
		return err
	}
	for i < len(data) {
		op := data[i]
		i++
		switch op {
		case RDBAUX:
			// key- value string metadta
			if _, err := readStr(&i, data); err != nil {
				return err
			}
			if _, err := readStr(&i, data); err != nil {
				return err
			}
		case RDBSELECTDB:
			// database selector, currently irrelevant (redis can have multiple databases)
			if _, _, err := readLen(&i, data); err != nil {
				return err
			}
		case RDBRESIZEDB:
			// size of the hash table that stores keys and values
			if _, _, err := readLen(&i, data); err != nil {
				return err
			}
			// size of the hash tble that stores the expires of the keys
			if _, _, err := readLen(&i, data); err != nil {
				return err
			}
		case RDBEXPIRETIME:
			t := time.Unix(int64(binary.LittleEndian.Uint64(data[i:i+4])), 0)
			i += 4
			exp = calcExpFromTime(t)

		case RDBEXPIRETIMEMS:
			t := time.UnixMilli(int64(binary.LittleEndian.Uint64(data[i : i+8])))
			i += 8
			exp = calcExpFromTime(t)
		case RDBSTRING:
			key, err := readStr(&i, data)
			if err != nil {
				return err
			}
			value, err := readStr(&i, data)
			if err != nil {
				return err
			}
			dbValue := db.StringType(value)
			fmt.Println(exp)
			if exp < -1 { // skip, already expired
				exp = -1
				continue
			}
			database.Set(key, db.NewData(dbValue, &exp))
			exp = -1
		case RDBEOF:
			return nil

		default:
			return fmt.Errorf("unknown op code: %02x at %d", op, i-1)
		}

	}
	return errors.New("rdb ended without EOF")
}

func consumeHeader(i *int, data []byte) error {
	*i += 9 // header length
	header := data[:9]
	if len(data) < *i {
		return errors.New("invalid header length")
	}

	if !slices.Equal(header[:5], []byte("REDIS")) {
		return fmt.Errorf("unknown prefix %s", string(data[:5]))
	}

	if _, err := strconv.Atoi(string(header[5:])); err != nil {
		return err
	}
	return nil
}

func readLen(i *int, data []byte) (int, bool, error) {
	lengthMask := byte(0b00111111)
	b := data[*i]
	*i++
	switch b >> 6 {
	case 0b00:
		// 00xxxxxx
		return int(b & lengthMask), false, nil
	case 0b01:
		// 00xxxxxx yyyyyy
		if *i > len(data) {
			return 0, false, errors.New("unexpected EOF")
		}
		next := data[*i]
		*i++
		return int(binary.BigEndian.Uint16([]byte{b & lengthMask, next})), false, nil
	case 0b10:
		// ignore first bit, read next 4
		len := int(binary.BigEndian.Uint16(data[*i : *i+4]))
		*i += 4
		return len, false, nil
	default:
		// next bytes are encoding
		return int(b & lengthMask), true, nil

	}

}

func readStr(i *int, data []byte) (string, error) {
	n, encoded, err := readLen(i, data)
	if err != nil {
		return "", err
	}
	if encoded {
		return readEncodsedStr(i, data, n)
	}

	if *i+n > len(data) {
		return "", errors.New("readStr unexpected EOF")
	}
	out := string(data[*i : *i+n])
	*i += n
	return out, nil
}

func readEncodsedStr(i *int, data []byte, encoding int) (string, error) {
	switch encoding {
	case 0:
		next := data[*i]
		*i++
		return strconv.Itoa(int(next)), nil
	case 1:
		nr := int(binary.BigEndian.Uint16(data[*i : *i+1]))
		*i += 2
		return strconv.Itoa(nr), nil
	case 2:
		// ignore first bit, read next 4
		nr := int(binary.BigEndian.Uint16(data[*i : *i+4]))
		*i += 4
		return strconv.Itoa(nr), nil
	default:
		return "", fmt.Errorf("unknown encoding: %d", encoding)

	}
}
