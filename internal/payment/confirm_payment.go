package payment

import (
	"database/sql"
	"errors"
	"log"
	"strconv"
	"strings"
)

var (
	ErrInvalidInput        = errors.New("invalid input")        // 缺少或無效的 record_id / user_id
	ErrRecordNotFound      = errors.New("record not found")     // 訂單不存在或 user_id 不符
	ErrPaymentNotPending   = errors.New("payment not pending")  // 已付款、已取消，或狀態已被變更
	ErrWalletNotFound      = errors.New("wallet not found")     // 使用者錢包不存在
	ErrInsufficientBalance = errors.New("insufficient balance") // 錢包餘額不足
)

// ConfirmPaymentRequest 確認支付請求（pending → paid）
type ConfirmPaymentRequest struct {
	RecordID string `json:"record_id"`
	UserID   string `json:"user_id"`
}

// ConfirmPaymentResult 確認支付結果
type ConfirmPaymentResult struct {
	Message       string
	RecordID      string
	UserID        string
	HotelID       string
	RoomID        string
	TotalPrice    string
	PaymentType   string
	PaymentStatus string
}

// ConfirmPayment 確認訂單付款
// 1. 檢查 record_id、user_id
// 2. 於交易中鎖定訂單列（FOR UPDATE），確認存在且屬於該使用者
// 3. 訂單須有效（status = TRUE）且付款狀態為 pending
// 4. 鎖定使用者錢包；payment_type = wallet 時檢查餘額並扣款
// 5. 將 payment_status 更新為 paid
func ConfirmPayment(db *sql.DB, req ConfirmPaymentRequest) (ConfirmPaymentResult, error) {
	recordIDStr := strings.TrimSpace(req.RecordID)
	userID := strings.TrimSpace(req.UserID)
	log.Printf("確認支付開始：record_id=%s user_id=%s", recordIDStr, userID)

	if recordIDStr == "" {
		log.Printf("確認支付欄位驗證失敗：缺少 record_id")
		return ConfirmPaymentResult{Message: "Record ID is required"}, ErrInvalidInput
	}
	recordID, err := strconv.Atoi(recordIDStr)
	if err != nil || recordID <= 0 {
		log.Printf("確認支付欄位驗證失敗：record_id 無效 record_id=%q", recordIDStr)
		return ConfirmPaymentResult{Message: "Record ID is required"}, ErrInvalidInput
	}
	if userID == "" {
		log.Printf("確認支付欄位驗證失敗：缺少 user_id")
		return ConfirmPaymentResult{Message: "User ID is required"}, ErrInvalidInput
	}

	tx, err := db.Begin()
	if err != nil {
		log.Printf("確認支付交易開始失敗：record_id=%d err=%v", recordID, err)
		return ConfirmPaymentResult{Message: "Failed to start transaction"}, err
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr == nil {
			log.Printf("確認支付交易已回滾：record_id=%d user_id=%s", recordID, userID)
		}
	}()

	// 查詢訂單records資訊
	var ownerID string
	var hotelID, roomID int
	var totalPrice float64
	var paymentType, paymentStatus string
	var status bool
	err = tx.QueryRow(
		`SELECT user_id::text, hotel_id, room_id, total_price, payment_type, payment_status, status FROM records WHERE id = $1 FOR UPDATE`,
		recordID,
	).Scan(&ownerID, &hotelID, &roomID, &totalPrice, &paymentType, &paymentStatus, &status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("確認支付被拒絕：訂單不存在 record_id=%d", recordID)
			return ConfirmPaymentResult{Message: "Reservation not found"}, ErrRecordNotFound
		}
		log.Printf("查詢支付訂單失敗：record_id=%d err=%v", recordID, err)
		return ConfirmPaymentResult{Message: "Failed to check reservation"}, err
	}

	if ownerID != userID {
		log.Printf("確認支付被拒絕：訂單不存在 record_id=%d user_id=%s", recordID, userID)
		return ConfirmPaymentResult{Message: "Reservation not found"}, ErrRecordNotFound
	}
	if !status {
		log.Printf("確認支付被拒絕：訂單已取消 record_id=%d", recordID)
		return ConfirmPaymentResult{Message: "Reservation is cancelled"}, ErrPaymentNotPending
	}
	if paymentStatus != "pending" {
		log.Printf("確認支付被拒絕：付款狀態不是 pending record_id=%d payment_status=%s", recordID, paymentStatus)
		return ConfirmPaymentResult{Message: "Payment is not pending"}, ErrPaymentNotPending
	}

	// 確認支付前檢查錢包；wallet 付款時扣減餘額
	if err := debitWalletIfNeeded(tx, userID, paymentType, totalPrice, recordID); err != nil {
		switch {
		case errors.Is(err, ErrWalletNotFound):
			return ConfirmPaymentResult{Message: "Wallet not found"}, ErrWalletNotFound
		case errors.Is(err, ErrInsufficientBalance):
			return ConfirmPaymentResult{Message: "Insufficient wallet balance"}, ErrInsufficientBalance
		default:
			return ConfirmPaymentResult{Message: "Failed to check wallet"}, err
		}
	}

	// 更新付款狀態
	var updatedID int
	err = tx.QueryRow(
		`UPDATE records SET payment_status = 'paid' WHERE id = $1 AND user_id = $2 AND payment_status = 'pending' AND status = TRUE RETURNING id`,
		recordID, userID,
	).Scan(&updatedID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("確認支付被拒絕：付款狀態已變更 record_id=%d", recordID)
			return ConfirmPaymentResult{Message: "Payment is not pending"}, ErrPaymentNotPending
		}
		log.Printf("更新付款狀態失敗：record_id=%d err=%v", recordID, err)
		return ConfirmPaymentResult{Message: "Failed to confirm payment"}, err
	}

	// 提交交易
	if err = tx.Commit(); err != nil {
		log.Printf("確認支付交易提交失敗：record_id=%d err=%v", recordID, err)
		return ConfirmPaymentResult{Message: "Failed to confirm payment"}, err
	}
	log.Printf("確認支付成功：record_id=%d user_id=%s payment_type=%s", updatedID, userID, paymentType)

	return ConfirmPaymentResult{
		Message:       "Payment confirmed successfully",
		RecordID:      strconv.Itoa(updatedID),
		UserID:        userID,
		HotelID:       strconv.Itoa(hotelID),
		RoomID:        strconv.Itoa(roomID),
		TotalPrice:    strconv.FormatFloat(totalPrice, 'f', 2, 64),
		PaymentType:   paymentType,
		PaymentStatus: "paid",
	}, nil
}

// debitWalletIfNeeded 確認支付前檢查錢包；wallet 付款時扣減餘額
func debitWalletIfNeeded(tx *sql.Tx, userID, paymentType string, amount float64, recordID int) error {
	// 確保使用者錢包存在, 沒有就加一個
	_, err := tx.Exec(
		`INSERT INTO wallets (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`,
		userID,
	)
	if err != nil {
		log.Printf("確保錢包存在失敗：user_id=%s record_id=%d err=%v", userID, recordID, err)
		return err
	}

	// 查詢使用者錢包餘額
	var balance float64
	err = tx.QueryRow(
		`SELECT balance FROM wallets WHERE user_id = $1 FOR UPDATE`,
		userID,
	).Scan(&balance)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("確認支付被拒絕：錢包不存在 user_id=%s record_id=%d", userID, recordID)
			return ErrWalletNotFound
		}
		log.Printf("查詢錢包失敗：user_id=%s record_id=%d err=%v", userID, recordID, err)
		return err
	}

	if !strings.EqualFold(strings.TrimSpace(paymentType), "wallet") {
		log.Printf("確認支付錢包檢查通過（非 wallet 付款）：user_id=%s balance=%.2f payment_type=%s", userID, balance, paymentType)
		return nil
	}

	// 檢查錢包餘額是否足夠
	if balance < amount {
		log.Printf("確認支付被拒絕：錢包餘額不足 user_id=%s balance=%.2f amount=%.2f record_id=%d", userID, balance, amount, recordID)
		return ErrInsufficientBalance
	}

	// 扣減錢包餘額
	_, err = tx.Exec(
		`UPDATE wallets SET balance = balance - $1, update_time = CURRENT_TIMESTAMP WHERE user_id = $2 AND balance >= $1`,
		amount, userID,
	)
	if err != nil {
		log.Printf("扣減錢包失敗：user_id=%s amount=%.2f record_id=%d err=%v", userID, amount, recordID, err)
		return err
	}
	log.Printf("錢包扣款成功：user_id=%s amount=%.2f record_id=%d", userID, amount, recordID)
	return nil
}
