package cmd

import (
	"errors"
	"fmt"
	"strings"

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

func replconf(command Command) ([]string, error) {
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
		return []string{"REPLCONF", "ACK", "0"}, nil
	}
	return nil, fmt.Errorf("unknown args: %s", args)

}
