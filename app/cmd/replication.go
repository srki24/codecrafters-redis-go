package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/db"
	"github.com/codecrafters-io/redis-starter-go/app/repl"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

func Info(command Command, database *db.Database) string {
	args := command.Args

	if len(args) < 1 {
		return "role:unknown no args"
	}
	key := args[0]
	if key == "replication" {
		replica := database.GetReplica()
		return replica.String()
	}
	return fmt.Sprintf("role:unknown key %s", key)

}

func replconf(command Command, database *db.Database, cs *ConnStatus, hub *repl.Hub) ([]string, error) {
	args := command.Args
	if len(args) < 2 {
		return nil, errors.New("not enought args")
	}
	if strings.ToUpper(args[0]) == "LISTENING-PORT" {
		port, err := strconv.Atoi(args[1])
		if err != nil {
			return nil, err
		}
		cs.SetPort(port)
		return []string{"OK"}, nil
	}
	if strings.ToUpper(args[0]) == "CAPA" {
		return []string{"OK"}, nil
	}
	if strings.ToUpper(args[0]) == "GETACK" {
		return []string{
			"REPLCONF",
			"ACK",
			strconv.Itoa(database.GetReplica().GetOffset()),
		}, nil
	}
	if strings.ToUpper(args[0]) == "ACK" {
		ack, err := strconv.Atoi(args[1])
		if err != nil {
			return nil, err
		}
		hub.SetAck(cs.GetPort(), ack)
		return nil, nil

	}
	return nil, fmt.Errorf("unknown args: %s", args)

}

func wait(command Command, database *db.Database, rh *repl.Hub) (int, error) {
	args := command.Args
	if len(args) < 2 {
		return 0, errors.New("not enought args")
	}

	timeoutMs, err := strconv.Atoi(args[1])
	if err != nil {
		return 0, err
	}

	requiredNr, err := strconv.Atoi(args[1])
	if err != nil {
		return 0, err
	}

	if database.GetReplica().GetOffset() == 0 {
		return rh.NumberOfReplica(), nil
	}
	start := time.Now()

	curreentOffset := database.GetReplica().GetOffset()
	cmd := resp.NewArray([]string{"REPLCONF", "GETACK", "*"})
	rh.Publish(cmd)
	for {
		cnt := 0
		for _, v := range rh.GetAck() {
			if v >= curreentOffset {
				cnt += 1
			}
		}
		if cnt >= requiredNr {
			return cnt, nil
		}

		end := time.Now()
		if end.Sub(start).Milliseconds() > int64(timeoutMs) {
			return cnt, nil
		}
		time.Sleep(100 * time.Microsecond)
	}
}
