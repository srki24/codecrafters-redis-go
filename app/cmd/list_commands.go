package cmd

import (
	"errors"
	"slices"
	"strconv"
)

type ListMapping = map[string][]string

func NewListMapping() ListMapping {
	mapping := make(ListMapping)
	return mapping

}

func ListPush(cmd Command, lst ListMapping, right bool) error {
	args := cmd.Args
	if len(args) < 2 {
		return errors.New("Failed to push to the list, not enough args")
	}

	key := args[0]
	values := args[1:]
	if !right {
		slices.Reverse(values)
	}

	if data, ok := lst[key]; ok {
		var v []string
		if right {
			v = append(data, values...)
		} else {
			v = append(values, data...)
		}
		lst[key] = v
	} else {
		lst[key] = values
	}

	return nil
}

func GetNrElems(cmd Command, lst ListMapping) int {
	args := cmd.Args
	if len(args) < 1 {
		return 0
	}

	if v, ok := lst[args[0]]; ok {
		return len(v)
	}
	return 0
}

func LRange(cmd Command, lst ListMapping) ([]string, error) {

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

	if data, ok := lst[key]; ok {
		nrElements := len(data)

		if fromIdx < 0 {
			fromIdx = max(0, nrElements+fromIdx)
		}

		if toIdx < 0 {
			toIdx = max(0, nrElements+toIdx+1)
		} else {
			toIdx = min(toIdx+1, nrElements)
		}

		return data[fromIdx:toIdx], nil
	}
	return nil, errors.New("Non existing list")

}

func ListPop(cmd Command, lst ListMapping) ([]string, error) {
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

	if data, ok := lst[key]; ok {
		toPop = min(toPop, len(data))
		lst[key] = data[toPop:]
		return data[:toPop], nil
	}
	return nil, errors.New("List doesn,t exist")

}
