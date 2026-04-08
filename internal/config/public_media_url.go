package config

import (
	"fmt"
	"net/http"
	"strings"
)

// RequestPublicBase devolve o URL base público (scheme + host) do pedido, respeitando proxies.
func RequestPublicBase(r *http.Request) string {
	host := r.Host
	if host == "" {
		return ""
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	if xh := r.Header.Get("X-Forwarded-Host"); xh != "" {
		host = xh
	}
	return scheme + "://" + host
}

// ResolvePublicMediaURL é a URL que o cliente deve usar no <img> / DB.
// Prioridade: PUBLIC_OBJECT_BASE_URL → CF_R2_PUBLIC_BASE_URL → proxy /storage na própria API → r2.dev.
func ResolvePublicMediaURL(r *http.Request, cfg R2, objectKey string) string {
	key := strings.TrimPrefix(strings.TrimSpace(objectKey), "/")
	if base := strings.TrimRight(cfg.PublicObjectBaseURL, "/"); base != "" {
		return base + "/storage/" + key
	}
	if base := strings.TrimRight(cfg.PublicBaseURL, "/"); base != "" {
		return base + "/" + key
	}
	if base := RequestPublicBase(r); base != "" {
		return base + "/storage/" + key
	}
	return fmt.Sprintf("https://%s.r2.dev/%s", cfg.Bucket, key)
}
