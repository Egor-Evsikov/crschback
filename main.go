package main

import (
	"log"

	"net/http"

	"github.com/Egor-Evsikov/crschback/src/api"
	"github.com/Egor-Evsikov/crschback/src/db"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Post("/login", api.UserLogin)

	cfg, err := db.LoadDBConfig("./src/db/dbConfig.yaml")

	if err != nil {
		log.Fatal("Ошибка загрузки конфига")
	}

	log.Print(cfg)

	dtbs, err := db.ConnDB(*cfg)
	if err != nil {
		log.Fatal("Ошибка подключения к бд      ", err)
	}
	defer dtbs.Close()
	db.CreateTable(dtbs)
	db.SaveUser(dtbs, "aaa", "bbb")

	http.ListenAndServe("127.0.0.1:8080", r)
}
