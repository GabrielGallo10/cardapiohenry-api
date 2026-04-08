package handler

import (
	"net/http"
	"henry-bebidas-api/internal/database"
	"encoding/json"
	"strconv"
)

type CategoriaRequest struct {
	Nome string `json:"name"`
}

type CategoriaResponse struct {
	ID int `json:"id"`
	Nome string `json:"name"`
}

func Categorias(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ListarCategorias(w, r)
	case http.MethodPost:
		CriarCategoria(w, r)
	default:
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
	}
}

func ListarCategorias(w http.ResponseWriter, r *http.Request) {
	rows, err := database.Pool.Query(
		r.Context(),
		`SELECT id_categoria, nome FROM categorias`,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	
	categorias := make([]CategoriaResponse, 0)
	for rows.Next() {
		var categoria CategoriaResponse
		if err := rows.Scan(&categoria.ID, &categoria.Nome); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		categorias = append(categorias, categoria)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(categorias); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func CriarCategoria(w http.ResponseWriter, r *http.Request) {
	var req CategoriaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Dados inválidos", http.StatusBadRequest)
		return
	}
	
	_, err := database.Pool.Exec(
		r.Context(),
		`INSERT INTO categorias (nome) VALUES ($1)`,
		req.Nome,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Categoria criada com sucesso"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func DeletarCategoria(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	
	idParam := r.PathValue("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		http.Error(w, "id invalido", http.StatusBadRequest)
		return
	}
	
	_, err = database.Pool.Exec(
		r.Context(),
		`DELETE FROM categorias WHERE id_categoria = $1`,
		id,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Categoria deletada com sucesso"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}