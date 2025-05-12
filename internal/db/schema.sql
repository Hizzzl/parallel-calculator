-- Таблица пользователей
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    login TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Таблица выражений
CREATE TABLE IF NOT EXISTS expressions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    original_expression TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    result NUMERIC DEFAULT NULL,
    error_message TEXT DEFAULT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Таблица операций
CREATE TABLE IF NOT EXISTS operations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    expression_id INTEGER NOT NULL,
    parent_operation_id INTEGER,
    child_position TEXT,
    left_value NUMERIC,
    right_value NUMERIC,
    operator TEXT,
    status TEXT DEFAULT 'pending',
    result NUMERIC DEFAULT NULL,
    error_message TEXT DEFAULT NULL,
    is_root_expression BOOLEAN NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (expression_id) REFERENCES expressions(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_operation_id) REFERENCES operations(id) ON DELETE CASCADE,
    CHECK (status IN ('pending', 'ready', 'processing', 'completed', 'error', 'canceled')),
    CHECK (child_position IN ('left', 'right', NULL))
);

-- Вставляем тестового пользователя, если таблица пуста
INSERT OR IGNORE INTO users (login, password_hash) 
VALUES ('test_user', '$2a$10$KrZ.f3n7AxVgAvUaK7DMW.Dt8e8IZOxvNrd9PQAYYZRw0bHdU4bUO'); -- пароль: password
