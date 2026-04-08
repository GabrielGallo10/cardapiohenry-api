package handler

import (
	"context"
	"log"
	"strings"

	"henry-bebidas-api/internal/config"
	"henry-bebidas-api/internal/database"
)

// deleteProductImageFromR2IfLastReference apaga o ficheiro no R2 quando nenhum produto usa mais esta URL.
func deleteProductImageFromR2IfLastReference(ctx context.Context, imageURL string) {
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" {
		return
	}
	var n int
	err := database.Pool.QueryRow(ctx,
		`SELECT COUNT(*)::int FROM produtos WHERE COALESCE(TRIM(url_imagem), '') = $1`,
		imageURL,
	).Scan(&n)
	if err != nil {
		log.Printf("[r2] count refs: %v", err)
		return
	}
	if n > 0 {
		return
	}
	r2cfg := config.LoadR2()
	key, ok := config.MediaObjectKeyFromStoredURL(imageURL, r2cfg)
	if !ok || key == "" {
		return
	}
	client, err := config.NewR2Storage(r2cfg)
	if err != nil {
		log.Printf("[r2] skip delete: %v", err)
		return
	}
	if !client.OwnedObjectKey(key) {
		return
	}
	if err := client.DeleteObject(ctx, key); err != nil {
		log.Printf("[r2] delete %s: %v", key, err)
	}
}
