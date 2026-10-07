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
	database.Set(key, db.Data{Value: value, Time: time.Now(), Exp: exp})

	return nil

}

func Get(command Command, database *db.Database) (db.StringType, error) {
	args := command.Args

	if len(args) < 1 {
		return "", errors.New("Failed to get mapping, not enough args")
	}

	key := args[0]

	if v, ok := database.Get(key); ok {

		cTime := time.Now()

		elapsed := int(cTime.Sub(v.Time).Milliseconds())

		if (v.Exp > 0) && elapsed > v.Exp {
			return "", errors.New("Key expired")
		}

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
	var value int
	args := command.Args

	if len(args) < 1 {
		return 0, errors.New("Failed to INCR, not enough args")
	}
	key := args[0]

	data, hasKey := database.Get(key)

	if !hasKey {
		value = 1 // initializing at 0
	} else {
		dbVal, isString := data.Value.(db.StringType)
		if !isString {
			return 0, fmt.Errorf("Value is not of the string type: %T", value)
		}

		nrVal, err := strconv.Atoi(string(dbVal))
		if err != nil {
			return 0, fmt.Errorf("Couldn't convert value to integer: %s", string(dbVal))
		}

		value = nrVal + 1
	}

	data.Value = db.StringType(strconv.Itoa(value))
	database.Set(key, data)
	return int(value), nil

}
