package handler

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"henry-bebidas-api/internal/auth"
	"henry-bebidas-api/internal/config"
	"henry-bebidas-api/internal/database"
	"henry-bebidas-api/internal/notify"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	pwdResetCodeTTL      = 10 * time.Minute
	pwdResetMaxAttempts  = 6
	pwdResetBcryptCost   = 10
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type pwdResetRequestBody struct {
	Email string `json:"email"`
}

type pwdResetVerifyBody struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type pwdResetConfirmBody struct {
	ResetToken           string `json:"reset_token"`
	Password             string `json:"password"`
	PasswordConfirmation string `json:"password_confirmation"`
}

func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func isValidEmailBasic(s string) bool {
	s = normalizeEmail(s)
	if len(s) < 5 || len(s) > 320 {
		return false
	}
	at := strings.LastIndex(s, "@")
	if at <= 0 || at == len(s)-1 {
		return false
	}
	local := s[:at]
	domain := s[at+1:]
	if len(local) == 0 || len(domain) < 3 || !strings.Contains(domain, ".") {
		return false
	}
	return true
}

func randomSixDigitCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(n.Int64()+100000, 10), nil
}

// PasswordResetRequest POST /password-reset/request — envia e-mail (Brevo) se o endereço existir.
func PasswordResetRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	var body pwdResetRequestBody
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Dados inválidos"})
		return
	}
	emailNorm := normalizeEmail(body.Email)
	if !isValidEmailBasic(emailNorm) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "E-mail inválido"})
		return
	}

	ctx := r.Context()
	var idUsuario int
	err := database.Pool.QueryRow(ctx,
		`SELECT id_usuario FROM usuarios WHERE LOWER(TRIM(COALESCE(email,''))) = $1 LIMIT 1`,
		emailNorm,
	).Scan(&idUsuario)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "Não há conta cadastrada com este e-mail. Confira o endereço ou crie uma conta.",
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Erro ao verificar o e-mail."})
		return
	}
	_ = idUsuario

	code, err := randomSixDigitCode()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Erro ao gerar código"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(code), pwdResetBcryptCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Erro ao processar código"})
		return
	}
	expires := time.Now().UTC().Add(pwdResetCodeTTL)

	_, _ = database.Pool.Exec(ctx,
		`UPDATE password_reset_challenges SET consumed_at = NOW()
		 WHERE email_norm = $1 AND consumed_at IS NULL`,
		emailNorm,
	)
	_, err = database.Pool.Exec(ctx,
		`INSERT INTO password_reset_challenges (email_norm, code_hash, expires_at)
		 VALUES ($1, $2, $3)`,
		emailNorm, string(hash), expires,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Falha ao registar pedido. Execute as migrações SQL (incl. migrations/002_email_password_reset.sql) se ainda não o fez.",
		})
		return
	}

	mailCfg := config.LoadTransactionalEmail()
	cfg := notify.TransactionalEmail{
		Provider:    mailCfg.Provider,
		APIKey:      mailCfg.BrevoAPIKey,
		SenderName:  mailCfg.SenderName,
		SenderEmail: mailCfg.SenderEmail,
	}
	if err := notify.SendPasswordResetEmail(ctx, cfg, emailNorm, code); err != nil {
		_, _ = database.Pool.Exec(ctx,
			`UPDATE password_reset_challenges SET consumed_at = NOW()
			 WHERE email_norm = $1 AND consumed_at IS NULL`,
			emailNorm,
		)
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"error": "Não foi possível enviar o e-mail. Verifique EMAIL_PROVIDER=brevo, BREVO_API_KEY e BREVO_SENDER_EMAIL (remetente confirmado na Brevo), ou use EMAIL_PROVIDER=log em desenvolvimento.",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Enviámos um código de verificação para o seu e-mail.",
	})
}

// PasswordResetVerify POST /password-reset/verify — valida código e devolve reset_token (JWT).
func PasswordResetVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	var body pwdResetVerifyBody
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Dados inválidos"})
		return
	}
	emailNorm := normalizeEmail(body.Email)
	code := strings.TrimSpace(body.Code)
	if !isValidEmailBasic(emailNorm) || len(code) < 4 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "E-mail ou código inválido"})
		return
	}

	ctx := r.Context()
	var id int64
	var codeHash string
	var attemptCount int
	err := database.Pool.QueryRow(ctx, `
		SELECT id, code_hash, attempt_count
		FROM password_reset_challenges
		WHERE email_norm = $1 AND consumed_at IS NULL AND expires_at > NOW()
		ORDER BY id DESC
		LIMIT 1
	`, emailNorm).Scan(&id, &codeHash, &attemptCount)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Código inválido ou expirado."})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Erro ao validar"})
		return
	}
	if attemptCount >= pwdResetMaxAttempts {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "Muitas tentativas. Peça um novo código."})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(codeHash), []byte(code)); err != nil {
		_, _ = database.Pool.Exec(ctx,
			`UPDATE password_reset_challenges SET attempt_count = attempt_count + 1 WHERE id = $1`,
			id,
		)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Código incorreto."})
		return
	}

	var userID int64
	err = database.Pool.QueryRow(ctx,
		`SELECT id_usuario FROM usuarios WHERE LOWER(TRIM(COALESCE(email,''))) = $1 LIMIT 1`,
		emailNorm,
	).Scan(&userID)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Utilizador não encontrado."})
		return
	}

	_, err = database.Pool.Exec(ctx,
		`UPDATE password_reset_challenges SET consumed_at = NOW() WHERE id = $1`,
		id,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Erro ao concluir validação"})
		return
	}

	jwtCfg := config.LoadJWT()
	expMin := config.PasswordResetTokenMinutes()
	token, err := auth.GeneratePasswordResetToken(jwtCfg.Secret, expMin, userID, emailNorm)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Erro ao emitir token"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"reset_token": token,
		"expires_in":  expMin * 60,
	})
}

// PasswordResetConfirm POST /password-reset/confirm — redefine senha com JWT de reset.
func PasswordResetConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	var body pwdResetConfirmBody
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Dados inválidos"})
		return
	}
	if body.Password != body.PasswordConfirmation {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "As senhas não coincidem"})
		return
	}
	if len(body.Password) < 6 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Senha demasiado curta (mín. 6 caracteres)"})
		return
	}
	token := strings.TrimSpace(body.ResetToken)
	if token == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Token em falta"})
		return
	}

	jwtCfg := config.LoadJWT()
	userID, emailNorm, err := auth.ValidatePasswordResetToken(jwtCfg.Secret, token)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Link de recuperação inválido ou expirado. Volte a pedir o código."})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Erro ao processar senha"})
		return
	}

	ctx := r.Context()
	tag, err := database.Pool.Exec(ctx,
		`UPDATE usuarios SET senha = $1 WHERE id_usuario = $2 AND LOWER(TRIM(COALESCE(email,''))) = $3`,
		string(hashed), userID, emailNorm,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Erro ao atualizar senha"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Não foi possível atualizar a conta."})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Senha atualizada. Já pode entrar."})
}
