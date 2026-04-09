// middleware CORS: libera requisições do frontend permitido.
package middleware

import (
	"net/http"
	"os"
	"strings"
)

// DefaultAllowedOrigins: origens permitidas quando CORS_ORIGINS não está definido.
var DefaultAllowedOrigins = []string{
	"https://henrybebidas.vercel.app",
}

// CORS adiciona os headers para o navegador aceitar requisições do front. Pode configurar origens em CORS_ORIGINS.
func CORS(next http.Handler) http.Handler {
	origins := DefaultAllowedOrigins
	if v := os.Getenv("CORS_ORIGINS"); v != "" {
		origins = splitTrim(v, ",")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && isAllowedOrigin(origin, origins) {
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if reqHeaders := r.Header.Get("Access-Control-Request-Headers"); reqHeaders != "" {
			w.Header().Set("Access-Control-Allow-Headers", reqHeaders)
		} else {
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
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

func isAllowedOrigin(origin string, allowed []string) bool {
	origin = normalizeOrigin(origin)
	for _, rule := range allowed {
		rule = normalizeOrigin(rule)
		if rule == origin {
			return true
		}

		// Permite wildcard simples, ex: https://*.vercel.app
		if strings.Contains(rule, "*") {
			parts := strings.Split(rule, "*")
			if len(parts) != 2 {
				continue
			}
			prefix := parts[0]
			suffix := parts[1]
			if strings.HasPrefix(origin, prefix) && strings.HasSuffix(origin, suffix) {
				return true
			}
		}
	}
	return false
}

func normalizeOrigin(v string) string {
	return strings.TrimRight(strings.TrimSpace(v), "/")
}
