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
	fmt.Println(args)

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
