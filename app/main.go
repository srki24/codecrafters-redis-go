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

	if AppConfig.IsReplica() {
		err := connection.Handshake(&AppConfig)
		if err != nil {
			fmt.Println("replica failed to connect: %v", err)
			os.Exit(1)
		}
	}
	db := db.NewDatabase(&AppConfig)
	connection.Open(&AppConfig, &db)

}
