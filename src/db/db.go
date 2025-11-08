package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	_ "github.com/lib/pq"
)

//docker exec -it pg psql -U Egrik -d pgbd заход в бдшку

type DbConfig struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	SslMode  string `yaml:"sslMode"`
}

//"user=Egrik password=n dbname=pgdb host=localhost port=8888 sslmode=disable"

func LoadDBConfig(path string) (*DbConfig, error) {

	if path == "" {
		log.Fatal("no path")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Fatal("файл не найден ", err)
	}

	var cfg DbConfig

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		log.Fatal("Не считал конфиг ", err)
	}

	return &cfg, nil
}

func ConnDB(conf DbConfig) (*sql.DB, error) {

	connStr := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=%s", conf.User, conf.Password, conf.Name, conf.Host, conf.Port, conf.SslMode)
	log.Print(connStr)

	db, err := sql.Open("postgres", connStr)

	if err != nil {
		log.Fatal("при открытии бд", err)
	}

	CreateTable(db)

	return db, db.Ping()
}

func CreateTable(db *sql.DB) {
	tableUsers := `CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		login VARCHAR(50) NOT NULL UNIQUE,
		password VARCHAR(50) NOT NULL
	)`
	tableDir := `CREATE TABLE IF NOT EXISTS directories(
		id SERIAL PRIMARY KEY,
		id_owner INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name VARCHAR(50)
	)`
	tableDirUsers := `CREATE TABLE IF NOT EXISTS dir_users(
		directory_id INT NOT NULL REFERENCES directories(id) ON DELETE CASCADE,
		user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		PRIMARY KEY (directory_id, user_id)
	)`
	tableMeds := `CREATE TABLE IF NOT EXISTS medicines(
		id SERIAL PRIMARY KEY,
		name VARCHAR(50) NOT NULL,
		date TIMESTAMPTZ DEFAULT NOW(),
		cost NUMERIC(5,2) DEFAULT 0,
		id_directory INT NOT NULL REFERENCES directories(id) ON DELETE CASCADE
	)`
	_, err := db.Exec(tableUsers)
	if err != nil {
		log.Fatal("ошибка создания таблицы1 ", err)
	}

	_, err = db.Exec(tableDir)
	if err != nil {
		log.Fatal("ошибка создания таблицы2 ", err)
	}

	_, err = db.Exec(tableDirUsers)
	if err != nil {
		log.Fatal("ошибка создания таблицы3 ", err)
	}

	_, err = db.Exec(tableMeds)
	if err != nil {
		log.Fatal("ошибка создания таблицы4 ", err)
	}

}

func SaveUser(dtbs *sql.DB, login string, password string) {
	insertStr := "INSERT INTO users(login, password) VALUES ( $1, $2 )"

	stmt, err := dtbs.Prepare(insertStr)
	if err != nil {
		log.Fatal("mistake1 ", err)
	}

	defer stmt.Close()

	if _, err = stmt.Exec(login, password); err != nil {
		log.Fatal("mistake2 ", err)
	}

}
