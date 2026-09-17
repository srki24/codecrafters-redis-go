package cmd

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/db"
)

func ListPush(cmd Command, database *db.Database, right bool) error {
	args := cmd.Args
	if len(args) < 2 {
		return errors.New("Failed to push to the list, not enough args")
	}

	key := args[0]
	values := args[1:]
	if !right {
		slices.Reverse(values)
	}

	if data, ok := database.Get(key); ok {
		if val, ok := data.Value.(db.ListType); ok {

			if right {
				values = append(val, values...)
			} else {
				values = append(values, val...)
			}
		} else {
			return fmt.Errorf("Expecte ListType value, got :%", reflect.TypeOf((data.Value)))

		}
	}
	database.Set(key, db.Data{Value: db.ListType(values), Time: time.Now(), Exp: -1})

	return nil
}

func GetNrElems(cmd Command, database *db.Database) int {
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
		return nil, fmt.Errorf("Expecte ListType value, got :%", reflect.TypeOf((data.Value)))

	}
	return nil, errors.New("Non existing list")

}

func ListPop(cmd Command, database *db.Database) ([]string, error) {
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

	if data, ok := database.Get(key); ok {
		if val, ok := data.Value.(db.ListType); ok {
			if len(val) == 0 {
				return nil, errors.New("No data to pop")
			}
			toPop = min(toPop, len(val))
			data.Value = val[toPop:]
			database.Set(key, data)

			return val[:toPop], nil
		}
		return nil, fmt.Errorf("Expecte ListType value, got :%", reflect.TypeOf((data.Value)))
	}
	return nil, errors.New("List doesn,t exist")

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
		if err == nil {
			return []string{key, value[0]}, nil
		}

		end := time.Now()

		if (timeout != 0) && end.Sub(start).Seconds() > timeout {
			return nil, errors.New("No data, timed out")
		}

	}

}
