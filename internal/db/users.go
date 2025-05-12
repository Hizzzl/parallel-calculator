package db

import (
	"database/sql"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Ошибки
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// CreateUser создает нового пользователя в базе данных
func CreateUser(login, password string) (*User, error) {
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

	// Возвращаем созданного пользователя
	return &User{
		ID:           id,
		Login:        login,
		PasswordHash: string(passwordHash),
		CreatedAt:    time.Now(),
	}, nil
}

// GetUserByID получает пользователя по ID
func GetUserByID(id int64) (*User, error) {
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

	// Пробуем различные форматы даты для парсинга
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.999999999Z07:00", // RFC3339
		"2006-01-02T15:04:05",
		time.RFC3339,
	}

	var parseErr error
	for _, format := range formats {
		user.CreatedAt, parseErr = time.Parse(format, createdAtStr)
		if parseErr == nil {
			break // Успешно разобрали дату
		}
	}

	// Если не удалось разобрать дату, устанавливаем текущее время
	if parseErr != nil {
		user.CreatedAt = time.Now()
	}

	return &user, nil
}

// GetUserByLogin получает пользователя по логину
func GetUserByLogin(login string) (*User, error) {
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

	// Пробуем различные форматы даты для парсинга
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.999999999Z07:00", // RFC3339
		"2006-01-02T15:04:05",
		time.RFC3339,
	}

	var parseErr error
	for _, format := range formats {
		user.CreatedAt, parseErr = time.Parse(format, createdAtStr)
		if parseErr == nil {
			break // Успешно разобрали дату
		}
	}

	// Если не удалось разобрать дату, устанавливаем текущее время
	if parseErr != nil {
		user.CreatedAt = time.Now()
	}

	return &user, nil
}

// AuthenticateUser проверяет, действительны ли предоставленные учетные данные
func AuthenticateUser(login, password string) (*User, error) {
	user, err := GetUserByLogin(login)
	if err != nil {
		return nil, err
	}

	// Сравниваем хешированный пароль с предоставленным паролем
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}
