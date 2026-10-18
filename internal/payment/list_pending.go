package payment

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidListParams = errors.New("invalid list parameters") // 列表查詢參數不合法（page / page_size）

// PendingRecord 待付款訂單（列表單筆）
type PendingRecord struct {
	RecordID      string `json:"record_id"`
	UserID        string `json:"user_id"`
	HotelID       string `json:"hotel_id"`
	RoomID        string `json:"room_id"`
	TotalPrice    string `json:"total_price"`
	PaymentType   string `json:"payment_type"`
	PaymentStatus string `json:"payment_status"`
	BookingTime   string `json:"booking_time"`
}

// ListPendingRequest 待付款列表請求（POST JSON：分頁與可選 user_id）
type ListPendingRequest struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	UserID   string `json:"user_id"` // 可選，依使用者篩選
}

// ListPendingResult 待付款分頁結果
type ListPendingResult struct {
	Records    []PendingRecord
	Total      int
	Page       int
	PageSize   int
	TotalPages int
}

const maxListPageSize = 100

// ListPending 分頁列出待付款且有效的訂單（payment_status = pending 且 status = TRUE）
func ListPending(db *sql.DB, req ListPendingRequest) (ListPendingResult, error) {
	if req.Page < 1 {
		return ListPendingResult{}, fmt.Errorf("%w: page is required and must be at least 1", ErrInvalidListParams)
	}
	if req.PageSize < 1 {
		return ListPendingResult{}, fmt.Errorf("%w: page_size is required and must be at least 1", ErrInvalidListParams)
	}

	page := req.Page
	pageSize := req.PageSize
	if pageSize > maxListPageSize {
		pageSize = maxListPageSize
	}

	// 固定條件：待付款 + 訂單有效；user_id 有值才加篩選
	conds := []string{"payment_status = 'pending'", "status = TRUE"}
	var args []interface{}
	argN := 1

	if userID := strings.TrimSpace(req.UserID); userID != "" {
		conds = append(conds, fmt.Sprintf("user_id = $%d", argN))
		args = append(args, userID)
		argN++
	}

	whereClause := "WHERE " + strings.Join(conds, " AND ")
	log.Printf("查詢待付款列表：page=%d page_size=%d user_id=%s", page, pageSize, strings.TrimSpace(req.UserID))

	// 先算總筆數，再查分頁資料
	countQuery := "SELECT COUNT(*) FROM records " + whereClause
	var total int
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		log.Printf("查詢待付款筆數失敗：err=%v", err)
		return ListPendingResult{}, err
	}

	offset := (page - 1) * pageSize
	listArgs := append(append([]interface{}{}, args...), pageSize, offset)
	limitArg := argN
	offsetArg := argN + 1
	selectQuery := fmt.Sprintf(
		`SELECT id, user_id::text, hotel_id, room_id, total_price, payment_type, payment_status, booking_time FROM records %s ORDER BY booking_time DESC LIMIT $%d OFFSET $%d`,
		whereClause, limitArg, offsetArg,
	)

	rows, err := db.Query(selectQuery, listArgs...)
	if err != nil {
		log.Printf("查詢待付款列表失敗：err=%v", err)
		return ListPendingResult{}, err
	}
	defer rows.Close()

	records := []PendingRecord{}
	for rows.Next() {
		var rec PendingRecord
		var id, hotelID, roomID int
		var totalPrice float64
		var bookingTime time.Time
		if err := rows.Scan(&id, &rec.UserID, &hotelID, &roomID, &totalPrice, &rec.PaymentType, &rec.PaymentStatus, &bookingTime); err != nil {
			return ListPendingResult{}, err
		}
		rec.RecordID = strconv.Itoa(id)
		rec.HotelID = strconv.Itoa(hotelID)
		rec.RoomID = strconv.Itoa(roomID)
		rec.TotalPrice = strconv.FormatFloat(totalPrice, 'f', 2, 64)
		rec.BookingTime = bookingTime.Format(time.RFC3339)
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return ListPendingResult{}, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	log.Printf("查詢待付款列表成功：total=%d page=%d", total, page)

	return ListPendingResult{
		Records:    records,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}
