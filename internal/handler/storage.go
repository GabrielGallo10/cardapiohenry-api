package handler

import (
	"io"
	"net/http"
	"strings"

	"henry-bebidas-api/internal/config"
)

// GetStorage faz streaming público de objetos do R2 (GET /storage/{path...}).
// Evita depender do subdomínio *.r2.dev, que pode responder 500 mesmo com bucket público.
func GetStorage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	key := r.PathValue("path")
	if key == "" || strings.Contains(key, "..") {
		http.NotFound(w, r)
		return
	}

	cfg := config.LoadR2()
	prefix := strings.Trim(cfg.ObjectKeyPrefix, "/")
	if prefix != "" {
		if key != prefix && !strings.HasPrefix(key, prefix+"/") {
			http.NotFound(w, r)
			return
		}
	}

	client, err := config.NewR2Storage(cfg)
	if err != nil {
		http.Error(w, "armazenamento indisponível", http.StatusInternalServerError)
		return
	}

	out, err := client.GetObject(r.Context(), key)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer out.Body.Close()

	if out.ContentType != nil && *out.ContentType != "" {
		w.Header().Set("Content-Type", *out.ContentType)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	if out.CacheControl != nil && *out.CacheControl != "" {
		w.Header().Set("Cache-Control", *out.CacheControl)
	} else {
		w.Header().Set("Cache-Control", "public, max-age=86400")
	}

	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, out.Body)
}
