package PaymentServices

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

// PurchasedBillRequest 帳單報表查詢（時間範圍必填，其餘篩選可選）
type ListRecordRequest struct {
	Page          int    `json:"page"`
	PageSize      int    `json:"page_size"`
	UserID        string `json:"user_id"`        // 可選，依使用者篩選
	HotelID       string `json:"hotel_id"`       // 可選，依飯店篩選
	RoomID        string `json:"room_id"`        // 可選，依房間篩選
	PaymentStatus string `json:"payment_status"` // 可選：pending / paid / failed / refunded
	Status        *bool  `json:"status"`         // 可選，訂單啟用 true / 停用 false；未傳表示不限
	TimeFilter    int    `json:"time_filter"`    // 1 = booking_time，2 = checkout_time，3 = create_time
	StartTime     string `json:"start_time"`     // 必填，格式 YYYY-MM-DD HH:MM:SS
	EndTime       string `json:"end_time"`       // 必填，格式 YYYY-MM-DD HH:MM:SS
}

// RecordItem 帳單報表單筆
type RecordItem struct {
	RecordID      string `json:"record_id"`
	UserID        string `json:"user_id"`
	HotelID       string `json:"hotel_id"`
	RoomID        string `json:"room_id"`
	TotalPrice    string `json:"total_price"`
	PaymentType   string `json:"payment_type"`
	PaymentStatus string `json:"payment_status"`
	Status        bool   `json:"status"`
	BookingTime   string `json:"booking_time"`
	CheckoutTime  string `json:"checkout_time"`
	CreateTime    string `json:"create_time"`
}

// PurchasedBillResult 帳單報表分頁結果
type ListRecordResult struct {
	List       []RecordItem
	Total      int
	Page       int
	PageSize   int
	TotalPages int
}

// 允許的時間欄位。只從這張表取欄位名，避免把請求數字以外的內容拼進 SQL
var listRecordTimeColumns = map[int]string{
	1: "booking_time",
	2: "checkout_time",
	3: "create_time",
}

// 允許的付款狀態，與 records.payment_status 一致
var listRecordPaymentStatuses = map[string]struct{}{
	"pending":  {},
	"paid":     {},
	"failed":   {},
	"refunded": {},
}

// ListPurchasedBills 依時間範圍與可選條件分頁查詢帳單
// 1. 檢查分頁、time_filter、起迄時間
// 2. 有傳才加上 status / user / hotel / room / payment_status
// 3. 先算總筆數，再查該頁資料
func ListRecord(db *sql.DB, req ListRecordRequest) (ListRecordResult, error) {
	// 分頁：page、page_size 至少為 1，且不得超過 maxListPageSize
	if req.Page < 1 {
		return ListRecordResult{}, fmt.Errorf("%w: page is required and must be at least 1", ErrInvalidListParams)
	}
	if req.PageSize < 1 {
		return ListRecordResult{}, fmt.Errorf("%w: page_size is required and must be at least 1", ErrInvalidListParams)
	}
	if req.PageSize > maxListPageSize {
		return ListRecordResult{}, fmt.Errorf("%w: page_size is too large", ErrInvalidListParams)
	}

	page := req.Page
	pageSize := req.PageSize

	// time_filter：1 booking_time、2 checkout_time、3 create_time
	timeColumn, ok := listRecordTimeColumns[req.TimeFilter]
	if !ok {
		return ListRecordResult{}, fmt.Errorf("%w: time_filter must be 1, 2, or 3", ErrInvalidListParams)
	}

	// 起迄時間必填，格式 YYYY-MM-DD HH:MM:SS，且 start 不能晚於 end
	startRaw := strings.TrimSpace(req.StartTime)
	endRaw := strings.TrimSpace(req.EndTime)
	if startRaw == "" || endRaw == "" {
		return ListRecordResult{}, fmt.Errorf("%w: start_time and end_time are required", ErrInvalidListParams)
	}
	startTime, err := time.Parse(time.DateTime, startRaw)
	if err != nil {
		return ListRecordResult{}, fmt.Errorf("%w: invalid start_time", ErrInvalidListParams)
	}
	endTime, err := time.Parse(time.DateTime, endRaw)
	if err != nil {
		return ListRecordResult{}, fmt.Errorf("%w: invalid end_time", ErrInvalidListParams)
	}
	if startTime.After(endTime) {
		return ListRecordResult{}, fmt.Errorf("%w: start_time must be before end_time", ErrInvalidListParams)
	}

	// 時間範圍是固定條件；下面的篩選有值才加入
	conds := []string{fmt.Sprintf("%s BETWEEN $1 AND $2", timeColumn)}
	args := []interface{}{startTime, endTime}
	argN := 3

	if req.Status != nil {
		conds = append(conds, fmt.Sprintf("status = $%d", argN))
		args = append(args, *req.Status)
		argN++
	}
	if userID := strings.TrimSpace(req.UserID); userID != "" {
		conds = append(conds, fmt.Sprintf("user_id = $%d", argN))
		args = append(args, userID)
		argN++
	}
	if hotelIDStr := strings.TrimSpace(req.HotelID); hotelIDStr != "" {
		hotelID, convErr := strconv.Atoi(hotelIDStr)
		if convErr != nil || hotelID <= 0 {
			return ListRecordResult{}, fmt.Errorf("%w: invalid hotel_id", ErrInvalidListParams)
		}
		conds = append(conds, fmt.Sprintf("hotel_id = $%d", argN))
		args = append(args, hotelID)
		argN++
	}
	if roomIDStr := strings.TrimSpace(req.RoomID); roomIDStr != "" {
		roomID, convErr := strconv.Atoi(roomIDStr)
		if convErr != nil || roomID <= 0 {
			return ListRecordResult{}, fmt.Errorf("%w: invalid room_id", ErrInvalidListParams)
		}
		conds = append(conds, fmt.Sprintf("room_id = $%d", argN))
		args = append(args, roomID)
		argN++
	}
	if paymentStatus := strings.TrimSpace(req.PaymentStatus); paymentStatus != "" {
		if _, allowed := listRecordPaymentStatuses[paymentStatus]; !allowed {
			return ListRecordResult{}, fmt.Errorf("%w: invalid payment_status", ErrInvalidListParams)
		}
		conds = append(conds, fmt.Sprintf("payment_status = $%d", argN))
		args = append(args, paymentStatus)
		argN++
	}

	whereClause := "WHERE " + strings.Join(conds, " AND ")
	log.Printf("查詢帳單報表：page=%d page_size=%d time_filter=%s start=%s end=%s", page, pageSize, timeColumn, startRaw, endRaw)

	// 先算總筆數，再依所選時間由新到舊取該頁
	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM records "+whereClause, args...).Scan(&total); err != nil {
		log.Printf("查詢帳單筆數失敗：err=%v", err)
		return ListRecordResult{}, err
	}

	// offset: 要跳過的比數
	offset := (page - 1) * pageSize
	listArgs := append(append([]interface{}{}, args...), pageSize, offset)
	selectQuery := fmt.Sprintf(
		`SELECT id, user_id::text, hotel_id, room_id, total_price, payment_type, payment_status, status, booking_time, checkout_time, create_time FROM records %s ORDER BY %s DESC LIMIT $%d OFFSET $%d`,
		whereClause, timeColumn, argN, argN+1,
	)

	rows, err := db.Query(selectQuery, listArgs...)
	if err != nil {
		log.Printf("查詢帳單報表失敗：err=%v", err)
		return ListRecordResult{}, err
	}
	defer rows.Close()

	// id、金額轉成字串；三個時間轉成 RFC3339
	list := []RecordItem{}
	for rows.Next() {
		var bill RecordItem
		var id, hotelID, roomID int
		var totalPrice float64
		var bookingTime, checkoutTime, createTime time.Time
		if err := rows.Scan(
			&id, &bill.UserID, &hotelID, &roomID, &totalPrice,
			&bill.PaymentType, &bill.PaymentStatus, &bill.Status,
			&bookingTime, &checkoutTime, &createTime,
		); err != nil {
			return ListRecordResult{}, err
		}
		bill.RecordID = strconv.Itoa(id)
		bill.HotelID = strconv.Itoa(hotelID)
		bill.RoomID = strconv.Itoa(roomID)
		bill.TotalPrice = strconv.FormatFloat(totalPrice, 'f', 2, 64)
		bill.BookingTime = bookingTime.Format(time.RFC3339)
		bill.CheckoutTime = checkoutTime.Format(time.RFC3339)
		bill.CreateTime = createTime.Format(time.RFC3339)
		list = append(list, bill)
	}
	if err := rows.Err(); err != nil {
		return ListRecordResult{}, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	log.Printf("查詢帳單報表成功：total=%d page=%d", total, page)

	return ListRecordResult{
		List:       list,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}
