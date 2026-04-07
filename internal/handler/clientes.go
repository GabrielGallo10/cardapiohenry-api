package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"cardapio-henry-api/internal/database"
)

type ClienteResponse struct {
	Nome string `json:"nome"`
	Telefone string `json:"telefone"`
}

func Clientes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	pesquisa := strings.TrimSpace(r.URL.Query().Get("pesquisa"))
	pesquisaLike := "%" + pesquisa + "%"
	
	rows, err := database.Pool.Query(
		r.Context(),
		`SELECT nome, telefone
		FROM usuarios
		WHERE cargo = 'client' AND ($1 = '%%' OR nome ILIKE $1 OR telefone ILIKE $1)`,
		pesquisaLike,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer rows.Close()
	clientes := make([]ClienteResponse, 0)
	for rows.Next() {
		var cliente ClienteResponse
		if err := rows.Scan(&cliente.Nome, &cliente.Telefone); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		clientes = append(clientes, cliente)
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(clientes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}