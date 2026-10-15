package cmd

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/db"
)

var ErrNotFound = errors.New("not found")

func ListPush(cmd Command, database *db.Database, right bool) (int, error) {
	database.Mu.Lock()
	defer database.Mu.Unlock()

	args := cmd.Args
	if len(args) < 2 {
		return 0, errors.New("Failed to push to the list, not enough args")
	}

	key := args[0]
	var values []string
	copy(args[1:], values)

	if !right {
		slices.Reverse(values)
	}

	if data, ok := database.Data[key]; ok {
		if val, ok := data.Value.(db.ListType); ok {

			if right {
				values = append(val, values...)
			} else {
				values = append(values, val...)
			}
		} else {
			return 0, fmt.Errorf("Expecte ListType value, got :%T", data.Value)

		}
	}
	database.Data[key] = db.Data{Value: db.ListType(values), CreatedTime: time.Now(), Exp: -1}

	return len(values), nil
}

func GetNumberOfElments(cmd Command, database *db.Database) int {
	args := cmd.Args
	if len(args) < 1 {
		return 0
	}
	key := args[0]
	if data, ok := database.Get(key); ok {
		if val, ok := data.Value.(db.ListType); ok {
			return len(val)

		}
	}
	return 0
}

func LRange(cmd Command, database *db.Database) ([]string, error) {

	args := cmd.Args
	if len(args) < 3 {
		return nil, errors.New("Failed to get range, not enough args")
	}

	key := args[0]

	fromIdx, err := strconv.Atoi(args[1])
	if err != nil {
		return nil, err
	}

	toIdx, err := strconv.Atoi(args[2])
	if err != nil {
		return nil, err
	}

	if data, ok := database.Get(key); ok {
		if val, ok := data.Value.(db.ListType); ok {

			nrElements := len(val)

			if fromIdx > nrElements {
				return nil, fmt.Errorf("Index out of range")
			}
			if fromIdx < 0 {
				fromIdx = max(0, nrElements+fromIdx)
			}

			if toIdx < 0 {
				toIdx = max(0, nrElements+toIdx+1)
			} else {
				toIdx = min(toIdx+1, nrElements)
			}

			return val[fromIdx:toIdx], nil
		}
		return nil, fmt.Errorf("Expecte ListType value, got :%T", data.Value)

	}
	return nil, errors.New("Non existing list")

}

func ListPop(cmd Command, database *db.Database) ([]string, error) {

	database.Mu.Lock()
	defer database.Mu.Unlock()

	args := cmd.Args
	if len(args) < 1 {
		return nil, errors.New("Failed to get pop element, not enough args")
	}

	key := args[0]
	toPop := 1

	if len(args) == 2 {
		newPop, err := strconv.Atoi(args[1])
		if err != nil {
			return nil, errors.New("Failed to parse pop argument")
		}
		toPop = newPop
	}
	if data, ok := database.Data[key]; ok {
		if val, ok := data.Value.(db.ListType); ok {
			if len(val) == 0 {
				return nil, ErrNotFound
			}
			toPop = min(toPop, len(val))
			data.Value = val[toPop:]
			database.Data[key] = data

			return val[:toPop], nil
		}
		return nil, fmt.Errorf("Expecte ListType value, got :%T", data.Value)
	}
	return nil, ErrNotFound

}

func ListBLPop(cmd Command, database *db.Database) ([]string, error) {
	args := cmd.Args
	if len(args) < 2 {
		return nil, errors.New("Couldnt block pop, not enough args")
	}

	key := args[0]
	timeout, err := strconv.ParseFloat(args[1], 64)

	if err != nil {
		return nil, err
	}

	popCmd := Command{Name: "POP", Args: []string{key}}

	start := time.Now()

	for {
		value, err := ListPop(popCmd, database)

		if errors.Is(err, ErrNotFound) {
			end := time.Now()

			if (timeout != 0) && end.Sub(start).Seconds() > timeout {
				return nil, errors.New("No data, timed out")
			}
			continue
		}

		if err != nil {
			return nil, err
		}
		return []string{key, value[0]}, nil

	}

}
