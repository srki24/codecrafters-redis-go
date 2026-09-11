package cmd

import "errors"

type ListMapping = map[string][]string

func NewListMapping() ListMapping {
	mapping := make(ListMapping)
	return mapping

}

func AddElement(cmd Command, lst ListMapping) (ListMapping, error) {
	args := cmd.Args
	if len(args) < 2 {
		return lst, errors.New("Failed to push to the list, not enough args")
	}

	key := args[0]
	value := args[1]

	if v, ok := lst[key]; ok {
		v := append(v, value)
		lst[key] = v
	} else {
		v := []string{value}
		lst[key] = v
	}

	return lst, nil
}

func GetNrElems(cmd Command, lst ListMapping) int {
	args := cmd.Args

	if v, ok := lst[args[0]]; ok {
		return len(v)
	}
	return 0
}
