package cmd

import (
	"errors"

	"github.com/codecrafters-io/redis-starter-go/app/db"
)

func Type(command Command, database db.Database) (string, error) {
	args := command.Args
	if len(args) < 1 {
		return "", errors.New("Failed to get type, not enough args")
	}

	key := args[0]

	if data, ok := database[key]; ok {
		return data.Value.GetType(), nil
	}
	return "none", nil
}
