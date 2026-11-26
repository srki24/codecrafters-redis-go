package cmd

import (
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/db"
)

var endTransactionCommands = []string{"EXEC", "DISCARD", "WATCH"}

type ConnStatus struct {
	queue        []Command
	transaction  bool
	watching     map[string]time.Time
	isReplica    bool
	isReplicated bool
	replicaPort  int
	subscribed   []string
	outbound     chan []byte
}

func NewConnStatus() ConnStatus {
	watching := make(map[string]time.Time)
	return ConnStatus{
		transaction:  false,
		watching:     watching,
		isReplica:    false,
		isReplicated: false,
		replicaPort:  -1,
		subscribed:   []string{},
		outbound:     nil,
	}
}

func (cs *ConnStatus) Outbound() chan []byte {
	if cs.outbound == nil {
		ch := make(chan []byte, 16)
		cs.outbound = ch
	}
	return cs.outbound

}
func (cs *ConnStatus) Subscribe(sub string) {
	cs.subscribed = append(cs.subscribed, sub)
}

func (cs *ConnStatus) Unsubscribe(sub string) {
	cs.subscribed = slices.DeleteFunc(cs.subscribed, func(c string) bool { return c == sub })
}

func (cs *ConnStatus) IsSub() bool {
	return cs.NumberOfSubscriptions() >= 1
}

func (cs *ConnStatus) NumberOfSubscriptions() int {
	return len(cs.subscribed)
}

func (cs *ConnStatus) QueueCommand(command Command) bool {
	endTransaction := slices.Contains(endTransactionCommands, strings.ToUpper(command.Name))
	shouldQueue := cs.IsTransaction() && !endTransaction
	if shouldQueue {
		cs.queue = append(cs.queue, command)
	}

	return shouldQueue
}

func (cs *ConnStatus) IsReplica() bool {
	return cs.isReplica
}

func (cs *ConnStatus) IsReplicated() bool {
	return cs.isReplicated
}

func (cs *ConnStatus) MarkAsReplica() {
	cs.isReplica = true
}

func (cs *ConnStatus) ShouldRepeplicate() bool {
	return cs.isReplica && !cs.isReplicated
}

func (cs *ConnStatus) MarkReplicated() {
	cs.isReplicated = true
}

func (cs *ConnStatus) IsTransaction() bool {
	return cs.transaction
}

func (cs *ConnStatus) BeginTransaction() {
	cs.transaction = true
}

func (cs *ConnStatus) ShouldAbort(database *db.Database) bool {
	if len(cs.watching) < 1 {
		return false
	}
	for key, seen := range cs.watching {
		data, exists := database.Get(key)
		if !exists {
			if seen.IsZero() {
				continue // doesn't exist, didn't exist when watch was alled
			} else {
				return true
			}
		}
		if data.UpdatedTime.After(seen) {
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

func (cs *ConnStatus) Watch(cmd Command, database *db.Database) error {
	if cs.IsTransaction() {
		return errors.New("ERR WATCH inside MULTI is not allowed")
	}
	args := cmd.Args
	if len(args) < 1 {
		return errors.New("Couldnt watch, key not provided")
	}
	for _, key := range args {
		data, exists := database.Get(key)
		if !exists {
			cs.watching[key] = time.Time{}
		} else {
			cs.watching[key] = data.UpdatedTime

		}
	}
	return nil
}

func (cs *ConnStatus) Unwatch() {
	clear(cs.watching)
}

func (cs *ConnStatus) SetPort(port int) {
	cs.replicaPort = port
}

func (cs *ConnStatus) GetPort() int {
	return cs.replicaPort
}
