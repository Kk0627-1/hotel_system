package payment

import (
	"database/sql"
	"errors"
	"log"
	"strconv"
	"strings"
)

var (
	ErrOrderDisabled = errors.New("order disabled") // 訂單已停用
)

// CancelPaymentRequest 取消未付款訂單請求
type CancelPaymentRequest struct {
	RecordID string `json:"record_id"`
	UserID   string `json:"user_id"`
}

// CancelPaymentResult 取消未付款訂單結果
type CancelPaymentResult struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// CancelPayment 取消未付款訂單
// 1. 訂單須為 pending 且 status = TRUE
// 2. 將 status 改為 FALSE，payment_status 維持 pending
// 3. 房間 quantity + 1
func CancelPayment(db *sql.DB, req CancelPaymentRequest) (CancelPaymentResult, error) {
	recordIDStr := strings.TrimSpace(req.RecordID)
	userID := strings.TrimSpace(req.UserID)
	log.Printf("取消訂單開始：record_id=%s user_id=%s", recordIDStr, userID)
	if userID == "" {
		log.Printf("取消訂單欄位驗證失敗：缺少 user_id")
		return CancelPaymentResult{Success: false, Message: "User ID is required"}, ErrInvalidInput
	}
	if recordIDStr == "" {
		log.Printf("取消訂單欄位驗證失敗：缺少 record_id")
		return CancelPaymentResult{Success: false, Message: "record_id is required"}, ErrInvalidInput
	}

	recordID, err := strconv.Atoi(recordIDStr)
	if err != nil || recordID <= 0 {
		log.Printf("取消訂單欄位驗證失敗：record_id 無效 record_id=%q", recordIDStr)
		return CancelPaymentResult{Success: false, Message: "invalid record_id"}, ErrInvalidInput
	}

	tx, err := db.Begin()
	if err != nil {
		log.Printf("取消訂單交易開始失敗：record_id=%d err=%v", recordID, err)
		return CancelPaymentResult{Success: false, Message: "failed to start transaction"}, err
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr == nil {
			log.Printf("取消訂單交易已回滾：record_id=%d", recordID)
		}
	}()

	// 查詢訂單records資訊
	var ownerID string
	var roomID int
	var bookingNum int
	var paymentStatus string
	var status bool
	err = tx.QueryRow(
		`SELECT user_id::text, room_id, booking_num, payment_status, status FROM records WHERE id = $1 FOR UPDATE`,
		recordID,
	).Scan(&ownerID, &roomID, &bookingNum, &paymentStatus, &status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("取消訂單被拒絕：訂單不存在 record_id=%d", recordID)
			return CancelPaymentResult{Success: false, Message: "record not found"}, ErrRecordNotFound
		}
		log.Printf("查詢取消訂單失敗：record_id=%d err=%v", recordID, err)
		return CancelPaymentResult{Success: false, Message: "failed to check record"}, err
	}
	if ownerID != userID {
		log.Printf("取消訂單被拒絕：訂單不屬於該使用者 record_id=%d user_id=%s", recordID, userID)
		return CancelPaymentResult{Success: false, Message: "record not found"}, ErrRecordNotFound
	}

	if !status {
		log.Printf("取消訂單被拒絕：訂單已停用 record_id=%d", recordID)
		return CancelPaymentResult{Success: false, Message: "order is disabled"}, ErrOrderDisabled
	}
	if paymentStatus != "pending" {
		log.Printf("取消訂單被拒絕：付款狀態不是 pending record_id=%d payment_status=%s", recordID, paymentStatus)
		return CancelPaymentResult{Success: false, Message: "payment is not pending"}, ErrPaymentNotPending
	}

	// 更新訂單狀態
	result, err := tx.Exec(
		`UPDATE records SET status = FALSE WHERE id = $1 AND user_id = $2 AND payment_status = 'pending' AND status = TRUE`,
		recordID, userID,
	)
	if err != nil {
		log.Printf("更新訂單狀態失敗：record_id=%d err=%v", recordID, err)
		return CancelPaymentResult{Success: false, Message: "failed to update record"}, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		log.Printf("讀取訂單更新筆數失敗：record_id=%d err=%v", recordID, err)
		return CancelPaymentResult{Success: false, Message: "failed to update record"}, err
	}
	if rows == 0 {
		log.Printf("取消訂單被拒絕：訂單狀態已變更 record_id=%d", recordID)
		return CancelPaymentResult{Success: false, Message: "payment is not pending"}, ErrPaymentNotPending
	}

	// rooms
	// 歸還房間庫存
	if bookingNum <= 0 {
		bookingNum = 1
	}
	// 更新房間庫存
	if _, err = tx.Exec(`UPDATE rooms SET quantity = quantity + $1 WHERE id = $2`, bookingNum, roomID); err != nil {
		log.Printf("歸還房間庫存失敗：record_id=%d room_id=%d err=%v", recordID, roomID, err)
		return CancelPaymentResult{Success: false, Message: "failed to restore room quantity"}, err
	}
	log.Printf("房間庫存已歸還：record_id=%d room_id=%d", recordID, roomID)

	if err = tx.Commit(); err != nil {
		log.Printf("取消訂單交易提交失敗：record_id=%d err=%v", recordID, err)
		return CancelPaymentResult{Success: false, Message: "failed to commit transaction"}, err
	}
	log.Printf("取消訂單成功：record_id=%d room_id=%d", recordID, roomID)
	return CancelPaymentResult{Success: true, Message: "order cancelled"}, nil
}
