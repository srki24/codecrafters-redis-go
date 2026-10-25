package db

import (
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/config"
)

const idSize = 40

type Replica struct {
	role             string
	masterReplid     string
	masterReplOffset int
}

func generateReplicaId() string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	id := make([]byte, idSize)

	for i := range idSize {
		id[i] = chars[rand.IntN(len(chars))]
	}
	return string(id)

}
func newReplica(config *config.Config) Replica {
	var role string
	replicaOf := config.ReplicaOf
	if replicaOf == "" {
		role = "master"
	} else {
		role = "slave"
	}

	return Replica{
		role:             role,
		masterReplid:     generateReplicaId(),
		masterReplOffset: 0,
	}
}

func (r Replica) Id() string {
	return r.masterReplid
}

func (r Replica) String() string {
	var out strings.Builder

	out.WriteString("role:" + r.role)
	out.WriteString("master_replid:" + r.masterReplid)
	out.WriteString("master_repl_offset:" + strconv.Itoa(r.masterReplOffset))

	return out.String()
}
