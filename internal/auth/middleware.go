package auth

import (
	"context"
	"net/http"
)

// Ключ контекста для пользователя
type contextKey string

const UserContextKey contextKey = "user"

// AuthMiddleware проверяет JWT токен в заголовке Authorization
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := ExtractTokenFromHeader(r)
		if err != nil {
			http.Error(w, "Не авторизован: "+err.Error(), http.StatusUnauthorized)
			return
		}

		claims, err := ValidateToken(tokenString)
		if err != nil {
			http.Error(w, "Не авторизован: "+err.Error(), http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, claims)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserFromContext извлекает пользовательские утверждения из контекста
func GetUserFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(UserContextKey).(*Claims)
	return claims, ok
}

// RequireAuth проверяет, аутентифицирован ли запрос, и возвращает ID пользователя
func RequireAuth(r *http.Request) (int64, error) {
	claims, ok := GetUserFromContext(r.Context())
	if !ok {
		return 0, ErrInvalidToken
	}
	return claims.UserID, nil
}
