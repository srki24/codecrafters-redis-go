package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/db"
)

func Type(command Command, database *db.Database) (string, error) {
	args := command.Args
	if len(args) < 1 {
		return "", errors.New("Failed to get type, not enough args")
	}

	key := args[0]

	if data, ok := database.Get(key); ok {
		return data.Value.GetType(), nil
	}
	return "none", nil
}

func ConfigGet(command Command, database *db.Database) (out []string, err error) {
	args := command.Args
	if len(args) < 2 {
		return out, errors.New("Failed to execute config get, not enough args")
	}

	if args[0] != "GET" {
		return out, fmt.Errorf("Unknown arg %s, expected GET", args[0])
	}

	for _, arg := range args[1:] {
		if strings.ToLower(arg) == "dir" {
			dir := []string{arg, database.GetConfig().Dir}
			out = append(out, dir...)
		} else if strings.ToLower(arg) == "dbfilename" {
			filename := []string{arg, database.GetConfig().DbFilename}
			out = append(out, filename...)
		} else {
			return out, fmt.Errorf("Unknown arg %s", arg)
		}

	}

	if len(out) == 0 {
		return out, fmt.Errorf("Unknown args: %s", args)
	}
	return out, nil
}
