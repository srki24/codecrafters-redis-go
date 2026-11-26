package cmd

import (
	"errors"

	"github.com/codecrafters-io/redis-starter-go/app/resp"
	"github.com/codecrafters-io/redis-starter-go/app/sub"
)

func Subscribe(command Command, cs *ConnStatus, sh *sub.Hub) (int, string, error) {
	args := command.Args
	if len(args) < 1 {
		return 0, "", errors.New("failed to subscribe, not enough args")
	}
	sub := args[0]
	cs.Subscribe(sub)
	sh.AddChannel(sub, cs.Outbound())
	return cs.NumberOfSubscriptions(), sub, nil
}

func Publish(command Command, cs *ConnStatus, sh *sub.Hub) (int, error) {
	args := command.Args
	if len(args) < 2 {
		return 0, errors.New("failed to publish, not enough args")
	}
	sub := args[0]
	text := args[1]

	msg := resp.NewArray([]string{"message", sub, text})

	err := sh.Publish(msg.Serialize(), sub)
	if err != nil {
		return 0, err
	}

	return sh.NumberOfSubscribers(sub), nil

}

func Unsusbscribe(command Command, cs *ConnStatus, sh *sub.Hub) (int, string, error) {
	args := command.Args
	if len(args) < 1 {
		return 0, "", errors.New("failed to subscribe, not enough args")
	}
	sub := args[0]
	cs.Unsubscribe(sub)
	sh.RemoveChannel(sub, cs.Outbound())
	return cs.NumberOfSubscriptions(), sub, nil
}
