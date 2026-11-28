package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/db"
)

func Zadd(cmd Command, database *db.Database) (int, error) {
	args := cmd.Args

	if len(args) < 3 {
		return 0, errors.New("Err Failed to add sorted set, not enough args")
	}
	dbKey := args[0]
	value, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return 0, err
	}
	key := args[2]

	fn := func(data db.Data, exists bool) (db.Data, int, error) {
		var ss db.SortedSet
		var inserted int = 0
		if !exists {
			ss = db.SortedSet{} // new
		} else {
			dbss, ok := data.Value.(db.SortedSet)
			if !ok {
				return data, 0, fmt.Errorf("expected sorted set for key %s, got %T", dbKey, data.Value)
			}
			ss = dbss

		}
		if idx := ss.Index(key); idx == -1 { //new key added
			inserted = 1
		}
		ss.Add(key, value)
		data.Value = ss
		data.UpdatedTime = time.Now()
		return data, inserted, nil
	}

	return db.Update(database, dbKey, fn)

}

func confirmZset(database *db.Database, key string) (db.SortedSet, error) {
	emptySet := db.SortedSet{}
	data, ok := database.Get(key)
	if !ok {
		return emptySet, ErrNotFound
	}
	ss, ok := data.Value.(db.SortedSet)
	if !ok {
		return emptySet, fmt.Errorf("expected sorted set for key %s, got %T", key, data.Value)
	}
	return ss, nil
}

func Zrank(cmd Command, database *db.Database) (int, error) {
	args := cmd.Args
	if len(args) < 2 {
		return 0, errors.New("Err Failed to get rank of a sorted set sorted set, not enough args")
	}
	dbKey := args[0]
	name := args[1]
	if ss, err := confirmZset(database, dbKey); err != nil {
		return 0, err
	} else {
		if idx := ss.Index(name); idx == -1 {
			return idx, ErrNotFound
		} else {
			return idx, nil
		}

	}

}

func Zrange(cmd Command, database *db.Database) ([]string, error) {
	args := cmd.Args
	if len(args) < 3 {
		return nil, errors.New("Err Failed to get range of a sorted set sorted set, not enough args")
	}
	dbKey := args[0]
	from, err := strconv.Atoi(args[1])
	if err != nil {
		return nil, err
	}
	to, err := strconv.Atoi(args[2])
	if err != nil {
		return nil, err
	}

	if ss, err := confirmZset(database, dbKey); err != nil {
		return nil, err
	} else {
		return ss.Range(from, to)
	}
}

func Zcard(cmd Command, database *db.Database) (int, error) {
	args := cmd.Args
	if len(args) < 1 {
		return 0, errors.New("Err Failed to get zcard of a sorted set sorted set, not enough args")
	}
	dbKey := args[0]

	if ss, err := confirmZset(database, dbKey); err != nil {
		if errors.Is(err, ErrNotFound) {
			return 0, nil
		} else {
			return 0, err
		}
	} else {
		return ss.NrElements(), nil
	}
}

func Zcore(cmd Command, database *db.Database) (string, error) {
	args := cmd.Args
	if len(args) < 2 {
		return "", errors.New("Err Failed to get zcard of a sorted set sorted set, not enough args")
	}
	dbKey := args[0]
	ssKey := args[1]

	if ss, err := confirmZset(database, dbKey); err != nil {
		return "", nil
	} else {
		f, ok := ss.Score(ssKey)
		if !ok {
			return "-1", nil
		}
		return strconv.FormatFloat(f, 'f', '4', 64), nil
	}
}
