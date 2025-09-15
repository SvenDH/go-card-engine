package server

import (
	"database/sql"
	"fmt"
)

type Repository struct {
	Db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			username TEXT,
			password TEXT
		);
	`)
	if err != nil {
		panic(err)
	}
	return &Repository{Db: db}
}

type User struct {
	Id       int64
	Name     string
	Username sql.NullString
	Password sql.NullString
}

func (repo *Repository) AddUser(name string) (*User, error) {
	res, err := repo.Db.Exec("INSERT INTO user(name) values(?)", name)
	if err != nil {
		return nil, fmt.Errorf("error in db execution: %w", err)
	}
	id, _ := res.LastInsertId()
	return &User{Id: id, Name: name}, nil
}

func (repo *Repository) SetPassword(user *User, password string) error {
	return repo.execWrap("UPDATE user SET password = ? WHERE id = ?", password, user.Id)
}

func (repo *Repository) FindUserById(id int) *User {
	row := repo.Db.QueryRow("SELECT id, name FROM user where id = ? LIMIT 1", id)
	var user User
	if err := row.Scan(&user.Id, &user.Username); err != nil {
		if err != sql.ErrNoRows {
			fmt.Printf("error in db execution: %v\n", err)
		}
		return nil
	}
	return &user
}

func (repo *Repository) FindUserByName(name string) *User {
	row := repo.Db.QueryRow("SELECT id, name, username, password FROM user where name = ? LIMIT 1", name)
	var user User
	if err := row.Scan(&user.Id, &user.Name, &user.Username, &user.Password); err != nil {
		if err != sql.ErrNoRows {
			fmt.Printf("error in db execution: %v\n", err)
		}
		return nil
	}
	return &user
}

func (repo *Repository) execWrap(query string, args ...any) error {
	if _, err := repo.Db.Exec(query, args...); err != nil {
		return fmt.Errorf("error in db execution: %w", err)
	}
	return nil
}