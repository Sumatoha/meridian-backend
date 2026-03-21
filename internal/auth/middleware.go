package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey string

const (
	userIDKey    contextKey = "user_id"
	supabaseKey  contextKey = "supabase_user_id"
)

type Middleware struct {
	jwtSecret []byte
}

func NewMiddleware(jwtSecret string) *Middleware {
	return &Middleware{jwtSecret: []byte(jwtSecret)}
}

func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":{"code":"unauthorized","message":"missing authorization header"}}`, http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader {
			http.Error(w, `{"error":{"code":"unauthorized","message":"invalid authorization format"}}`, http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return m.jwtSecret, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, `{"error":{"code":"unauthorized","message":"invalid or expired token"}}`, http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, `{"error":{"code":"unauthorized","message":"invalid token claims"}}`, http.StatusUnauthorized)
			return
		}

		sub, ok := claims["sub"].(string)
		if !ok || sub == "" {
			http.Error(w, `{"error":{"code":"unauthorized","message":"missing subject in token"}}`, http.StatusUnauthorized)
			return
		}

		supabaseUID, err := uuid.Parse(sub)
		if err != nil {
			http.Error(w, `{"error":{"code":"unauthorized","message":"invalid subject format"}}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), supabaseKey, supabaseUID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func SupabaseUserID(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(supabaseKey).(uuid.UUID)
	return id
}

func SetUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

func UserID(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(userIDKey).(uuid.UUID)
	return id
}
