package main

import (
	"log"
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
			log.Printf("replica failed to connect: %v", err)
			os.Exit(1)
		}
		log.Printf("connected to replica")
	}

	// regular mode
	err := connection.OpenClient(&AppConfig, &db)
	if err != nil {
		log.Printf("couldnt connect: %v", err)
		os.Exit(1)
	}

}
