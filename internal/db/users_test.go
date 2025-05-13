package db

import (
	"testing"
)

// TestCreateUser проверяет создание нового пользователя
func TestCreateUser(t *testing.T) {
	// Инициализируем конфигурацию
	InitTest(t)

	defer CleanupDB()

	// Тестируем создание пользователя
	login := "testuser_create"
	password := "testpassword"

	user, err := CreateUser(login, password)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Проверяем, что пользователь создан с правильными параметрами
	if user.ID <= 0 {
		t.Errorf("Expected user ID > 0, got %d", user.ID)
	}

	if user.Login != login {
		t.Errorf("Expected login '%s', got '%s'", login, user.Login)
	}

	if user.PasswordHash == "" {
		t.Errorf("Expected non-empty password hash")
	}

	// Проверяем, что время создания установлено
	if user.CreatedAt.IsZero() {
		t.Errorf("Expected non-zero creation time")
	}

	// Проверяем, что не можем создать пользователя с тем же логином
	_, err = CreateUser(login, password)
	if err != ErrUserAlreadyExists {
		t.Errorf("Expected ErrUserAlreadyExists when creating duplicate user, got %v", err)
	}
}

// TestGetUserByID проверяет получение пользователя по ID
func TestGetUserByID(t *testing.T) {
	// Инициализируем конфигурацию
	InitTest(t)

	defer CleanupDB()

	// Создаем пользователя для тестирования
	login := "testuser_get_id"
	password := "testpassword"

	createdUser, err := CreateUser(login, password)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Тестируем получение пользователя по ID
	retrievedUser, err := GetUserByID(createdUser.ID)
	if err != nil {
		t.Errorf("Failed to get user by ID: %v", err)
		return
	}

	// Проверяем, что полученные данные совпадают с созданными
	if retrievedUser.ID != createdUser.ID {
		t.Errorf("Expected user ID %d, got %d", createdUser.ID, retrievedUser.ID)
	}

	if retrievedUser.Login != createdUser.Login {
		t.Errorf("Expected login '%s', got '%s'", createdUser.Login, retrievedUser.Login)
	}

	if retrievedUser.PasswordHash != createdUser.PasswordHash {
		t.Errorf("Expected password hash '%s', got '%s'", createdUser.PasswordHash, retrievedUser.PasswordHash)
	}

	// Проверяем, что CreatedAt установлен и в пределах разумного диапазона
	if retrievedUser.CreatedAt.IsZero() {
		t.Errorf("Expected non-zero creation time")
	}

	timeDiff := retrievedUser.CreatedAt.Sub(createdUser.CreatedAt).Seconds()
	if timeDiff < -1 || timeDiff > 1 {
		t.Errorf("Expected similar creation times, got diff of %f seconds", timeDiff)
	}

	// Тестируем получение несуществующего пользователя
	_, err = GetUserByID(999999)
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound when getting non-existent user, got %v", err)
	}
}

// TestGetUserByLogin проверяет получение пользователя по логину
func TestGetUserByLogin(t *testing.T) {
	// Инициализируем конфигурацию
	InitTest(t)

	defer CleanupDB()

	// Создаем пользователя для тестирования
	login := "testuser_get_login"
	password := "testpassword"

	createdUser, err := CreateUser(login, password)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Тестируем получение пользователя по логину
	retrievedUser, err := GetUserByLogin(login)
	if err != nil {
		t.Errorf("Failed to get user by login: %v", err)
		return
	}

	// Проверяем, что полученные данные совпадают с созданными
	if retrievedUser.ID != createdUser.ID {
		t.Errorf("Expected user ID %d, got %d", createdUser.ID, retrievedUser.ID)
	}

	if retrievedUser.Login != createdUser.Login {
		t.Errorf("Expected login '%s', got '%s'", createdUser.Login, retrievedUser.Login)
	}

	if retrievedUser.PasswordHash != createdUser.PasswordHash {
		t.Errorf("Expected password hash '%s', got '%s'", createdUser.PasswordHash, retrievedUser.PasswordHash)
	}

	// Тестируем получение несуществующего пользователя
	_, err = GetUserByLogin("nonexistent_user")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound when getting non-existent user, got %v", err)
	}
}

// TestAuthenticateUser проверяет аутентификацию пользователя
func TestAuthenticateUser(t *testing.T) {
	// Инициализируем конфигурацию
	InitTest(t)

	defer CleanupDB()

	// Создаем пользователя для тестирования
	login := "testuser_auth"
	password := "testpassword"

	createdUser, err := CreateUser(login, password)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Тестируем успешную аутентификацию
	authenticatedUser, err := AuthenticateUser(login, password)
	if err != nil {
		t.Errorf("Failed to authenticate user with correct credentials: %v", err)
		return
	}

	// Проверяем, что аутентифицированный пользователь совпадает с созданным
	if authenticatedUser.ID != createdUser.ID {
		t.Errorf("Expected user ID %d, got %d", createdUser.ID, authenticatedUser.ID)
	}

	if authenticatedUser.Login != createdUser.Login {
		t.Errorf("Expected login '%s', got '%s'", createdUser.Login, authenticatedUser.Login)
	}

	// Тестируем аутентификацию с неверным паролем
	_, err = AuthenticateUser(login, "wrong_password")
	if err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials with wrong password, got %v", err)
	}

	// Тестируем аутентификацию с несуществующим пользователем
	_, err = AuthenticateUser("nonexistent_user", password)
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound with non-existent user, got %v", err)
	}
}
