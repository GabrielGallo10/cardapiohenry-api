package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const passwordResetScope = "password_reset"

// PasswordResetClaims JWT de curta duração só para POST /password-reset/confirm.
type PasswordResetClaims struct {
	jwt.RegisteredClaims
	UserID int64  `json:"uid"`
	Scope  string `json:"scope"`
	Email  string `json:"email"`
}

// GeneratePasswordResetToken emite token após validar o código enviado por e-mail (exp em minutos).
func GeneratePasswordResetToken(secret string, expMinutes int, userID int64, emailNorm string) (string, error) {
	if secret == "" {
		return "", errors.New("JWT secret não pode ser vazio")
	}
	if expMinutes <= 0 {
		expMinutes = 15
	}
	now := time.Now()
	exp := now.Add(time.Duration(expMinutes) * time.Minute)
	claims := PasswordResetClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		UserID: userID,
		Scope:  passwordResetScope,
		Email:  emailNorm,
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

// ValidatePasswordResetToken valida o token e devolve utilizador + e-mail normalizado esperados.
func ValidatePasswordResetToken(secret, tokenString string) (userID int64, emailNorm string, err error) {
	if secret == "" || tokenString == "" {
		return 0, "", errors.New("token inválido")
	}
	tok, err := jwt.ParseWithClaims(tokenString, &PasswordResetClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("método de assinatura inválido")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return 0, "", err
	}
	claims, ok := tok.Claims.(*PasswordResetClaims)
	if !ok || !tok.Valid || claims.Scope != passwordResetScope {
		return 0, "", errors.New("token inválido")
	}
	return claims.UserID, claims.Email, nil
}
