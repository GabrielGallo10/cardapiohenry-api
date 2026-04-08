package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"henry-bebidas-api/internal/auth"
	"henry-bebidas-api/internal/config"
	"henry-bebidas-api/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

// loginRequest: body do POST /login (telefone e senha).
type loginRequest struct {
	Telefone    string `json:"tel"`
	Password string `json:"password"`
}

// loginResponse: resposta de sucesso (token JWT e dados do usuário).
type loginResponse struct {
	Token     string `json:"token"`
	Cargo string `json:"cargo"`
}

// Login valida telefone e senha no banco e devolve um token JWT se estiver correto.
func Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	var req loginRequest
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Dados inválidos", http.StatusBadRequest)
		return
	}

	telInput := digitsOnlyPhone(req.Telefone)
	if len(telInput) < 10 {
		http.Error(w, "Telefone inválido", http.StatusBadRequest)
		return
	}

	var idUsuario int64
	var senha pgtype.Text
	var tel string
	var cargo pgtype.Text
	var err error
	for _, candidate := range phoneLookupCandidates(telInput) {
		err = database.Pool.QueryRow(
			r.Context(),
			`SELECT id_usuario, telefone, senha, cargo FROM usuarios WHERE telefone = $1`,
			candidate,
		).Scan(&idUsuario, &tel, &senha, &cargo)
		if err == nil {
			break
		}
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		log.Printf("[login] postgres: %v", err)
		http.Error(w, "Erro ao validar login", http.StatusInternalServerError)
		return
	}
	if err != nil {
		http.Error(w, "Telefone ou senha incorretos.", http.StatusUnauthorized)
		return
	}

	if !senha.Valid || senha.String == "" {
		http.Error(w, "Telefone ou senha incorretos.", http.StatusUnauthorized)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(senha.String), []byte(req.Password)); err != nil {
		http.Error(w, "Telefone ou senha incorretos.", http.StatusUnauthorized)
		return
	}

	cargoStr := "client"
	if cargo.Valid {
		cargoStr = cargo.String
	}

	cfg := config.LoadJWT()
	token, err := auth.GenerateToken(cfg.Secret, cfg.ExpirationHours, idUsuario, tel)
	if err != nil {
		http.Error(w, "Erro ao gerar token", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(loginResponse{
		Token: token,
		Cargo: cargoStr,
	}); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}