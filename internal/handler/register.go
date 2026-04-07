package handler

import (
	"encoding/json"
	"net/http"

	"cardapio-henry-api/internal/database"
	"golang.org/x/crypto/bcrypt"
)

type registerRequest struct {
	Name                 string `json:"name"`
	Telefone             string `json:"tel"`
	Password             string `json:"password"`
	PasswordConfirmation string `json:"password_confirmation"`
	Cargo string `json:"cargo"`
}

// Register cadastra um novo funcionário. A senha é salva em hash (bcrypt).
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

	// Converte a senha em hash para não armazenar texto puro
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Erro ao processar senha", http.StatusInternalServerError)
		return
	}

	// Salva no banco; telefone é único
	_, err = database.Pool.Exec(
		r.Context(),
		`INSERT INTO usuarios (nome, telefone, senha, cargo)
		VALUES ($1, $2, $3, $4)`,
		req.Name, req.Telefone, string(hashedPassword), req.Cargo,
	)
	if err != nil {
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
