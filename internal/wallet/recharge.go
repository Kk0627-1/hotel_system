package wallet

import (
	"database/sql"
	"errors"
	"strings"
)

var (
	ErrInvalidInput        = errors.New("invalid input")
	ErrUserNotFound        = errors.New("user not found")
	ErrWalletNotFound      = errors.New("wallet not found")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrRechargeFailed      = errors.New("recharge failed")
)

type RechargeRequest struct {
	UserID string  `json:"user_id"`
	Amount float64 `json:"amount"`
}

type RechargeResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func Recharge(db *sql.DB, req RechargeRequest) (RechargeResponse, error) {
	// 檢查request輸入
	userID := strings.TrimSpace(req.UserID)
	amount := req.Amount

	if userID == "" || amount <= 0 {
		return RechargeResponse{Success: false, Message: "Invalid input"}, ErrInvalidInput
	}

	// 查詢使用者users資訊
	var userExists bool
	err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&userExists)
	if err != nil {
		return RechargeResponse{Success: false, Message: "Failed to check user"}, err
	}
	if !userExists {
		return RechargeResponse{Success: false, Message: "User not found"}, ErrUserNotFound
	}

	// 查詢使用者錢包wallets資訊
	var walletExists bool
	err = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM wallets WHERE user_id = $1)`, userID).Scan(&walletExists)
	if err != nil {
		return RechargeResponse{Success: false, Message: "Failed to check wallet"}, err
	}
	if !walletExists {
		return RechargeResponse{Success: false, Message: "Wallet not found"}, ErrWalletNotFound
	}

	// 更新使用者錢包wallets資訊
	_, err = db.Exec(`UPDATE wallets SET balance = balance + $1 WHERE user_id = $2`, amount, userID)
	if err != nil {
		return RechargeResponse{Success: false, Message: "Failed to update wallet"}, err
	}

	return RechargeResponse{Success: true, Message: "Recharge successful"}, nil
}
