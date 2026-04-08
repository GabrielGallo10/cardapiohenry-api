package database

import (
	"context"
	"fmt"

	"henry-bebidas-api/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool é o pool de conexões PostgreSQL (reutilizável na aplicação).
var Pool *pgxpool.Pool

// Connect abre o pool de conexões com o PostgreSQL usando a config carregada.
// Deve ser chamado na inicialização da API (ex.: main.go).
func Connect(ctx context.Context, cfg config.DB) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	Pool = pool
	return pool, nil
}

// Close encerra o pool de conexões. Chamar ao desligar a aplicação (defer).
func Close() {
	if Pool != nil {
		Pool.Close()
	}
}
