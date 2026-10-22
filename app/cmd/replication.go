package cmd

import (
	"fmt"

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
