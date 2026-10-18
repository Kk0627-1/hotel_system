package user

import (
	"database/sql"
	"errors"
	"strings"

	"hotelsystem/internal/auth"
)

var ErrUsernameNotFound = errors.New("username not found")

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResult struct {
	UserID   string
	Username string
	Message  string
	Token    string
}

func LoginUser(db *sql.DB, req LoginRequest) (LoginResult, error) {
	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	if username == "" || password == "" {
		return LoginResult{
			Message:  "Username and password are required",
			UserID:   "",
			Username: "",
			Token:    "",
		}, ErrInvalidInput
	}
	if len(username) > 45 || len(password) > 10 {
		return LoginResult{
			Message:  "Username and password must be less than 45 and 10 characters respectively",
			UserID:   "",
			Username: req.Username,
			Token:    "",
		}, ErrInvalidInput
	}
	var userID, storedPassword string
	err := db.QueryRow(`SELECT id::text, password FROM users WHERE username = $1`, username).Scan(&userID, &storedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// 使用者不存在
			return LoginResult{
				Message:  "Username not found or incorrect password",
				UserID:   "",
				Username: req.Username,
				Token:    "",
			}, ErrUsernameNotFound
		}
		// 其他錯誤
		return LoginResult{
			Message:  "Login failed",
			UserID:   "",
			Username: req.Username,
			Token:    "",
		}, err
	}
	// 驗證密碼
	if storedPassword != password {
		return LoginResult{
			Message:  "Username not found or incorrect password",
			UserID:   "",
			Username: req.Username,
			Token:    "",
		}, ErrUsernameNotFound
	}
	// 簽發登入用 Bearer token
	token, err := auth.IssueAccessToken(userID)
	if err != nil {
		return LoginResult{
			Message:  "Login failed",
			UserID:   "",
			Username: username,
			Token:    "",
		}, err
	}
	return LoginResult{
		Message:  "Login successful",
		UserID:   userID,
		Username: username,
		Token:    token,
	}, nil
}
