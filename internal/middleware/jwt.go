package middleware

import (
	"errors"
	"net/http"
	"strings"

	"henry-bebidas-api/internal/auth"
	"henry-bebidas-api/internal/config"
	"henry-bebidas-api/internal/database"
	"encoding/json"
	"log"
	"context"
)

// ContextKeyClaims é a chave usada para armazenar os claims JWT no context da requisição.
type ContextKeyClaims struct{}

// jwtErrorResponse formato JSON para respostas 401 do middleware JWT.
type jwtErrorResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func writeJWTError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(jwtErrorResponse{Message: message, Code: status})
}

// JWT valida o header "Authorization: Bearer <token>" e coloca os claims no context.
// Se não houver token ou for inválido, responde 401 com JSON.
func JWT(next http.Handler) http.Handler {
	cfg := config.LoadJWT()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		const prefix = "Bearer "

		if authHeader == "" {
			log.Printf("[JWT] %s %s | 401: header Authorization ausente", r.Method, r.URL.Path)
			writeJWTError(w, "Token não informado. Faça login novamente.", http.StatusUnauthorized)
			return
		}
		if !strings.HasPrefix(authHeader, prefix) {
			log.Printf("[JWT] %s %s | 401: Authorization sem 'Bearer ' (valor recebido: %q)", r.Method, r.URL.Path, truncateForLog(authHeader, 50))
			writeJWTError(w, "Token não informado. Faça login novamente.", http.StatusUnauthorized)
			return
		}
		tokenString := strings.TrimSpace(authHeader[len(prefix):])
		if tokenString == "" {
			log.Printf("[JWT] %s %s | 401: token vazio após prefixo Bearer", r.Method, r.URL.Path)
			writeJWTError(w, "Token não informado. Faça login novamente.", http.StatusUnauthorized)
			return
		}

		claims, err := auth.ValidateToken(cfg.Secret, tokenString)
		if err != nil {
			log.Printf("[JWT] %s %s | 401: validação falhou: %v | token (início): %s", r.Method, r.URL.Path, err, truncateForLog(tokenString, 30))
			writeJWTError(w, "Token inválido ou expirado. Faça login novamente.", http.StatusUnauthorized)
			return
		}

		log.Printf("[JWT] %s %s | OK: user_id=%d email=%s", r.Method, r.URL.Path, claims.UserID, claims.Email)
		ctx := context.WithValue(r.Context(), ContextKeyClaims{}, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func truncateForLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// GetClaims retorna os claims JWT do context da requisição, ou nil se não houver.
func GetClaims(ctx context.Context) *auth.Claims {
	c, _ := ctx.Value(ContextKeyClaims{}).(*auth.Claims)
	return c
}

func GetUserIDFromToken(r *http.Request) (int64, error) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader == "" {
		return 0, errors.New("token não informado")
	}

	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if token == authHeader || token == "" {
		return 0, errors.New("header Authorization inválido")
	}

	jwtCfg := config.LoadJWT()
	claims, err := auth.ValidateToken(jwtCfg.Secret, token)
	if err != nil {
		return 0, errors.New("token inválido")
	}

	if claims.UserID <= 0 {
		return 0, errors.New("token sem user_id")
	}

	return claims.UserID, nil
}

func IsAdmin(ctx context.Context, userID int64) (bool, error) {
	var cargo string
	err := database.Pool.QueryRow(
		ctx,
		`SELECT cargo FROM usuarios WHERE id_usuario = $1`,
		userID,
	).Scan(&cargo)
	if err != nil {
		return false, err
	}

	return cargo == "admin", nil
}
