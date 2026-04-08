package config

import (
	"net/url"
	"strings"
)

// MediaObjectKeyFromStoredURL extrai a chave do objeto R2 a partir da URL guardada em produtos.url_imagem.
// Suporta proxy da API (/storage/…), URL pública R2 e CF_R2_PUBLIC_BASE_URL.
func MediaObjectKeyFromStoredURL(raw string, cfg R2) (key string, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	path := strings.Trim(u.Path, "/")
	prefix := strings.Trim(cfg.ObjectKeyPrefix, "/")

	var candidate string
	if strings.HasPrefix(path, "storage/") {
		candidate = strings.TrimPrefix(path, "storage/")
		candidate = strings.TrimPrefix(candidate, "/")
	} else {
		candidate = path
	}
	if candidate == "" {
		return "", false
	}

	if prefix == "" {
		return candidate, !strings.Contains(candidate, "..")
	}
	if candidate == prefix || strings.HasPrefix(candidate, prefix+"/") {
		return candidate, true
	}
	return "", false
}
