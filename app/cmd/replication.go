package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/db"
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

func replconf(command Command, database *db.Database) ([]string, error) {
	args := command.Args
	if len(args) < 2 {
		return nil, errors.New("not enought args")
	}
	if strings.ToUpper(args[0]) == "LISTENING-PORT" {
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
	return nil, fmt.Errorf("unknown args: %s", args)

}

func wait(command Command, database *db.Database) (int, error) {
	args := command.Args
	if len(args) < 2 {
		return 0, errors.New("not enought args")
	}
	timeout := args[1]
	timeoutMs, err := strconv.Atoi(timeout)
	if err != nil {
		return 0, err
	}
	start := time.Now()
	for {
		end := time.Now()
		time.Sleep(500 * time.Microsecond)

		if end.Sub(start).Milliseconds() > int64(timeoutMs) {
			return 0, errors.New("timeout")
		}
		break
	}
	return 0, nil
}
