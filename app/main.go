package main

import (
	"fmt"
	"os"

	"github.com/codecrafters-io/redis-starter-go/app/config"
	"github.com/codecrafters-io/redis-starter-go/app/connection"
	"github.com/codecrafters-io/redis-starter-go/app/db"
)

var AppConfig config.Config

func main() {
	config.ParseInto(&AppConfig)

	db := db.NewDatabase(&AppConfig)

	// replication mode
	if AppConfig.IsReplica() {
		err := connection.OpenReplica(&AppConfig, &db)
		if err != nil {
			fmt.Printf("replica failed to connect: %v", err)
			os.Exit(1)
		}
	}

	// regular mode
	err := connection.OpenClient(&AppConfig, &db)
	if err != nil {
		fmt.Printf("couldnt connect: %v", err)
		os.Exit(1)
	}

}
