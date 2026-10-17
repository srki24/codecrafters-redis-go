package cmd

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/db"
)

func ListPush(cmd Command, database *db.Database, right bool) (int, error) {
	args := cmd.Args
	if len(args) < 2 {
		return 0, errors.New("failed to push to the list, not enough args")
	}
	key := args[0]
	upd := func(data db.Data, exists bool) (db.Data, int, error) {
		values := make([]string, len(args[1:]))
		copy(values, args[1:])
		if !right {
			slices.Reverse(values)
		}
		if !exists {
			def := db.NewData(db.ListType(values), nil)
			return def, len(values), nil
		}
		val, ok := data.Value.(db.ListType)
		if !ok {
			return db.Data{}, 0, fmt.Errorf("expecte ListType value, got :%T", data.Value)
		}
		if right {
			values = append(val, values...)
		} else {
			values = append(values, val...)
		}
		data.Value = db.ListType(values)
		return data, len(values), nil
	}

	out, err := db.Update(
		database,
		key,
		upd)

	if err != nil {
		return 0, err
	}

	return out, nil
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
		return nil, errors.New("failed to get range, not enough args")
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
				return nil, errors.New("index out of range")
			}
			if fromIdx < 0 {
				fromIdx = max(0, nrElements+fromIdx)
			}

			if toIdx < 0 {
				toIdx = max(0, nrElements+toIdx+1)
			} else {
				toIdx = min(toIdx+1, nrElements)
			}

			if toIdx < fromIdx {
				return nil, errors.New("to index must be greater then from index")
			}
			return val[fromIdx:toIdx], nil
		}
		return nil, fmt.Errorf("expecte ListType value, got :%T", data.Value)

	}
	return nil, ErrNotFound

}

func ListPop(cmd Command, database *db.Database) ([]string, error) {

	args := cmd.Args
	if len(args) < 1 {
		return nil, errors.New("failed to get pop element, not enough args")
	}

	key := args[0]
	toPop := 1

	if len(args) == 2 {
		newPop, err := strconv.Atoi(args[1])
		if err != nil {
			return nil, errors.New("failed to parse pop argument")
		}
		toPop = newPop
	}

	upd := func(data db.Data, exists bool) (db.Data, []string, error) {

		if !exists {
			return db.Data{}, nil, ErrNotFound
		}
		val, ok := data.Value.(db.ListType)
		if !ok {
			return db.Data{}, nil, fmt.Errorf("expecte ListType value, got :%T", data.Value)
		}

		if len(val) == 0 {
			return db.Data{}, nil, ErrNotFound
		}
		n := min(toPop, len(val))

		data.Value = val[n:]
		data.UpdatedTime = time.Now()

		return data, val[:n], nil
	}

	out, err := db.Update(database, key, upd)

	if err != nil {
		return nil, err
	}

	return out, nil

}

func ListBLPop(cmd Command, database *db.Database) ([]string, error) {
	args := cmd.Args
	if len(args) < 2 {
		return nil, errors.New("couldnt block pop, not enough args")
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
				return nil, errors.New("no data, timed out")
			}
			time.Sleep(500 * time.Microsecond)
			continue
		}

		if err != nil {
			return nil, err
		}
		return []string{key, value[0]}, nil

	}

}
