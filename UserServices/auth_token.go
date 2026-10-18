package UserServices

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"
)

var ErrInvalidToken = errors.New("invalid or expired token")

type tokenPayload struct {
	UserID string `json:"uid"`
	Exp    int64  `json:"exp"`
}

func jwtSecret() []byte {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return []byte(s)
	}
	return []byte("hotelsystem-dev-secret-change-in-production")
}

// IssueAccessToken 簽發登入用 Bearer token（預設 24 小時有效）。
// 格式為 body + "." + sig：body 為可解碼的 uid/exp；sig 為 HMAC-SHA256 防篡改簽名（驗證見 ParseAccessToken）。
func IssueAccessToken(userID string) (string, error) {
	if strings.TrimSpace(userID) == "" {
		return "", ErrInvalidToken
	}
	// 載荷：誰登入、何時過期（Unix 秒）
	payload, err := json.Marshal(tokenPayload{
		UserID: userID,
		Exp:    time.Now().Add(24 * time.Hour).Unix(),
	})
	if err != nil {
		return "", err
	}
	// URL 安全 Base64，作為 token 前半段（非加密，僅方便放在 Header）
	body := base64.RawURLEncoding.EncodeToString(payload)
	// 對 body 字串本身簽名（與 ParseAccessToken 餵入的 parts[0] 一致）
	mac := hmac.New(sha256.New, jwtSecret())
	_, _ = mac.Write([]byte(body))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return body + "." + sig, nil
}

// ParseAccessToken 驗證 token 並回傳 user_id
func ParseAccessToken(token string) (string, error) {
	token = strings.TrimSpace(token)
	parts := strings.Split(token, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", ErrInvalidToken
	}
	mac := hmac.New(sha256.New, jwtSecret())
	_, _ = mac.Write([]byte(parts[0]))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expectedSig), []byte(parts[1])) {
		return "", ErrInvalidToken
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", ErrInvalidToken
	}
	var p tokenPayload
	if err := json.Unmarshal(raw, &p); err != nil || p.UserID == "" {
		return "", ErrInvalidToken
	}
	if time.Now().Unix() > p.Exp {
		return "", ErrInvalidToken
	}
	return p.UserID, nil
}
