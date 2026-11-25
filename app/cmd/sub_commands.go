package cmd

import "errors"

func Subscribe(command Command, cs *ConnStatus) (int, string, error) {
	args := command.Args
	if len(args) < 1 {
		return 0, "", errors.New("failed to subscribe, not enough args")
	}
	ch := args[0]
	cs.Subscribe(ch)
	return cs.NumberOfSubscriptions(), ch, nil
}
