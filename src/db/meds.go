package db

import "time"

type Medicine struct {
	Id     int       `json:"id"`
	Name   string    `json:"name"`
	Date   time.Time `json:"date"`
	Cost   float64   `json:"cost"`
	Amount int       `json:"amount"`
	DirId  int       `json:"id_directory"`
}
