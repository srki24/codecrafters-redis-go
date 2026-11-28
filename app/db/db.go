package db

import (
	"errors"
	"fmt"
	"math"
	"path"
	"sort"
	"sync"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/config"
)

type Data struct {
	Value       DbValue
	CreatedTime time.Time
	UpdatedTime time.Time
	ExpMS       int
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
		ExpMS:       *exp,
	}
}

func (d Data) Expired() bool {
	if d.ExpMS <= 0 {
		return false
	}
	cTime := time.Now()
	elapsed := int(cTime.Sub(d.CreatedTime).Milliseconds())
	return elapsed > d.ExpMS
}

// Database
type Database struct {
	data    map[string]Data
	mu      sync.RWMutex
	replica Replica
	cfg     *config.Config
}

func (db *Database) get(key string) (Data, bool) {
	data, ok := db.data[key]

	expired := data.Expired()
	if expired {
		delete(db.data, key)
		data = Data{}
	}
	return data, ok && !expired
}

func (db *Database) Get(key string) (Data, bool) {
	db.mu.Lock() //rw lock, get deletes expired key
	defer db.mu.Unlock()
	return db.get(key)
}

func (db *Database) GetKeys(pattern string) ([]string, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	var out []string

	for key := range db.data {
		if matched, err := path.Match(pattern, key); matched {
			if err != nil {
				return out, err
			}
			out = append(out, key)
		}

	}
	return out, nil
}

func (db *Database) Set(key string, value Data) {
	db.mu.Lock()
	defer db.mu.Unlock()
	value.UpdatedTime = time.Now()
	db.data[key] = value

}

func (db *Database) GetReplica() Replica {
	return db.replica
}

func (db *Database) IsMaster() bool {
	return db.replica.role == "master"
}

func (db *Database) AddReplicaOffset(request []byte) {
	db.mu.Lock()
	defer db.mu.Unlock()
	n := len(request)
	db.replica.AddOffset(n)
}

func NewDatabase(config *config.Config) Database {
	data := make(map[string]Data)
	replica := newReplica(config)

	return Database{data: data, replica: replica, cfg: config}
}

func (db *Database) GetConfig() *config.Config {
	return db.cfg
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
	if !exists {
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

func (self EntryId) Greater(other EntryId) bool {
	if self.MillisecondsTime > other.MillisecondsTime {
		return true
	}
	if self.MillisecondsTime == other.MillisecondsTime {
		return self.SequenceNumber > other.SequenceNumber
	}
	return false
}

func (self EntryId) Equal(other EntryId) bool {
	return self.MillisecondsTime == other.MillisecondsTime && self.SequenceNumber == other.SequenceNumber
}

func (self EntryId) Less(other EntryId) bool {
	return !self.Greater(other) && !self.Equal(other)
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

// sorted set

type SortedMember struct {
	Key   string
	Value float64
}

type SortedSet struct {
	items []SortedMember
}

func (ss *SortedSet) Index(key string) int {
	for idx, e := range ss.items {
		if e.Key == key {
			return idx
		}
	}
	return -1
}

func (ss *SortedSet) Add(key string, value float64) {
	if idx := ss.Index(key); idx >= 0 {
		ss.items[idx].Value = value
		return
	}
	newMemeber := SortedMember{key, value}
	ss.AddElement(newMemeber)
}

func (ss *SortedSet) AddElement(element SortedMember) {
	tolerance := 0.0000000001

	ss.items = append(ss.items, element)
	sort.Slice(ss.items, func(i, j int) bool {
		e1 := ss.items[i]
		e2 := ss.items[j]

		if math.Abs(e1.Value-e2.Value) < tolerance { // same value, return lexigraphically
			return e1.Key < e2.Key
		}
		return e1.Value < e2.Value
	})
}

func (ss *SortedSet) Remove(key string) {
	if idx := ss.Index(key); idx != -1 {
		ss.items = append(ss.items[:idx], ss.items[idx:]...)
	}
}

func (ss *SortedSet) Score(key string) (float64, bool) {
	if idx := ss.Index(key); idx != -1 {
		return ss.items[idx].Value, true
	}
	return 0, false
}

func (ss *SortedSet) Range(fromIdx, toIdx int) ([]string, error) {
	nrElements := len(ss.items)

	if fromIdx > nrElements {
		return nil, errors.New("index out of range")
	}
	if fromIdx < 0 {
		fromIdx = max(0, nrElements+fromIdx)
	}
	if toIdx < 0 {
		toIdx = max(0, nrElements+toIdx+1)
	} else {
		toIdx = min(toIdx+1, nrElements)
	}
	if toIdx < fromIdx {
		return nil, errors.New("to index must be greater then from index")
	}
	var out []string
	for _, e := range ss.items[fromIdx:toIdx] {
		out = append(out, e.Key)
	}
	return out, nil

}

func (ss SortedSet) NrElements() int {
	return len(ss.items)
}

func (ss SortedSet) GetType() string {
	return "sortedset"
}
