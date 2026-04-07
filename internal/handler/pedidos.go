package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"
	"cardapio-henry-api/internal/middleware"
	"cardapio-henry-api/internal/database"
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

type PedidoRequest struct {
	NomeCliente     string  `json:"nome_cliente"`
	TelefoneCliente string  `json:"telefone_cliente"`
	IDEndereco      int     `json:"id_endereco"`
	FormaPagamento  string  `json:"forma_pagamento"`
	Observacoes     *string `json:"observacoes"`
	ItemsPedido   []ItemsPedido   `json:"items_pedido"`
}

type ItemsPedido struct {
	IDProduto int `json:"id_produto"`
	Quantidade int `json:"quantidade"`
}

type PedidoResponse struct {
	ID int `json:"id"`
	NomeCliente string `json:"nome_cliente"`
	TelefoneCliente string `json:"telefone_cliente"`
	DataHoraPedido time.Time `json:"data_hora_pedido"`
	ApelidoEndereco *string `json:"apelido_endereco"`
	RuaEndereco string `json:"rua_endereco"`
	NumeroEndereco string `json:"numero_endereco"`
	ComplementoEndereco string `json:"complemento_endereco"`
	BairroEndereco string `json:"bairro_endereco"`
	CidadeEndereco string `json:"cidade_endereco"`
	UFEndereco string `json:"uf_endereco"`
	CEPEndereco string `json:"cep_endereco"`
	StatusPedido string `json:"status_pedido"`
	ValorTotal float64 `json:"valor_total"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ItemsPedidoByID struct {
	NomeProduto string `json:"nome_produto"`
	Preco float64 `json:"preco"`
	Quantidade int `json:"quantidade"`
}

type PedidoResponseByID struct {
	ID int `json:"id"`
	NomeCliente string `json:"nome_cliente"`
	TelefoneCliente string `json:"telefone_cliente"`
	DataHoraPedido time.Time `json:"data_hora_pedido"`
	ApelidoEndereco *string `json:"apelido_endereco"`
	RuaEndereco string `json:"rua_endereco"`
	NumeroEndereco string `json:"numero_endereco"`
	ComplementoEndereco string `json:"complemento_endereco"`
	BairroEndereco string `json:"bairro_endereco"`
	CidadeEndereco string `json:"cidade_endereco"`
	UFEndereco string `json:"uf_endereco"`
	CEPEndereco string `json:"cep_endereco"`
	StatusPedido string `json:"status_pedido"`
	ValorTotal float64 `json:"valor_total"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ItemsPedido []ItemsPedidoByID `json:"items_pedido"`
}

type AtualizarStatusPedidoRequest struct {
	StatusPedido string `json:"status_pedido"`
}

func Pedidos(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserIDFromToken(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		ListarPedidos(w, r, userID)
	case http.MethodPost:
		CriarPedido(w, r, userID)
	case http.MethodDelete:
		DeletarPedidos(w, r, userID)
	default:
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
	}
}

func PedidosByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	userID, err := middleware.GetUserIDFromToken(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		http.Error(w, "ID do pedido inválido", http.StatusBadRequest)
		return
	}

	var pedido PedidoResponseByID
	err = database.Pool.QueryRow(
		r.Context(),
		`SELECT p.id_pedido, p.nome, p.telefone, p.data_horario, e.apelido, e.rua, e.numero, e.complemento, e.bairro, e.cidade, e.uf, e.cep, p.status_pedido, p.valor_total, p.created_at, p.updated_at
		FROM pedidos p
		LEFT JOIN enderecos e ON e.id_endereco = p.cd_endereco
		WHERE p.id_pedido = $1 AND p.cd_usuario = $2`,
		id, userID,
	).Scan(
		&pedido.ID,
		&pedido.NomeCliente,
		&pedido.TelefoneCliente,
		&pedido.DataHoraPedido,
		&pedido.ApelidoEndereco,
		&pedido.RuaEndereco,
		&pedido.NumeroEndereco,
		&pedido.ComplementoEndereco,
		&pedido.BairroEndereco,
		&pedido.CidadeEndereco,
		&pedido.UFEndereco,
		&pedido.CEPEndereco,
		&pedido.StatusPedido,
		&pedido.ValorTotal,
		&pedido.CreatedAt,
		&pedido.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Pedido não encontrado", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rows, err := database.Pool.Query(
		r.Context(),
		`SELECT p.nome, p.preco, ip.quantidade
		FROM itens_pedido ip
		INNER JOIN produtos p ON p.id_produto = ip.cd_produto
		WHERE ip.cd_pedido = $1`,
		id,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	pedido.ItemsPedido = make([]ItemsPedidoByID, 0)
	for rows.Next() {
		var item ItemsPedidoByID
		if err := rows.Scan(&item.NomeProduto, &item.Preco, &item.Quantidade); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		pedido.ItemsPedido = append(pedido.ItemsPedido, item)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(pedido); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func ListarPedidos(w http.ResponseWriter, r *http.Request, userID int64) {
	rows, err := database.Pool.Query(
		r.Context(),
		`SELECT p.id_pedido, p.nome, p.telefone, p.data_horario, e.apelido, e.rua, e.numero, e.complemento, e.bairro, e.cidade, e.uf, e.cep, p.status_pedido, p.valor_total, p.created_at, p.updated_at
		FROM pedidos p
		LEFT JOIN enderecos e ON e.id_endereco = p.cd_endereco
		WHERE cd_usuario = $1
		ORDER BY id_pedido DESC`,
		userID,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	pedidos := make([]PedidoResponse, 0)
	for rows.Next() {
		var pedido PedidoResponse
		err := rows.Scan(
			&pedido.ID,
			&pedido.NomeCliente,
			&pedido.TelefoneCliente,
			&pedido.DataHoraPedido,
			&pedido.ApelidoEndereco,
			&pedido.RuaEndereco,
			&pedido.NumeroEndereco,
			&pedido.ComplementoEndereco,
			&pedido.BairroEndereco,
			&pedido.CidadeEndereco,
			&pedido.UFEndereco,
			&pedido.CEPEndereco,
			&pedido.StatusPedido,
			&pedido.ValorTotal,
			&pedido.CreatedAt,
			&pedido.UpdatedAt,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		pedidos = append(pedidos, pedido)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(pedidos); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func CriarPedido(w http.ResponseWriter, r *http.Request, userID int64) {
	var pedido PedidoRequest
	err := json.NewDecoder(r.Body).Decode(&pedido)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if pedido.IDEndereco <= 0 || len(pedido.ItemsPedido) == 0 {
		http.Error(w, "id_endereco e items_pedido são obrigatórios", http.StatusBadRequest)
		return
	}

	tx, err := database.Pool.Begin(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	var enderecoExiste bool
	err = tx.QueryRow(
		r.Context(),
		`SELECT EXISTS(
			SELECT 1
			FROM enderecos
			WHERE id_endereco = $1 AND cd_usuario = $2
		)`,
		pedido.IDEndereco,
		userID,
	).Scan(&enderecoExiste)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !enderecoExiste {
		http.Error(w, "endereço não encontrado para este usuário", http.StatusNotFound)
		return
	}

	valorTotal := 0.0
	for _, item := range pedido.ItemsPedido {
		if item.IDProduto <= 0 || item.Quantidade <= 0 {
			http.Error(w, "itens inválidos", http.StatusBadRequest)
			return
		}

		var preco float64
		err = tx.QueryRow(
			r.Context(),
			`SELECT preco FROM produtos WHERE id_produto = $1`,
			item.IDProduto,
		).Scan(&preco)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				http.Error(w, "produto não encontrado", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		valorTotal += preco * float64(item.Quantidade)
	}

	var idPedido int
	err = tx.QueryRow(
		r.Context(),
		`INSERT INTO pedidos (cd_usuario, cd_endereco, nome, telefone, valor_total, metodo_pagamento, observacoes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id_pedido`,
		userID,
		pedido.IDEndereco,
		pedido.NomeCliente,
		pedido.TelefoneCliente,
		valorTotal,
		pedido.FormaPagamento,
		pedido.Observacoes,
	).Scan(&idPedido)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, item := range pedido.ItemsPedido {
		_, err = tx.Exec(
			r.Context(),
			`INSERT INTO itens_pedido (cd_pedido, cd_produto, quantidade)
			VALUES ($1, $2, $3)`,
			idPedido,
			item.IDProduto,
			item.Quantidade,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"message":     "Pedido criado com sucesso.",
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func DeletarPedidos(w http.ResponseWriter, r *http.Request, _ int64) {
	result, err := database.Pool.Exec(
		r.Context(),
		`DELETE FROM pedidos`,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"message":           "Pedidos deletados com sucesso.",
		"pedidos_removidos": result.RowsAffected(),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func AtualizarStatusPedido(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || id <= 0 {
		http.Error(w, "ID do pedido inválido", http.StatusBadRequest)
		return
	}

	var atualizarStatusPedidoRequest AtualizarStatusPedidoRequest
	err = json.NewDecoder(r.Body).Decode(&atualizarStatusPedidoRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID, err := middleware.GetUserIDFromToken(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	_, err = database.Pool.Exec(
		r.Context(),
		`UPDATE pedidos SET status_pedido = $1 WHERE id_pedido = $2 AND cd_usuario = $3`,
		atualizarStatusPedidoRequest.StatusPedido,
		id,
		userID,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"message": "Status do pedido atualizado com sucesso.",
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}