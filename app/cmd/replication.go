package cmd

import "fmt"

func Info(command Command) string {
	args := command.Args

	if len(args) < 1 {
		return "role:unknown no args"
	}

	key := args[0]
	if key == "replication" {
		return "role:master"
	}
	return fmt.Sprintf("role:unknown key %s", key)

}
