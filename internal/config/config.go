package config

import (
	"net/url"
	"os"
	"strconv"
)

// DB contém os parâmetros de conexão com o PostgreSQL.
type DB struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string
}

// DSN retorna a connection string no formato aceito pelo pgx/postgres.
// Usa net/url para codificar usuário e senha (evita que @, #, : etc. quebrem a URL).
func (c DB) DSN() string {
	u := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.User, c.Password),
		Host:     c.Host + ":" + c.Port,
		Path:     "/" + c.Database,
		RawQuery: "sslmode=" + c.SSLMode,
	}
	return u.String()
}

// LoadDB lê a configuração do banco a partir de variáveis de ambiente.
// Use o arquivo .env na raiz do projeto (não versionado) para definir os valores.
func LoadDB() DB {
	return DB{
		Host:     getEnv("DB_HOST", "10.0.0.1"),
		Port:     getEnv("DB_PORT", "5440"),
		User:     getEnv("DB_USER", "dev_gabriel"),
		Password: getEnv("DB_PASSWORD", "ADPG87784554@#"),
		Database: getEnv("DB_NAME", "adpg_barber_shop"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}
}

// JWT contém a configuração para geração de tokens.
type JWT struct {
	Secret          string
	ExpirationHours int
}

type R2 struct {
	AccountID      string
	AccessKeyID    string
	SecretAccessKey string
	Bucket         string
	Region         string
	Endpoint       string
	PublicBaseURL  string
	ObjectKeyPrefix string
}

// LoadJWT lê a configuração JWT a partir de variáveis de ambiente.
func LoadJWT() JWT {
	hours := 24
	if h, err := strconv.Atoi(getEnv("JWT_EXPIRATION_HOURS", "24")); err == nil && h > 0 {
		hours = h
	}
	return JWT{
		Secret:          getEnv("JWT_SECRET", "HDSAHJ@$412aAFhJ341AFKALC3441@facj"),
		ExpirationHours: hours,
	}
}

func LoadR2() R2 {
	return R2{
		AccountID:       getEnv("CF_R2_ACCOUNT_ID", ""),
		AccessKeyID:     getEnv("CF_R2_ACCESS_KEY_ID", ""),
		SecretAccessKey: getEnv("CF_R2_SECRET_ACCESS_KEY", ""),
		Bucket:          getEnv("CF_R2_BUCKET", ""),
		Region:          getEnv("CF_R2_REGION", "auto"),
		Endpoint:        getEnv("CF_R2_ENDPOINT", ""),
		PublicBaseURL:   getEnv("CF_R2_PUBLIC_BASE_URL", ""),
		ObjectKeyPrefix: getEnv("CF_R2_OBJECT_KEY_PREFIX", "uploads"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
