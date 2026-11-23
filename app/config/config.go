package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port           int
	ReplicaOf      string
	Dir            string
	DbFilename     string
	AppendOnly     string
	AppendDirName  string
	AppendFileName string
	AppendFSync    string
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
	pwd, _ := os.Getwd()
	flag.IntVar(&config.Port, "port", 6379, "Port to start the server on")
	flag.StringVar(&config.ReplicaOf, "replicaof", "", "Start Redis server as a replica")
	flag.StringVar(&config.Dir, "dir", pwd, "Directory where RBD file is stored")
	flag.StringVar(&config.DbFilename, "dbfilename", "", "RDB filename")
	flag.StringVar(&config.AppendOnly, "appendonly", "no", "Append only mode")
	flag.StringVar(&config.AppendDirName, "appenddirname", "appendonlydir", "Append only subdir")
	flag.StringVar(&config.AppendFileName, "appendfilename", "appendonly.aof", "Append only filename")
	flag.StringVar(&config.AppendFSync, "appendfsync", "everysec", "Frequency of flush writes")
	flag.Parse()
}
