package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/db"
)

func Set(command Command, database *db.Database) error {

	args := command.Args
	exp := -1

	if len(args) < 2 {
		return errors.New("Failed to set mapping, not enough args")
	}
	key := args[0]
	value := db.StringType(args[1])

	if len(args) == 4 {
		option := strings.ToUpper(args[2])
		optionVal := args[3]
		switch option {
		case "EX", "PX":
			optionVal, err := strconv.Atoi(optionVal)
			if err != nil {
				return fmt.Errorf("Couldn't convert value to an integer: %s", optionVal)
			}
			factor := 1
			if option == "EX" {
				factor = 1000
			}
			exp = optionVal * factor
		default:
			return errors.New("Unknown option")
		}
	}
	database.Set(key, db.Data{Value: value, CreatedTime: time.Now(), Exp: exp})

	return nil

}

func Get(command Command, database *db.Database) (db.StringType, error) {
	args := command.Args
	if len(args) < 1 {
		return "", errors.New("Failed to get mapping, not enough args")
	}

	key := args[0]

	if v, ok := database.Get(key); ok {

		if value, ok := v.Value.(db.StringType); ok {
			return value, nil

		} else {
			return "", fmt.Errorf("Expecte StringType value, got :%T", v.Value)
		}

	} else {
		return "", errors.New("Missing key")
	}
}

func Incr(command Command, database *db.Database) (int, error) {
	args := command.Args

	if len(args) < 1 {
		return 0, errors.New("failed to INCR, not enough args")
	}
	key := args[0]

	upd := func(data db.Data, exists bool) (db.Data, int, error) {
		var value int

		if !exists {
			value = 1 // initializing at 0

		} else {
			dbVal, ok := data.Value.(db.StringType)

			if !ok {
				return data, 0, fmt.Errorf("failed to INCR, wrong data type;: %T", data.Value)
			}

			nrVal, err := strconv.Atoi(string(dbVal))
			if err != nil {
				return data, 0, errors.New("ERR value is not an integer or out of range")
			}

			value = nrVal + 1
		}

		data.Value = db.StringType(strconv.Itoa(value))
		return data, value, nil
	}

	out, err := db.Update(database, key, upd)
	if err != nil {
		return 0, err
	}
	return out, nil

}
