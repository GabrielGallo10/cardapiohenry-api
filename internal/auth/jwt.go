package auth

import (
	"errors"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims são os dados gravados dentro do token (id do usuário, email, datas).
type Claims struct {
	jwt.RegisteredClaims
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
}

// GenerateToken cria um token JWT válido por expHours horas.
func GenerateToken(secret string, expHours int, userID int64, email string) (string, error) {
	if secret == "" {
		log.Printf("[JWT] GenerateToken: erro - JWT_SECRET está vazio")
		return "", errors.New("JWT secret não pode ser vazio")
	}

	now := time.Now()
	exp := now.Add(time.Duration(expHours) * time.Hour)
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		UserID: userID,
		Email:  email,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		log.Printf("[JWT] GenerateToken: erro ao assinar token user_id=%d: %v", userID, err)
		return "", err
	}
	log.Printf("[JWT] GenerateToken: OK user_id=%d email=%s expira=%s", userID, email, exp.Format("2006-01-02 15:04"))
	return signed, nil
}

// ValidateToken verifica o token e retorna os claims. Retorna erro se inválido ou expirado.
func ValidateToken(secret, tokenString string) (*Claims, error) {
	if secret == "" {
		log.Printf("[JWT] ValidateToken: JWT_SECRET vazio no servidor")
		return nil, errors.New("JWT secret não pode ser vazio")
	}
	if tokenString == "" {
		return nil, errors.New("token não informado")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("método de assinatura inválido")
		}
		return []byte(secret), nil
	})
	if err != nil {
		log.Printf("[JWT] ValidateToken: parse falhou - %v", err)
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		log.Printf("[JWT] ValidateToken: claims inválidos ou token expirado ok=%v valid=%v", ok, token.Valid)
		return nil, errors.New("token inválido")
	}
	return claims, nil
}
