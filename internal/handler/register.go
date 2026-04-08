package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"henry-bebidas-api/internal/database"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

type registerRequest struct {
	Name                 string `json:"name"`
	Email                string `json:"email"`
	Telefone             string `json:"tel"`
	Password             string `json:"password"`
	PasswordConfirmation string `json:"password_confirmation"`
}

func normalizeEmailRegister(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func isValidEmailRegister(emailNorm string) bool {
	if len(emailNorm) < 5 || len(emailNorm) > 320 {
		return false
	}
	at := strings.LastIndex(emailNorm, "@")
	if at <= 0 || at == len(emailNorm)-1 {
		return false
	}
	local := emailNorm[:at]
	domain := emailNorm[at+1:]
	if len(local) == 0 || len(domain) < 3 || !strings.Contains(domain, ".") {
		return false
	}
	return true
}

// Register cadastra um novo cliente. A senha é salva em hash (bcrypt).
func Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	var req registerRequest
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Dados inválidos", http.StatusBadRequest)
		return
	}

	if req.Password != req.PasswordConfirmation {
		http.Error(w, "Senhas não conferem", http.StatusBadRequest)
		return
	}

	emailNorm := normalizeEmailRegister(req.Email)
	if !isValidEmailRegister(emailNorm) {
		http.Error(w, "E-mail inválido", http.StatusBadRequest)
		return
	}
	emailStored := strings.TrimSpace(req.Email)

	var telB strings.Builder
	for _, r := range req.Telefone {
		if r >= '0' && r <= '9' {
			telB.WriteRune(r)
		}
	}
	tel := telB.String()
	if len(tel) < 10 {
		http.Error(w, "Telefone inválido", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Erro ao processar senha", http.StatusInternalServerError)
		return
	}

	_, err = database.Pool.Exec(
		r.Context(),
		`INSERT INTO usuarios (nome, telefone, email, senha, cargo)
		VALUES ($1, $2, $3, $4, $5)`,
		strings.TrimSpace(req.Name), tel, emailStored, string(hashedPassword), "client",
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			http.Error(w, "E-mail ou telefone já cadastrados", http.StatusConflict)
			return
		}
		http.Error(w, "Erro ao cadastrar usuário.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(map[string]string{
		"message": "Usuário cadastrado com sucesso",
	}); err != nil {
		http.Error(w, "Erro ao enviar resposta", http.StatusInternalServerError)
		return
	}
}
