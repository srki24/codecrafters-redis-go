package db

import (
	"fmt"
	"sync"
	"time"
)

// Data field
type Data struct {
	Value DbValue
	Time  time.Time
	Exp   int
}

// Database
type Database struct {
	data map[string]Data
	mu   sync.Mutex
}

func (db *Database) Get(key string) (Data, bool) {
	db.mu.Lock()
	defer db.mu.Unlock()

	data, ok := db.data[key]
	return data, ok
}
func InitializeDb() Database {
	data := make(map[string]Data, 0)
	return Database{data: data}
}

func (db *Database) Set(key string, value Data) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.data[key] = value

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
	Id    EntryId
	Entry []EntryData
}

type Stream struct {
	Data        []StreamData
	LatestMs    int
	LatestSeqNr int
}

func (st Stream) GetType() string {
	return "stream"
}
