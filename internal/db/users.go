package db

import (
	"database/sql"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// CreateUser создает нового пользователя в базе данных
func CreateUser(login, password string) (*User, error) {
	DbMutex.Lock()
	defer DbMutex.Unlock()
	// Проверяем, что пользователь уже существует
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM users WHERE login = ?", login).Scan(&count)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrUserAlreadyExists
	}

	// Хешируем пароль
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Добавляем нового пользователя
	result, err := DB.Exec(
		"INSERT INTO users (login, password_hash) VALUES (?, ?)",
		login, string(passwordHash),
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &User{
		ID:           id,
		Login:        login,
		PasswordHash: string(passwordHash),
		CreatedAt:    time.Now(),
	}, nil
}

// GetUserByID получает пользователя по ID
func GetUserByID(id int64) (*User, error) {
	DbMutex.Lock()
	defer DbMutex.Unlock()
	var user User
	var createdAtStr string

	err := DB.QueryRow(
		"SELECT id, login, password_hash, created_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Login, &user.PasswordHash, &createdAtStr)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	user.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByLogin получает пользователя по логину
func GetUserByLogin(login string) (*User, error) {
	DbMutex.Lock()
	defer DbMutex.Unlock()
	var user User
	var createdAtStr string

	err := DB.QueryRow(
		"SELECT id, login, password_hash, created_at FROM users WHERE login = ?",
		login,
	).Scan(&user.ID, &user.Login, &user.PasswordHash, &createdAtStr)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	user.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// AuthenticateUser проверяет, действительны ли предоставленные учетные данные
func AuthenticateUser(login, password string) (*User, error) {
	user, err := GetUserByLogin(login)
	if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}
