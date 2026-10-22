package config

import "flag"

type Config struct {
	Port      int
	ReplicaOf string
}

func ParseInto(config *Config) {
	flag.IntVar(&config.Port, "port", 6379, "Port to start the server on")
	flag.StringVar(&config.ReplicaOf, "replicaof", "", "Start Redis server as a replica")
	flag.Parse()
}
