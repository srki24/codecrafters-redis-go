package cmd

import "errors"

func NewList() []string {
	return []string{}
}

func AddElement(cmd Command, lst []string) ([]string, error) {
	args := cmd.Args
	if len(args) < 2 {
		return lst, errors.New("Failed to push to the list, not enough args")
	}
	lst = append(lst, args[1])
	return lst, nil
}
