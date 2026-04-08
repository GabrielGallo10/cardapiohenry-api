package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"henry-bebidas-api/internal/database"
	"henry-bebidas-api/internal/middleware"
	"github.com/jackc/pgx/v5"
)

type EnderecoRequest struct {
	Apelido     *string `json:"apelido"`
	Rua         string  `json:"rua"`
	Numero      string  `json:"numero"`
	Complemento string  `json:"complemento"`
	Bairro      string  `json:"bairro"`
	Cidade      string  `json:"cidade"`
	UF          string  `json:"uf"`
	CEP         string  `json:"cep"`
}

type EnderecoResponse struct {
	ID          int       `json:"id"`
	Apelido     *string   `json:"apelido"`
	Rua         string    `json:"rua"`
	Numero      string    `json:"numero"`
	Complemento string    `json:"complemento"`
	Bairro      string    `json:"bairro"`
	Cidade      string    `json:"cidade"`
	UF          string    `json:"uf"`
	CEP         string    `json:"cep"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func Enderecos(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserIDFromToken(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		ListarEnderecos(w, r, userID)
	case http.MethodPost:
		CriarEndereco(w, r, userID)
	default:
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
	}
}

func EnderecosByID(w http.ResponseWriter, r *http.Request) {
	idParam := r.PathValue("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		http.Error(w, "id invalido", http.StatusBadRequest)
		return
	}

	userID, err := middleware.GetUserIDFromToken(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		BuscarEndereco(w, r, userID, id)
	case http.MethodPut:
		AtualizarEndereco(w, r, userID, id)
	case http.MethodDelete:
		DeletarEndereco(w, r, userID, id)
	default:
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
	}
	
}

func ListarEnderecos(w http.ResponseWriter, r *http.Request, userID int64) {
	rows, err := database.Pool.Query(
		r.Context(),
		`SELECT id_endereco, apelido, rua, numero, complemento, bairro, cidade, uf, cep, created_at, updated_at
		FROM enderecos
		WHERE cd_usuario = $1
		ORDER BY id_endereco DESC`,
		userID,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	enderecos := make([]EnderecoResponse, 0)
	for rows.Next() {
		var item EnderecoResponse
		if err := rows.Scan(
			&item.ID,
			&item.Apelido,
			&item.Rua,
			&item.Numero,
			&item.Complemento,
			&item.Bairro,
			&item.Cidade,
			&item.UF,
			&item.CEP,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		enderecos = append(enderecos, item)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(enderecos); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func CriarEndereco(w http.ResponseWriter, r *http.Request, userID int64) {
	var req EnderecoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Dados inválidos", http.StatusBadRequest)
		return
	}

	_, err := database.Pool.Exec(
		r.Context(),
		`INSERT INTO enderecos (cd_usuario, apelido, rua, numero, complemento, bairro, cidade, uf, cep)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		userID,
		req.Apelido,
		req.Rua,
		req.Numero,
		req.Complemento,
		req.Bairro,
		req.Cidade,
		req.UF,
		req.CEP,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Endereço criado com sucesso"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func BuscarEndereco(w http.ResponseWriter, r *http.Request, userID int64, id int) {
	var endereco EnderecoResponse
	err := database.Pool.QueryRow(
		r.Context(),
		`SELECT id_endereco, apelido, rua, numero, complemento, bairro, cidade, uf, cep, created_at, updated_at
		FROM enderecos
		WHERE cd_usuario = $1 AND id_endereco = $2`,
		userID,
		id,
	).Scan(
		&endereco.ID,
		&endereco.Apelido,
		&endereco.Rua,
		&endereco.Numero,
		&endereco.Complemento,
		&endereco.Bairro,
		&endereco.Cidade,
		&endereco.UF,
		&endereco.CEP,
		&endereco.CreatedAt,
		&endereco.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Endereço não encontrado", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(endereco); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func AtualizarEndereco(w http.ResponseWriter, r *http.Request, userID int64, id int) {
	var req EnderecoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Dados inválidos", http.StatusBadRequest)
		return
	}
	
	result, err := database.Pool.Exec(
		r.Context(),
		`UPDATE enderecos SET apelido = $1, rua = $2, numero = $3, complemento = $4, bairro = $5, cidade = $6, uf = $7, cep = $8, updated_at = NOW() WHERE cd_usuario = $9 AND id_endereco = $10`,
		req.Apelido, req.Rua, req.Numero, req.Complemento, req.Bairro, req.Cidade, req.UF, req.CEP, userID, id,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if result.RowsAffected() == 0 {
		http.Error(w, "Endereço não encontrado", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Endereço atualizado com sucesso"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func DeletarEndereco(w http.ResponseWriter, r *http.Request, userID int64, id int) {
	result, err := database.Pool.Exec(
		r.Context(),
		`DELETE FROM enderecos WHERE cd_usuario = $1 AND id_endereco = $2`,
		userID, id,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if result.RowsAffected() == 0 {
		http.Error(w, "Endereço não encontrado", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Endereço deletado com sucesso"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}