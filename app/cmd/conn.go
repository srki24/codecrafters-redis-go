package cmd

import (
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/db"
)

var endTransactionCommands = []string{"EXEC", "DISCARD", "WATCH"}
var canModifyKeyCommands = []string{"SET", "INCR"}

type ConnStatus struct {
	queue       []Command
	transaction bool
	watching    map[string]time.Time
}

func NewConnStatus() ConnStatus {
	watching := make(map[string]time.Time)
	return ConnStatus{transaction: false, watching: watching}
}

func (cs *ConnStatus) QueueCommand(command Command) bool {
	endTransaction := slices.Contains(endTransactionCommands, strings.ToUpper(command.Name))
	shouldQueue := cs.IsTransaction() && !endTransaction
	if shouldQueue {
		cs.queue = append(cs.queue, command)
	}

	return shouldQueue
}

func (cs *ConnStatus) IsTransaction() bool {
	return cs.transaction
}

func (cs *ConnStatus) BeginTransaction() {
	cs.transaction = true
}

func (cs *ConnStatus) IsWatchingKeyUpdated(database *db.Database) bool {

	if len(cs.watching) < 1 {
		return false
	}

	for key, watchSetTime := range cs.watching {

		data, ok := database.Get(key)
		if !ok {
			continue // arg doesn't exist in the database
		}

		if data.UpdatedTime.Compare(watchSetTime) == 1 {
			return true // updated
		}

	}
	return false
}

func (cs *ConnStatus) EndTransaction() {
	cs.transaction = false
	cs.watching = make(map[string]time.Time)

	cs.queue = nil
}
func (cs *ConnStatus) Exec(database *db.Database) (error, []Command) {
	if !cs.IsTransaction() {
		return errors.New("ERR EXEC without MULTI"), nil
	}

	queue := make([]Command, len(cs.queue))
	copy(queue, cs.queue)
	cs.EndTransaction()
	return nil, queue
}

func (cs *ConnStatus) Discard() error {
	if !cs.IsTransaction() {
		return errors.New("ERR DISCARD without MULTI")
	} else {
		cs.EndTransaction()
	}
	return nil
}

func (cs *ConnStatus) Watch(cmd Command) error {
	if cs.IsTransaction() {
		return errors.New("ERR WATCH inside MULTI is not allowed")
	}

	args := cmd.Args

	if len(args) < 1 {
		return errors.New("Couldnt watch, key not provided")
	}

	watchingTime := time.Now()
	for _, key := range args {
		cs.watching[key] = watchingTime

	}
	return nil
}

func (cs *ConnStatus) Unwatch() {
	clear(cs.watching)
}
