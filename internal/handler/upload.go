package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"cardapio-henry-api/internal/config"
)

func Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "erro ao ler arquivo", http.StatusBadRequest)
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "arquivo 'file' é obrigatório", http.StatusBadRequest)
		return
	}
	defer file.Close()

	contentType := fileHeader.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		http.Error(w, "somente imagens são permitidas", http.StatusBadRequest)
		return
	}

	r2Cfg := config.LoadR2()
	r2Client, err := config.NewR2Storage(r2Cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	url, objectKey, err := r2Client.UploadImage(r.Context(), file, fileHeader.Filename, contentType)
	if err != nil {
		http.Error(w, "erro ao enviar para o R2", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message":    "upload realizado com sucesso",
		"url":        url,
		"object_key": objectKey,
	})
}
