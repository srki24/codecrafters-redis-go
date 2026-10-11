package cmd

import (
	"errors"
	"fmt"
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
	watching := make(map[string]time.Time, 0)
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
		fmt.Println(key)

		data, ok := database.Get(key)
		if !ok {
			continue // arg doesn't exist in the database
		}

		fmt.Println(data.UpdatedTime)
		fmt.Println(watchSetTime)
		fmt.Println(data.UpdatedTime.Compare(watchSetTime))
		if data.UpdatedTime.Compare(watchSetTime) == 1 {
			return true // updated
		}

	}
	return false
}

func (cs *ConnStatus) EndTransaction() {
	cs.transaction = false
	clear(cs.queue)
}
func (cs *ConnStatus) Commit(database *db.Database) (error, []Command) {
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
		cs.transaction = false
		clear(cs.queue)
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
	key := args[0]

	cs.watching[key] = time.Now()
	return nil
}
