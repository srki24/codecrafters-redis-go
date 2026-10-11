package cmd

import (
	"slices"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/db"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

var endTransactionCommands = []string{"EXEC", "DISCARD"}

type ConnStatus struct {
	queue       []Command
	transaction bool
}

func NewConnStatus() ConnStatus {
	return ConnStatus{transaction: false}
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

func (cs *ConnStatus) Commit(database *db.Database) (response resp.RESPValue) {
	var responses []resp.RESPValue
	if !cs.IsTransaction() {
		response = resp.SimpleError{Data: "ERR EXEC without MULTI"}
	} else {
		cs.transaction = false
		for _, cmd := range cs.queue {
			r := GenerateResponse(cmd, database, cs)
			responses = append(responses, r)
		}
		response = resp.Array{Data: responses}
		clear(cs.queue)
	}
	return response
}

func (cs *ConnStatus) Discard() (response resp.RESPValue) {
	if !cs.IsTransaction() {
		response = resp.SimpleError{Data: "ERR DISCARD without MULTI"}
	} else {
		response = resp.SimpleString{Data: []byte("OK")}
		cs.transaction = false
		clear(cs.queue)
	}
	return response
}
