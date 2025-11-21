package main

import (
	"flag"
	"log"

	"github.com/Egor-Evsikov/crschback/src/db"
	"github.com/Egor-Evsikov/crschback/src/server"
)

var (
	configServerPath string
	configDBPath     string
)

func init() {
	flag.StringVar(&configServerPath, "server-config-path", "./src/server/serverconf.yaml", "path to server config file")
	flag.StringVar(&configDBPath, "DB-config-path", "./src/db/dbConfig.yaml", "path to DB config file")

}

func main() {
	flag.Parse()

	sconf, err := server.LoadServerConfig(configServerPath)
	if err != nil {
		log.Fatal("Ошибка в loadserverconfig ", err)
	}

	dbconf, err := db.LoadDBConfig(configDBPath)
	if err != nil {
		log.Fatal("Ошибка в loaddbconfig ", err)
	}

	dtbs, err := db.ConnDB(dbconf)
	if err != nil {
		log.Fatal("Ошибка в conndb ", err)
	}

	apiServer := server.NewServer(sconf, dtbs)
	err = apiServer.Start()
	if err != nil {
		log.Fatal("Ошибка в start ", err)
	}

}
