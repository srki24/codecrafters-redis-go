package db

import (
	"sync"
	"time"
)

type Data struct {
	Value dbValue
	Time  time.Time
	Exp   int
}

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

func (db *Database) Set(key string, value Data) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.data[key] = value

}

type dbValue interface {
	GetType() string
}

type StringType string

func (st StringType) GetType() string {
	return "string"
}

type ListType []string

func (lt ListType) GetType() string {
	return "list"
}

func InitializeDb() Database {
	data := make(map[string]Data, 0)
	return Database{data: data}
}
