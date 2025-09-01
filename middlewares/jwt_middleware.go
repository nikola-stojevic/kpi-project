package middlewares

import (
	"context"
	"net/http"
	"strings"

	"kpiproject/utils"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type contextKey string

const UserContextKey contextKey = "user"

func JWTMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				utils.HandleMessageResponse(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				utils.HandleMessageResponse(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})

			if err != nil {
				utils.HandleMessageResponse(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			if claims, ok := token.Claims.(*Claims); ok && token.Valid {
				ctx := context.WithValue(r.Context(), UserContextKey, claims.Username)
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				utils.HandleMessageResponse(w, "Invalid token claims", http.StatusUnauthorized)
				return
			}
		})
	}
}

func GetUsernameFromContext(ctx context.Context) string {
	if username, ok := ctx.Value(UserContextKey).(string); ok {
		return username
	}
	return ""
}
