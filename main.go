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

	// r := chi.NewRouter()
	// r.Use(middleware.Logger)
	// h, d, err := Collector()
	// if err != nil {
	// 	panic("Ошибка коллектора, проект не запустится ")
	// }
	// defer d.Close()

	// r.Post("/login", h.UserLogin())
	// r.Post("/register", h.UserRegister())

	// http.ListenAndServe("127.0.0.1:8080", r)
}

// func Collector() (*api.HandlerDB, *db.Repo, error) {
// 	cfg, err := db.LoadDBConfig("./src/db/dbConfig.yaml")

// 	if err != nil {
// 		log.Println("Ошибка при загрузке конфига для бд ", err)
// 		return nil, nil, err
// 	}

// 	dtbs, err := db.ConnDB(*cfg)
// 	if err != nil {
// 		log.Println("Ошибка при подключении к бд")
// 		return nil, nil, err
// 	}

// 	h := api.NewHandlerDB(dtbs)
// 	return h, dtbs, nil
// }
