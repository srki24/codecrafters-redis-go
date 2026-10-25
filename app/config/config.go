package config

import (
	"flag"
	"fmt"
	"strings"
)

type Config struct {
	Port      int
	ReplicaOf string
}

func (c *Config) ParseReplica() (host, port string, err error) {
	identity := c.ReplicaOf

	parts := strings.SplitN(identity, " ", 2)
	if len(parts) != 2 {
		err = fmt.Errorf("Failed to parse replica, <HOST> <PORT> expected, got: %s", identity)
		return
	}
	host = parts[0]
	port = parts[1]
	return host, port, nil

}

func (c *Config) IsReplica() bool {
	return c.ReplicaOf != ""
}

func ParseInto(config *Config) {
	flag.IntVar(&config.Port, "port", 6379, "Port to start the server on")
	flag.StringVar(&config.ReplicaOf, "replicaof", "", "Start Redis server as a replica")
	flag.Parse()
}
