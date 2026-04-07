# Cardapio Henry API

Base inicial de API em Go para um sistema de cardapio de distribuidora de bebidas, com conexao PostgreSQL e endpoint de health check.

## Requisitos

- Go 1.22+
- Docker (opcional, para subir o PostgreSQL local)

## Configuracao

1. Copie o arquivo de exemplo:

```bash
cp .env.example .env
```

2. Suba o banco local:

```bash
docker compose up -d postgres
```

3. Rode a API:

```bash
go run ./cmd/api
```

## Endpoints iniciais

- `GET /health`

## Comandos uteis

- `make run` - roda a API
- `make test` - executa testes
- `make tidy` - organiza dependencias
- `make db-up` - sobe PostgreSQL no Docker
- `make db-down` - derruba containers
