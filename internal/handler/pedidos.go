package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"henry-bebidas-api/internal/database"
	"henry-bebidas-api/internal/middleware"
	"henry-bebidas-api/internal/realtime"

	"github.com/jackc/pgx/v5"
)

type PedidoRequest struct {
	NomeCliente     string  `json:"nome_cliente"`
	TelefoneCliente string  `json:"telefone_cliente"`
	IDEndereco      int     `json:"id_endereco"`
	FormaPagamento  string  `json:"forma_pagamento"`
	Observacoes     *string `json:"observacoes"`
	ItemsPedido     []ItemsPedido `json:"items_pedido"`
	// Opcionais: para "Cartão na entrega" — crédito ou débito + bandeira (taxa aplicada no valor_total).
	TipoCartao string `json:"tipo_cartao"` // "credito" | "debito"
	Bandeira   string `json:"bandeira"`   // "visa" | "mastercard" | "amex" | "elo"
}

// cardFeePercentBr retorna a taxa % válida para o par tipo+bandeira, ou ok=false.
func cardFeePercentBr(tipo, bandeira string) (pct float64, ok bool) {
	t := strings.TrimSpace(strings.ToLower(tipo))
	b := strings.TrimSpace(strings.ToLower(bandeira))
	switch t {
	case "credito":
		switch b {
		case "visa", "mastercard":
			return 3.15, true
		case "amex", "elo":
			return 4.91, true
		}
	case "debito":
		switch b {
		case "visa", "mastercard":
			return 1.37, true
		case "elo":
			return 2.58, true
		}
	}
	return 0, false
}

func isFormaCartaoNaEntrega(forma string) bool {
	l := strings.ToLower(strings.TrimSpace(forma))
	return strings.Contains(l, "cartão na entrega") || strings.Contains(l, "cartao na entrega")
}

func labelBandeira(b string) string {
	switch strings.TrimSpace(strings.ToLower(b)) {
	case "visa":
		return "Visa"
	case "mastercard":
		return "Mastercard"
	case "amex":
		return "Amex"
	case "elo":
		return "Elo"
	default:
		return b
	}
}

func formatPctBR(p float64) string {
	s := fmt.Sprintf("%.2f", p)
	return strings.Replace(s, ".", ",", 1)
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
	MetodoPagamento *string `json:"metodo_pagamento"`
	Observacoes *string `json:"observacoes"`
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
	MetodoPagamento *string `json:"metodo_pagamento"`
	Observacoes *string `json:"observacoes"`
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

	isAdmin, err := middleware.IsAdmin(r.Context(), userID)
	if err != nil {
		http.Error(w, "erro ao validar permissão", http.StatusInternalServerError)
		return
	}

	var pedido PedidoResponseByID
	query := `SELECT p.id_pedido, p.nome, p.telefone, p.data_horario, e.apelido, e.rua, e.numero, e.complemento, e.bairro, e.cidade, e.uf, e.cep, p.status_pedido, p.valor_total, p.metodo_pagamento, p.observacoes, p.created_at, p.updated_at
		FROM pedidos p
		LEFT JOIN enderecos e ON e.id_endereco = p.cd_endereco
		WHERE p.id_pedido = $1 AND p.cd_usuario = $2`
	args := []any{id, userID}
	if isAdmin {
		query = `SELECT p.id_pedido, p.nome, p.telefone, p.data_horario, e.apelido, e.rua, e.numero, e.complemento, e.bairro, e.cidade, e.uf, e.cep, p.status_pedido, p.valor_total, p.metodo_pagamento, p.observacoes, p.created_at, p.updated_at
		FROM pedidos p
		LEFT JOIN enderecos e ON e.id_endereco = p.cd_endereco
		WHERE p.id_pedido = $1`
		args = []any{id}
	}
	err = database.Pool.QueryRow(r.Context(), query, args...).Scan(
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
		&pedido.MetodoPagamento,
		&pedido.Observacoes,
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
	isAdmin, err := middleware.IsAdmin(r.Context(), userID)
	if err != nil {
		http.Error(w, "erro ao validar permissão", http.StatusInternalServerError)
		return
	}

	var rows pgx.Rows
	if isAdmin {
		rows, err = database.Pool.Query(
			r.Context(),
			`SELECT p.id_pedido, p.nome, p.telefone, p.data_horario, e.apelido, e.rua, e.numero, e.complemento, e.bairro, e.cidade, e.uf, e.cep, p.status_pedido, p.valor_total, p.metodo_pagamento, p.observacoes, p.created_at, p.updated_at
			FROM pedidos p
			LEFT JOIN enderecos e ON e.id_endereco = p.cd_endereco
			ORDER BY p.id_pedido DESC`,
		)
	} else {
		rows, err = database.Pool.Query(
			r.Context(),
			`SELECT p.id_pedido, p.nome, p.telefone, p.data_horario, e.apelido, e.rua, e.numero, e.complemento, e.bairro, e.cidade, e.uf, e.cep, p.status_pedido, p.valor_total, p.metodo_pagamento, p.observacoes, p.created_at, p.updated_at
			FROM pedidos p
			LEFT JOIN enderecos e ON e.id_endereco = p.cd_endereco
			WHERE p.cd_usuario = $1
			ORDER BY p.id_pedido DESC`,
			userID,
		)
	}
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
			&pedido.MetodoPagamento,
			&pedido.Observacoes,
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

	metodoPagamento := strings.TrimSpace(pedido.FormaPagamento)
	tipoC := strings.TrimSpace(pedido.TipoCartao)
	band := strings.TrimSpace(pedido.Bandeira)

	if isFormaCartaoNaEntrega(metodoPagamento) {
		if tipoC == "" || band == "" {
			http.Error(w, "Para cartão na entrega informe tipo_cartao (credito ou debito) e bandeira (visa, mastercard, amex ou elo).", http.StatusBadRequest)
			return
		}
		pct, ok := cardFeePercentBr(tipoC, band)
		if !ok {
			http.Error(w, "Combinação inválida de tipo_cartao e bandeira para cartão na entrega.", http.StatusBadRequest)
			return
		}
		valorTotal = valorTotal + valorTotal*(pct/100.0)
		tipoLabel := "crédito"
		if strings.ToLower(tipoC) == "debito" {
			tipoLabel = "débito"
		}
		metodoPagamento = "Cartão " + tipoLabel + " na entrega — " + labelBandeira(band) + " (taxa " + formatPctBR(pct) + "%)"
	} else if tipoC != "" || band != "" {
		http.Error(w, "tipo_cartao e bandeira só devem ser enviados para pagamento com cartão na entrega.", http.StatusBadRequest)
		return
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
		metodoPagamento,
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
	realtime.Publish("orders.updated")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"message":     "Pedido criado com sucesso.",
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func DeletarPedidos(w http.ResponseWriter, r *http.Request, _ int64) {
	userID, err := middleware.GetUserIDFromToken(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	isAdmin, err := middleware.IsAdmin(r.Context(), userID)
	if err != nil {
		http.Error(w, "erro ao validar permissão", http.StatusInternalServerError)
		return
	}
	if !isAdmin {
		http.Error(w, "acesso negado: apenas admin pode limpar pedidos", http.StatusForbidden)
		return
	}

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
	realtime.Publish("orders.updated")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"message":           "Pedidos deletados com sucesso.",
		"pedidos_removidos": result.RowsAffected(),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func AtualizarStatusPedido(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
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

	isAdmin, err := middleware.IsAdmin(r.Context(), userID)
	if err != nil {
		http.Error(w, "erro ao validar permissão", http.StatusInternalServerError)
		return
	}

	statusPermitido := map[string]bool{
		"novo":       true,
		"em_preparo": true,
		"pronto":     true,
		"concluido":  true,
	}
	if !statusPermitido[atualizarStatusPedidoRequest.StatusPedido] {
		http.Error(w, "status_pedido inválido", http.StatusBadRequest)
		return
	}

	selectCurrentStatusQuery := `SELECT status_pedido FROM pedidos WHERE id_pedido = $1 AND cd_usuario = $2`
	selectArgs := []any{id, userID}
	if isAdmin {
		selectCurrentStatusQuery = `SELECT status_pedido FROM pedidos WHERE id_pedido = $1`
		selectArgs = []any{id}
	}
	var statusAtual string
	err = database.Pool.QueryRow(r.Context(), selectCurrentStatusQuery, selectArgs...).Scan(&statusAtual)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Pedido não encontrado", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ordemStatus := map[string]int{
		"novo":       0,
		"em_preparo": 1,
		"pronto":     2,
		"concluido":  3,
	}
	idxAtual, okAtual := ordemStatus[statusAtual]
	idxNovo, okNovo := ordemStatus[atualizarStatusPedidoRequest.StatusPedido]
	if !okAtual || !okNovo {
		http.Error(w, "status_pedido inválido", http.StatusBadRequest)
		return
	}
	if idxNovo < idxAtual {
		http.Error(w, "não é permitido voltar status do pedido", http.StatusBadRequest)
		return
	}
	if statusAtual == "pronto" && atualizarStatusPedidoRequest.StatusPedido != "pronto" && atualizarStatusPedidoRequest.StatusPedido != "concluido" {
		http.Error(w, "pedido pronto só pode ser alterado para concluído", http.StatusBadRequest)
		return
	}
	if statusAtual == "concluido" && atualizarStatusPedidoRequest.StatusPedido != "concluido" {
		http.Error(w, "pedido concluído não pode voltar de status", http.StatusBadRequest)
		return
	}

	query := `UPDATE pedidos SET status_pedido = $1, updated_at = NOW() WHERE id_pedido = $2 AND cd_usuario = $3`
	args := []any{atualizarStatusPedidoRequest.StatusPedido, id, userID}
	if isAdmin {
		query = `UPDATE pedidos SET status_pedido = $1, updated_at = NOW() WHERE id_pedido = $2`
		args = []any{atualizarStatusPedidoRequest.StatusPedido, id}
	}
	result, err := database.Pool.Exec(r.Context(), query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if result.RowsAffected() == 0 {
		http.Error(w, "Pedido não encontrado", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	realtime.Publish("orders.updated")
	resp := map[string]any{
		"message": "Status do pedido atualizado com sucesso.",
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}