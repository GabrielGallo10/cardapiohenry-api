package handler

import (
	"encoding/json"
	"net/http"

	"henry-bebidas-api/internal/config"
	"henry-bebidas-api/internal/database"
	"henry-bebidas-api/internal/auth"
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

	// Busca o funcionário pelo telefone
	var idUsuario int
	var passwordHash string
	var tel string
	var cargo string
	err := database.Pool.QueryRow(
		r.Context(),
		`SELECT id_usuario, telefone, senha, cargo FROM usuarios WHERE telefone = $1`,
		req.Telefone,
	).Scan(&idUsuario, &tel, &passwordHash, &cargo)
	if err != nil {
		http.Error(w, "Telefone não encontrado!", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		http.Error(w, "Senha inválida!", http.StatusUnauthorized)
		return
	}

	cfg := config.LoadJWT()
	token, err := auth.GenerateToken(cfg.Secret, cfg.ExpirationHours, int64(idUsuario), tel)
	if err != nil {
		http.Error(w, "Erro ao gerar token", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(loginResponse{
		Token:     token,
		Cargo: cargo,
	}); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}