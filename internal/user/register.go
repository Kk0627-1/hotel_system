package user

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/lib/pq"
)

var (
	ErrInvalidInput   = errors.New("invalid username or password")
	ErrUsernameExists = errors.New("username already exists")
)

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterResult 註冊成功後含新使用者的 id（供建立 player_profiles 等）。
type RegisterResult struct {
	UserID string
}

func RegisterUser(db *sql.DB, req RegisterRequest) (RegisterResult, error) {
	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	if username == "" || password == "" {
		return RegisterResult{}, ErrInvalidInput
	}
	if len(username) > 45 || len(password) > 10 {
		return RegisterResult{}, ErrInvalidInput
	}

	// 開始交易 錢包寫入失敗時，使用者資料也會一起回滾
	tx, err := db.Begin()
	if err != nil {
		return RegisterResult{}, err
	}
	defer tx.Rollback()

	var userID string
	err = tx.QueryRow(`INSERT INTO users (username, password) VALUES ($1, $2) RETURNING id`, username, password).Scan(&userID)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return RegisterResult{}, ErrUsernameExists
		}
		return RegisterResult{}, err
	}

	if _, err = tx.Exec(`INSERT INTO wallets (user_id) VALUES ($1)`, userID); err != nil {
		return RegisterResult{}, err
	}
	if err = tx.Commit(); err != nil {
		return RegisterResult{}, err
	}
	return RegisterResult{UserID: userID}, nil
}
