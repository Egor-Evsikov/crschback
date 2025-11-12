package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

//docker exec -it pg psql -U Egrik -d pg заход в бдшку

// * Абстракция на бд
type Repo struct {
	*sql.DB
}

// * Подключение к дб и подгрузка конфигов
func ConnDB(conf *DbConfig) (*Repo, error) {

	connStr := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=%s", conf.User, conf.Password, conf.Name, conf.Host, conf.Port, conf.SslMode)
	log.Println("Данные конфига бд ", connStr)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	r := NewRepo(db)
	r.CreateTable()

	return r, r.Ping()
}

// * Инициализация кастомной бд
func NewRepo(s *sql.DB) *Repo {
	return &Repo{s}
}

// * функции работы с кастомной бд
// * Создание таблиц
func (s *Repo) CreateTable() error {
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
	_, err := s.Exec(tableUsers)
	if err != nil {
		return err
	}

	_, err = s.Exec(tableDir)
	if err != nil {
		return err
	}

	_, err = s.Exec(tableDirUsers)
	if err != nil {
		return err
	}

	_, err = s.Exec(tableMeds)
	if err != nil {
		return err
	}
	return nil
}

// * Сохранение пользователя
func (s *Repo) SaveUser(login string, password string) error {

	insertStr := "INSERT INTO users(login, password) VALUES ( $1, $2 )"
	stmt, err := s.Prepare(insertStr)
	if err != nil {
		return err
	}

	defer stmt.Close()

	if _, err = stmt.Exec(login, password); err != nil {
		return err
	}

	return nil
}

// * Проверка наличия пользователя
func (s *Repo) CheckUser(login string) {

}
