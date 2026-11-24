package main

import (
	"log"
	"os"

	"github.com/codecrafters-io/redis-starter-go/app/aof"
	"github.com/codecrafters-io/redis-starter-go/app/config"
	"github.com/codecrafters-io/redis-starter-go/app/connection"
	"github.com/codecrafters-io/redis-starter-go/app/db"
	"github.com/codecrafters-io/redis-starter-go/app/rdb"
)

var AppConfig config.Config

func main() {
	config.ParseInto(&AppConfig)

	db := db.NewDatabase(&AppConfig)

	if err := rdb.Load(&db, AppConfig.Dir, AppConfig.DbFilename); err != nil {
		log.Printf("failed to load RDB file: %v", err)
		os.Exit(1)
	}

	if err := aof.InitializeAOF(&AppConfig, &db); err != nil {
		log.Printf("failed to initilize AOF: %v", err)
		os.Exit(1)
	}
	// replication mode
	if AppConfig.IsReplica() {
		go func() {
			err := connection.OpenReplica(&AppConfig, &db)
			if err != nil {
				log.Printf("replica failed to connect: %v", err)
				os.Exit(1)
			}
		}()
	}
	// regular mode
	err := connection.OpenClient(&AppConfig, &db)
	if err != nil {
		log.Printf("couldnt connect: %v", err)
		os.Exit(1)
	}

}
