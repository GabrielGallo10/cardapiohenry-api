// middleware CORS: libera requisições do frontend (Vercel e localhost).
package middleware

import (
	"net/http"
	"os"
)

// DefaultAllowedOrigins: origens permitidas quando CORS_ORIGINS não está definido.
var DefaultAllowedOrigins = []string{
	"https://studio-leoleite.vercel.app",
	"http://localhost:3000",
	"http://localhost:3001",
	"http://127.0.0.1:3000",
	"http://127.0.0.1:3001",
}

// CORS adiciona os headers para o navegador aceitar requisições do front. Pode configurar origens em CORS_ORIGINS.
func CORS(next http.Handler) http.Handler {
	origins := DefaultAllowedOrigins
	if v := os.Getenv("CORS_ORIGINS"); v != "" {
		origins = splitTrim(v, ",")
	}
	allowed := make(map[string]bool)
	for _, o := range origins {
		allowed[o] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func splitTrim(s, sep string) []string {
	var out []string
	// split simples por vírgula
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			t := trim(s[start:i])
			if t != "" {
				out = append(out, t)
			}
			start = i + 1
		}
	}
	return out
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
