package db

import (
	"time"
)

type Data struct {
	Value dbValue
	Time  time.Time
	Exp   int
}

type Database map[string]Data

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
	db := make(Database, 0)
	return db
}
