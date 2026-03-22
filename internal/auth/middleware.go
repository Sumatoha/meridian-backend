package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey string

const (
	userIDKey   contextKey = "user_id"
	supabaseKey contextKey = "supabase_user_id"
)

type Middleware struct {
	jwtSecret []byte
}

func NewMiddleware(jwtSecret string) *Middleware {
	// Trim whitespace/newlines — common env var copy-paste issue
	cleaned := strings.TrimSpace(jwtSecret)
	if len(cleaned) != len(jwtSecret) {
		slog.Warn("jwt secret had leading/trailing whitespace — trimmed",
			slog.Int("original_len", len(jwtSecret)),
			slog.Int("cleaned_len", len(cleaned)),
		)
	}
	slog.Info("auth middleware initialized",
		slog.Int("secret_len", len(cleaned)),
		slog.String("secret_prefix", safePrefix(cleaned, 4)),
	)
	return &Middleware{jwtSecret: []byte(cleaned)}
}

func safePrefix(s string, n int) string {
	if len(s) <= n {
		return "***"
	}
	return s[:n] + "..."
}

func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			slog.Warn("auth: missing authorization header",
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
			)
			http.Error(w, `{"error":{"code":"unauthorized","message":"missing authorization header"}}`, http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader {
			slog.Warn("auth: invalid authorization format (missing Bearer prefix)",
				slog.String("path", r.URL.Path),
			)
			http.Error(w, `{"error":{"code":"unauthorized","message":"invalid authorization format"}}`, http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				slog.Warn("auth: unexpected signing method",
					slog.String("alg", token.Header["alg"].(string)),
				)
				return nil, jwt.ErrSignatureInvalid
			}
			return m.jwtSecret, nil
		})
		if err != nil || !token.Valid {
			errMsg := "unknown"
			if err != nil {
				errMsg = err.Error()
			}
			slog.Warn("auth: jwt verification failed",
				slog.String("error", errMsg),
				slog.String("path", r.URL.Path),
				slog.Int("secret_len", len(m.jwtSecret)),
				slog.Int("token_len", len(tokenStr)),
			)
			http.Error(w, `{"error":{"code":"unauthorized","message":"invalid or expired token"}}`, http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			slog.Warn("auth: invalid token claims type", slog.String("path", r.URL.Path))
			http.Error(w, `{"error":{"code":"unauthorized","message":"invalid token claims"}}`, http.StatusUnauthorized)
			return
		}

		sub, ok := claims["sub"].(string)
		if !ok || sub == "" {
			slog.Warn("auth: missing subject in token", slog.String("path", r.URL.Path))
			http.Error(w, `{"error":{"code":"unauthorized","message":"missing subject in token"}}`, http.StatusUnauthorized)
			return
		}

		supabaseUID, err := uuid.Parse(sub)
		if err != nil {
			slog.Warn("auth: invalid subject format",
				slog.String("sub", sub),
				slog.String("path", r.URL.Path),
			)
			http.Error(w, `{"error":{"code":"unauthorized","message":"invalid subject format"}}`, http.StatusUnauthorized)
			return
		}

		slog.Debug("auth: authenticated",
			slog.String("uid", supabaseUID.String()),
			slog.String("path", r.URL.Path),
		)

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
