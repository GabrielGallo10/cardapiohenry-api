package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"cardapio-henry-api/internal/database"
	"github.com/jackc/pgx/v5"
)

type ProdutoRequest struct {
	IDCategoria int `json:"id_categoria"`
	Nome string `json:"nome"`
	Descricao string `json:"descricao"`
	Preco float64 `json:"preco"`
	Disponivel bool `json:"disponivel"`
	URLImagem string `json:"url_imagem"`
}

type ProdutosResponse struct {
	ID int `json:"id"`
	NomeCategoria string `json:"nome_categoria"`
	Nome string `json:"nome"`
	Preco float64 `json:"preco"`
	Disponivel bool `json:"disponivel"`
	URLImagem string `json:"url_imagem"`
}

type ProdutoResponse struct {
	IDCategoria int `json:"id_categoria"`
	Nome string `json:"nome"`
	Preco float64 `json:"preco"`
	Descricao string `json:"descricao"`
	Disponivel bool `json:"disponivel"`
	URLImagem string `json:"url_imagem"`
}

func Produtos(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ListarProdutos(w, r)
	case http.MethodPost:
		CriarProdutos(w, r)
	default:
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
	}
}

func ProdutosByID(w http.ResponseWriter, r *http.Request) {
	idParam := r.PathValue("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		http.Error(w, "id invalido", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		BuscarProduto(w, r, id)
	case http.MethodPut:
		AtualizarProduto(w, r, id)
	case http.MethodDelete:
		DeletarProduto(w, r, id)
	default:
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
	}
}

func ListarProdutos(w http.ResponseWriter, r *http.Request) {
	rows, err := database.Pool.Query(
		r.Context(), 
		`SELECT p.id_produto, c.nome as nome_categoria, p.nome, p.preco, p.disponivel, p.url_imagem 
		FROM produtos p 
		LEFT JOIN categorias c ON p.cd_categoria = c.id_categoria`,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	produtos := make([]ProdutosResponse, 0)
	for rows.Next() {
		var produto ProdutosResponse
		if err := rows.Scan(&produto.ID, &produto.NomeCategoria, &produto.Nome, &produto.Preco, &produto.Disponivel, &produto.URLImagem); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		produtos = append(produtos, produto)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(produtos); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func CriarProdutos(w http.ResponseWriter, r *http.Request) {
	var req ProdutoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Dados inválidos", http.StatusBadRequest)
		return
	}

	_, err := database.Pool.Exec(
		r.Context(),
		`INSERT INTO produtos (cd_categoria, nome, descricao, preco, disponivel, url_imagem)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		req.IDCategoria, req.Nome, req.Descricao, req.Preco, req.Disponivel, req.URLImagem,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Produto criado com sucesso"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func BuscarProduto(w http.ResponseWriter, r *http.Request, id int) {
	var produto ProdutoResponse
	err := database.Pool.QueryRow(
		r.Context(),
		`SELECT c.id_categoria, p.nome, p.preco, p.descricao, p.disponivel, p.url_imagem
		FROM produtos p
		LEFT JOIN categorias c ON p.cd_categoria = c.id_categoria
		WHERE p.id_produto = $1`,
		id,
	).Scan(&produto.IDCategoria, &produto.Nome, &produto.Preco, &produto.Descricao, &produto.Disponivel, &produto.URLImagem)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Produto não encontrado", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(produto); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func AtualizarProduto(w http.ResponseWriter, r *http.Request, id int) {
	var req ProdutoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Dados inválidos", http.StatusBadRequest)
		return
	}

	_, err := database.Pool.Exec(
		r.Context(),
		`UPDATE produtos SET cd_categoria = $1, nome = $2, descricao = $3, preco = $4, disponivel = $5, url_imagem = $6
		WHERE id_produto = $7`,
		req.IDCategoria, req.Nome, req.Descricao, req.Preco, req.Disponivel, req.URLImagem, id,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Produto atualizado com sucesso"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func DeletarProduto(w http.ResponseWriter, r *http.Request, id int) {
	_, err := database.Pool.Exec(
		r.Context(),
		`DELETE FROM produtos WHERE id_produto = $1`,
		id,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Produto deletado com sucesso"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}