package db

import (
	"fmt"
	"sync"
	"time"
)

// Data field
type Data struct {
	Value       DbValue
	CreatedTime time.Time
	UpdatedTime time.Time
	Exp         int
}

// Database
type Database struct {
	Data map[string]Data
	Mu   sync.Mutex
}

func (db *Database) Get(key string) (Data, bool) {
	db.Mu.Lock()
	defer db.Mu.Unlock()

	data, ok := db.Data[key]
	return data, ok
}
func NewDatabase() Database {
	data := make(map[string]Data)
	return Database{Data: data}
}

func (db *Database) Set(key string, value Data) {
	db.Mu.Lock()
	defer db.Mu.Unlock()
	value.UpdatedTime = time.Now()
	db.Data[key] = value

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
