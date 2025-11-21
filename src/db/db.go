package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	tableUsers := `CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		login VARCHAR(50) NOT NULL,
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
	_, err := s.ExecContext(ctx, tableUsers)
	if err != nil {
		return err
	}

	_, err = s.ExecContext(ctx, tableDir)
	if err != nil {
		return err
	}

	_, err = s.ExecContext(ctx, tableDirUsers)
	if err != nil {
		return err
	}

	_, err = s.ExecContext(ctx, tableMeds)
	if err != nil {
		return err
	}
	return nil
}

// * Сохранение пользователя
func (s *Repo) SaveUser(ctx context.Context, login string, password string) (int, error) {

	check, err := s.CheckUser(ctx, login, password)
	if err != nil {
		return 0, err
	}
	if check {
		return 0, fmt.Errorf("пользователь существует")
	}

	query := "INSERT INTO users(login, password) VALUES ($1, $2) RETURNING id"
	var key int

	err = s.QueryRowContext(ctx, query, login, password).Scan(&key)
	if err != nil {
		return 0, err
	}

	return key, nil
}

// * Проверка существование пользователя
func (s *Repo) CheckUser(ctx context.Context, login string, password string) (bool, error) {

	query := "SELECT EXISTS (SELECT 1 FROM 	users WHERE login = $1 AND password = $2)"
	var exist bool
	err := s.QueryRowContext(ctx, query, login, password).Scan(&exist)
	if err != nil {
		return false, err
	}
	return exist, nil

}

// * Получение id пользователя
func (s *Repo) GetUserId(ctx context.Context, login string) (int, error) {

	query := "SELECT id FROM users WHERE login = $1"
	var id int
	err := s.QueryRowContext(ctx, query, login).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil

}

// * Создание директории
func (s *Repo) MakeDirectory(ctx context.Context, name string, login string) (int, error) {

	ok, err := s.CheckDir(ctx, name, login)
	if ok {
		log.Println(" директория существует")
		return 0, err
	}

	uid, err := s.GetUserId(ctx, login)
	if err != nil {
		return 0, err
	}

	query := "INSERT INTO directories(name, id_owner) VALUES ($1, $2) RETURNING id"
	var id int
	err = s.QueryRowContext(ctx, query, name, uid).Scan(&id)
	if err != nil {
		log.Println(err)
		return 0, err
	}
	return id, nil

}

// * Удаление директории
func (s *Repo) DeleteDirectory(ctx context.Context, name string, login string) error {

	uid, err := s.GetUserId(ctx, login)
	if err != nil {
		return err
	}

	query := "DELETE FROM directories WHERE name = $1 AND id_owner = $2"
	_, err = s.QueryContext(ctx, query, name, uid)
	if err != nil {
		return err
	}
	return nil

}

// * Возврат всех директорий пользователя
func (s *Repo) GetDirectories(ctx context.Context, login string) ([]Dir, error) {

	uid, err := s.GetUserId(ctx, login)
	if err != nil {
		return nil, nil
	}

	query := `SELECT DISTINCT d.id, d.name, u.login AS owner_login
	FROM directories d
	JOIN users u ON d.id_owner = u.id
	WHERE d.id_owner = $1
   	OR d.id IN (SELECT directory_id FROM dir_users WHERE user_id = $2)
	ORDER BY d.id;`

	rows, err := s.QueryContext(ctx, query, uid, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dirs := make([]Dir, 0)
	for rows.Next() {
		var id int
		var name string
		var ownerLogin string

		if err := rows.Scan(&id, &name, &ownerLogin); err != nil {
			return nil, err
		}

		dirs = append(dirs, NewDir(id, name, ownerLogin))
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return dirs, nil
}

// * Проверка существования директории
func (s *Repo) CheckDir(ctx context.Context, name string, login string) (bool, error) {

	query := "SELECT EXISTS (SELECT 1 FROM directories WHERE name=$1 AND id_owner=$2)"
	id, err := s.GetUserId(ctx, login)
	if err != nil {
		return false, err
	}
	var ok bool
	err = s.QueryRowContext(ctx, query, name, id).Scan(&ok)
	if err != nil {
		return false, err
	}

	return ok, nil

}

// * Получение id директории
func (s *Repo) GetDirId(ctx context.Context, name string, login string) (int, error) {
	query := "SELECT id FROM directories WHERE name = $1 AND id_owner = $2"
	uid, err := s.GetUserId(ctx, login)
	if err != nil {
		return 0, err
	}
	var id int
	err = s.QueryRowContext(ctx, query, name, uid).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// * Добавление лекарства
func (s *Repo) AddMedicine(ctx context.Context, name string, date time.Time, cost float64, amount int, dirId int) (int, error) {
	query := `INSERT INTO medicines(name, date, cost, amount, id_directory)
              VALUES ($1, $2, $3, $4, $5) RETURNING id`
	var id int
	err := s.QueryRowContext(ctx, query, name, date, cost, amount, dirId).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// * Получение всех лекарств директории
func (s *Repo) GetMedicines(ctx context.Context, dirId int) ([]Medicine, error) {
	query := `SELECT id, name, date, cost, amount, id_directory
              FROM medicines
              WHERE id_directory = $1
              ORDER BY date DESC, id DESC`
	rows, err := s.QueryContext(ctx, query, dirId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	meds := make([]Medicine, 0)
	for rows.Next() {
		var m Medicine
		var cost sql.NullFloat64
		var amount sql.NullInt64
		var dt sql.NullTime
		if err := rows.Scan(&m.Id, &m.Name, &dt, &cost, &amount, &m.DirId); err != nil {
			return nil, err
		}
		if dt.Valid {
			m.Date = dt.Time
		} else {
			m.Date = time.Time{}
		}
		if cost.Valid {
			m.Cost = cost.Float64
		}
		if amount.Valid {
			m.Amount = int(amount.Int64)
		}
		meds = append(meds, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return meds, nil
}

// * Получение определенного лекарства
func (s *Repo) GetMedicineByID(ctx context.Context, medId int) (Medicine, error) {
	query := `SELECT id, name, date, cost, amount, id_directory FROM medicines WHERE id = $1`
	var m Medicine
	var cost sql.NullFloat64
	var amount sql.NullInt64
	var dt sql.NullTime
	err := s.QueryRowContext(ctx, query, medId).Scan(&m.Id, &m.Name, &dt, &cost, &amount, &m.DirId)
	if err != nil {
		return Medicine{}, err
	}
	if dt.Valid {
		m.Date = dt.Time
	}
	if cost.Valid {
		m.Cost = cost.Float64
	}
	if amount.Valid {
		m.Amount = int(amount.Int64)
	}
	return m, nil
}

// * Обновление определенного лекарства
func (s *Repo) UpdateMedicine(ctx context.Context, medId int, name string, date time.Time, cost float64, amount int) error {
	query := `UPDATE medicines SET name = $1, date = $2, cost = $3, amount = $4 WHERE id = $5`
	_, err := s.ExecContext(ctx, query, name, date, cost, amount, medId)
	return err
}

// * Удаление лекарства
func (s *Repo) DeleteMedicine(ctx context.Context, medId int) error {
	query := `DELETE FROM medicines WHERE id = $1`
	_, err := s.ExecContext(ctx, query, medId)
	return err
}

// * Проверка на принадлежность пользователя директории
func (s *Repo) IsUserInDirectory(ctx context.Context, login string, dirId int) (bool, error) {
	// Сначала получаем ID пользователя
	uid, err := s.GetUserId(ctx, login)
	if err != nil {
		return false, err
	}

	// Проверяем: пользователь либо владелец, либо участник
	query := `
        SELECT EXISTS(
            SELECT 1 FROM directories WHERE id = $1 AND id_owner = $2
        ) OR EXISTS(
            SELECT 1 FROM dir_users WHERE directory_id = $1 AND user_id = $2
        )
    `

	var exists bool
	err = s.QueryRowContext(ctx, query, dirId, uid).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// * Возврат обьекта структуры директории
func (s *Repo) GetDirectoryById(ctx context.Context, id int) (Dir, error) {
	query := `
        SELECT d.id, d.name, u.login
        FROM directories d
        JOIN users u ON d.id_owner = u.id
        WHERE d.id = $1
    `

	var dir Dir
	err := s.QueryRowContext(ctx, query, id).
		Scan(&dir.Id, &dir.Name, &dir.Owner)
	if err != nil {
		return Dir{}, err
	}
	return dir, nil
}

// * Привязка пользователя к аудитории
func (s *Repo) AddUserToDirectory(ctx context.Context, login string, dirId int) error {
	uid, err := s.GetUserId(ctx, login)
	if err != nil {
		return err
	}

	// Проверяем, что директория существует
	_, err = s.GetDirectoryById(ctx, dirId)
	if err != nil {
		return err
	}

	// Если владелец — добавлять не нужно
	queryOwner := `SELECT id_owner FROM directories WHERE id = $1`
	var ownerID int
	err = s.QueryRowContext(ctx, queryOwner, dirId).Scan(&ownerID)
	if err != nil {
		return err
	}
	if ownerID == uid {
		return nil // владелец всегда имеет доступ, запись не нужна
	}

	// Добавляем участника
	query := `
        INSERT INTO dir_users(directory_id, user_id)
        VALUES($1, $2)
        ON CONFLICT DO NOTHING
    `
	_, err = s.ExecContext(ctx, query, dirId, uid)
	return err
}

// * Удаление пользователя из директории
func (s *Repo) RemoveUserFromDirectory(ctx context.Context, login string, dirId int) error {
	// получаем id пользователя
	uid, err := s.GetUserId(ctx, login)
	if err != nil {
		// если пользователь не найден, пробрасываем ошибку
		return err
	}

	// проверяем, существует ли директория и получим id_owner
	var ownerID int
	err = s.QueryRowContext(ctx, `SELECT id_owner FROM directories WHERE id = $1`, dirId).Scan(&ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("directory not found")
		}
		return err
	}

	// не позволяем удалять владельца
	if ownerID == uid {
		return fmt.Errorf("cannot remove owner from directory")
	}

	// удаляем запись
	res, err := s.ExecContext(ctx, `DELETE FROM dir_users WHERE directory_id = $1 AND user_id = $2`, dirId, uid)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("user is not a member of directory")
	}
	return nil
}

// * Получение всех пользователей директории
func (s *Repo) GetUsersByDirectory(ctx context.Context, dirId int) ([]string, error) {

	query := `
		SELECT DISTINCT u.login
		FROM users u
		JOIN dir_users du ON u.id = du.user_id
		WHERE du.directory_id = $1

		UNION

		SELECT u2.login
		FROM directories d
		JOIN users u2 ON d.id_owner = u2.id
		WHERE d.id = $1

		ORDER BY login;
	`

	rows, err := s.QueryContext(ctx, query, dirId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]string, 0, 8)
	for rows.Next() {
		var login string
		if err := rows.Scan(&login); err != nil {
			return nil, err
		}
		users = append(users, login)
	}

	return users, rows.Err()
}
