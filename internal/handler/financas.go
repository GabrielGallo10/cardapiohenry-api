package handler

import (
	"net/http"
	"cardapio-henry-api/internal/database"
	"encoding/json"
	"strconv"
	"time"
)

type FinancaRequest struct {
	Tipo string `json:"tipo"`
	Descricao string `json:"descricao"`
	Valor float64 `json:"valor"`
	Data time.Time `json:"data"`
}

type FinancaResponse struct {
	ID int `json:"id"`
	Tipo string `json:"tipo"`
	Descricao string `json:"descricao"`
	Valor float64 `json:"valor"`
	Data time.Time `json:"data"`
}

func Financas(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ListarFinancas(w, r)
	case http.MethodPost:
		CriarFinanca(w, r)
	default:
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
}

func FinancasByID(w http.ResponseWriter, r *http.Request) {
	idParam := r.PathValue("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		http.Error(w, "id invalido", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		AtualizarFinanca(w, r, id)
	case http.MethodDelete:
		DeletarFinanca(w, r, id)
	default:
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
}

func ListarFinancas(w http.ResponseWriter, r *http.Request) {
	rows, err := database.Pool.Query(
		r.Context(),
		`SELECT id_financa, tipo, descricao, valor, data_financa FROM financas`,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	financas := make([]FinancaResponse, 0)
	for rows.Next() {
		var financa FinancaResponse
		if err := rows.Scan(&financa.ID, &financa.Tipo, &financa.Descricao, &financa.Valor, &financa.Data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		financas = append(financas, financa)
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(financas); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func CriarFinanca(w http.ResponseWriter, r *http.Request) {
	var req FinancaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Dados inválidos", http.StatusBadRequest)
		return
	}
	
	_, err := database.Pool.Exec(
		r.Context(),
		`INSERT INTO financas (tipo, descricao, valor, data_financa) VALUES ($1, $2, $3, $4)`,
		req.Tipo, req.Descricao, req.Valor, req.Data,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Finança criada com sucesso"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func AtualizarFinanca(w http.ResponseWriter, r *http.Request, id int) {
	var req FinancaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Dados inválidos", http.StatusBadRequest)
		return
	}
	
	result, err := database.Pool.Exec(
		r.Context(),
		`UPDATE financas SET tipo = $1, descricao = $2, valor = $3, data_financa = $4 WHERE id_financa = $5`,
		req.Tipo, req.Descricao, req.Valor, req.Data, id,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if result.RowsAffected() == 0 {
		http.Error(w, "Finança não encontrada", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Finança atualizada com sucesso"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func DeletarFinanca(w http.ResponseWriter, r *http.Request, id int) {
	result, err := database.Pool.Exec(
		r.Context(),
		`DELETE FROM financas WHERE id_financa = $1`,
		id,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if result.RowsAffected() == 0 {
		http.Error(w, "Finança não encontrada", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Finança deletada com sucesso"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}