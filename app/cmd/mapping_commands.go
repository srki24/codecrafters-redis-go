package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Mapping struct {
	value string
	time  time.Time
	exp   int
}

func NewMapping() map[string]Mapping {
	mapping := make(map[string]Mapping)
	return mapping

}
func SetMapping(command Command, mapping map[string]Mapping) error {

	args := command.Args
	exp := -1

	if len(args) < 2 {
		return errors.New("Failed to set mapping, not enough args")
	}
	key := args[0]
	value := args[1]

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
	mapping[key] = Mapping{value: value, time: time.Now(), exp: exp}

	return nil

}

func GetMapping(command Command, mapping map[string]Mapping) (string, error) {
	args := command.Args
	fmt.Println(args)

	if len(args) < 1 {
		return "", errors.New("Failed to get mapping, not enough args")
	}

	k := args[0]

	if v, ok := mapping[k]; ok {

		cTime := time.Now()

		elapsed := int(cTime.Sub(v.time).Milliseconds())

		if (v.exp > 0) && elapsed > v.exp {
			return "", errors.New("Key expired")
		}
		return v.value, nil
	} else {
		return "", errors.New("Missing key")
	}
}
