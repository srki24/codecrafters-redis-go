package db

import (
	"fmt"
	"sync"
	"time"
)

type Data struct {
	Value       DbValue
	CreatedTime time.Time
	UpdatedTime time.Time
	Exp         int
}

func NewData(value DbValue, exp *int) Data {
	if exp == nil {
		defaultExp := -1
		exp = &defaultExp
	}

	now := time.Now()
	return Data{Value: value,
		CreatedTime: now,
		UpdatedTime: now,
		Exp:         *exp,
	}
}

func (d Data) expired() bool {
	if d.Exp <= 0 {
		return false
	}
	cTime := time.Now()
	elapsed := int(cTime.Sub(d.CreatedTime).Milliseconds())
	return elapsed > d.Exp
}

// Database
type Database struct {
	data map[string]Data
	mu   sync.Mutex
}

func (db *Database) get(key string) (Data, bool) {
	data, ok := db.data[key]

	expired := data.expired()
	if expired {
		delete(db.data, key)
		data = Data{}
	}
	return data, ok && !expired
}

func (db *Database) Get(key string) (Data, bool) {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.get(key)
}

func NewDatabase() Database {
	data := make(map[string]Data)
	return Database{data: data}
}

func (db *Database) Set(key string, value Data) {
	db.mu.Lock()
	defer db.mu.Unlock()
	value.UpdatedTime = time.Now()
	db.data[key] = value

}

func Update[T any](
	database *Database,
	key string,
	fn func(curr Data, exists bool) (next Data, out T, err error)) (out T, err error) {
	database.mu.Lock()
	defer database.mu.Unlock()

	data, exists := database.get(key)

	next, out, err := fn(data, exists)

	if err != nil {
		return out, err
	}
	now := time.Now()
	if exists {
		next.CreatedTime = now
	}
	next.UpdatedTime = now
	database.data[key] = next
	return out, nil

}

// Db value interface
type DbValue interface {
	GetType() string
}

// String type
type StringType string

func (st StringType) GetType() string {
	return "string"
}

// List type
type ListType []string

func (lt ListType) GetType() string {
	return "list"
}

// Stream type

type EntryId struct {
	MillisecondsTime int
	SequenceNumber   int
}

func (id EntryId) String() string {
	return fmt.Sprintf("%d-%d", id.MillisecondsTime, id.SequenceNumber)
}

type EntryData struct {
	Key   string
	Value string
}

type StreamData struct {
	Id   EntryId
	Data []EntryData
}

type Stream struct {
	Data        []StreamData
	LatestMs    int
	LatestSeqNr int
}

func (st Stream) GetType() string {
	return "stream"
}
