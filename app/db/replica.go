package db

import (
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/config"
)

const idSize = 40

type Replica struct {
	role   string
	id     string
	offset int
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
		role:   role,
		id:     generateReplicaId(),
		offset: 0,
	}
}

func (r *Replica) AddOffset(n int) {
	r.offset += n
}

func (r Replica) GetOffset() int {
	return r.offset
}

func (r Replica) Id() string {
	return r.id
}

func (r Replica) String() string {
	var out strings.Builder

	out.WriteString("role:")
	out.WriteString(r.role)
	out.WriteString("master_replid:")
	out.WriteString(r.id)
	out.WriteString("master_repl_offset:")
	out.WriteString(strconv.Itoa(r.offset))
	return out.String()
}
