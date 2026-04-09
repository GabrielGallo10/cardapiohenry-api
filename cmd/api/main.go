package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"


	"henry-bebidas-api/internal/config"
	"henry-bebidas-api/internal/database"
	"henry-bebidas-api/internal/handler"
	"henry-bebidas-api/internal/middleware"
	"github.com/joho/godotenv"
)

func main() {
	ctx := context.Background()

	// Carrega variáveis de ambiente do arquivo .env (se existir).
	// Tenta múltiplos caminhos: ao rodar "go run" de cmd/api, o CWD é cmd/api e o .env está na raiz (../.env).
	for _, p := range []string{".env", "../.env", "../../.env"} {
		if err := godotenv.Load(p); err == nil {
			break
		}
	}

	// Conecta ao PostgreSQL usando variáveis de ambiente (.env)
	cfg := config.LoadDB()
	_, err := database.Connect(ctx, cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer database.Close()

	log.Println("conectado ao PostgreSQL")

	mux := http.NewServeMux()

	// ROTAS PUBLICAS
	mux.HandleFunc("POST /register", handler.Register)
	mux.HandleFunc("POST /login", handler.Login)
	mux.HandleFunc("GET /storage/{path...}", handler.GetStorage)
	mux.HandleFunc("POST /password-reset/request", handler.PasswordResetRequest)
	mux.HandleFunc("POST /password-reset/verify", handler.PasswordResetVerify)
	mux.HandleFunc("POST /password-reset/confirm", handler.PasswordResetConfirm)
	mux.HandleFunc("GET /produtos", handler.Produtos)
	mux.HandleFunc("GET /categorias", handler.Categorias)

	// ROTAS PROTEGIDAS POR TOKEN
	mux.Handle("/categorias", middleware.JWT(http.HandlerFunc(handler.Categorias)))
	mux.Handle("DELETE /categorias/{id}", middleware.JWT(http.HandlerFunc(handler.DeletarCategoria)))
	mux.Handle("POST /produtos", middleware.JWT(http.HandlerFunc(handler.CriarProdutos)))
	mux.Handle("PUT /produtos/{id}", middleware.JWT(http.HandlerFunc(handler.ProdutosByID)))
	mux.Handle("DELETE /produtos/{id}", middleware.JWT(http.HandlerFunc(handler.ProdutosByID)))
	mux.Handle("GET /clientes", middleware.JWT(http.HandlerFunc(handler.Clientes)))
	mux.Handle("/enderecos", middleware.JWT(http.HandlerFunc(handler.Enderecos)))
	mux.Handle("/enderecos/{id}", middleware.JWT(http.HandlerFunc(handler.EnderecosByID)))
	mux.Handle("/pedidos", middleware.JWT(http.HandlerFunc(handler.Pedidos)))
	mux.Handle("GET /pedidos/{id}", middleware.JWT(http.HandlerFunc(handler.PedidosByID)))
	mux.Handle("PUT /pedidos/{id}", middleware.JWT(http.HandlerFunc(handler.AtualizarStatusPedido)))
	mux.Handle("POST /upload", middleware.JWT(http.HandlerFunc(handler.Upload)))
	mux.Handle("/financas", middleware.JWT(http.HandlerFunc(handler.Financas)))
	mux.Handle("/financas/{id}", middleware.JWT(http.HandlerFunc(handler.FinancasByID)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	srv := &http.Server{
		Addr:    addr,
		Handler: middleware.CORS(mux),
	}

	go func() {
		log.Printf("Servidor rodando na porta %s", port)
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	// Aguarda Ctrl+C ou sinal de encerramento para desligar com segurança
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("encerrando...")

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("erro ao desligar servidor HTTP: %v", err)
	}
}
