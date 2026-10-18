package ReservationServices

import (
	"database/sql"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidInput      = errors.New("invalid input")
	ErrUserNotFound      = errors.New("user not found")
	ErrRoomNotFound      = errors.New("room not found")
	ErrRoomNotAvailable  = errors.New("room not available")
	ErrReservationFailed = errors.New("reservation failed")
)

type CreateReservationRequest struct {
	UserID       string `json:"user_id"`
	RoomID       int    `json:"room_id"`
	CheckinTime  string `json:"checkin_time"`  // 格式 YYYY-MM-DD HH:MM:SS
	CheckoutTime string `json:"checkout_time"` // 格式 YYYY-MM-DD HH:MM:SS
	BookingNum   int    `json:"booking_num"`   // 訂房間數，預設 1
}

type CreateReservationResult struct {
	Message    string `json:"message"`
	RecordID   string `json:"record_id"`
	UserID     string `json:"user_id"`
	HotelID    string `json:"hotel_id"`
	RoomID     string `json:"room_id"`
	BookingNum string `json:"booking_num"`
	Nights     string `json:"nights"`
	TotalPrice string `json:"total_price"`
}

// CreateReservation 創建訂房記錄
// 1. 檢查 user 是否存在
// 2. 檢查 room_id 是否存在
// 3. 檢查 room quantity 是否 >= booking_num
// 4. 依晚數 × 間數計算 total_price，扣減庫存
// 5. 建立訂單記錄
func CreateReservation(db *sql.DB, req CreateReservationRequest) (CreateReservationResult, error) {
	userID := strings.TrimSpace(req.UserID)
	roomID := req.RoomID
	checkinTimeStr := strings.TrimSpace(req.CheckinTime)
	checkoutTimeStr := strings.TrimSpace(req.CheckoutTime)
	bookingNum := req.BookingNum
	if bookingNum <= 0 {
		bookingNum = 1
	}
	log.Printf("訂房服務開始：user_id=%s room_id=%d booking_num=%d", userID, roomID, bookingNum)

	if userID == "" {
		log.Printf("訂房欄位驗證失敗：缺少 user_id")
		return CreateReservationResult{Message: "User ID is required"}, ErrInvalidInput
	}
	if roomID <= 0 {
		log.Printf("訂房欄位驗證失敗：room_id 無效 room_id=%d", roomID)
		return CreateReservationResult{Message: "Room ID is required"}, ErrInvalidInput
	}
	if checkinTimeStr == "" {
		log.Printf("訂房欄位驗證失敗：缺少 checkin_time")
		return CreateReservationResult{Message: "Check-in time is required"}, ErrInvalidInput
	}
	if checkoutTimeStr == "" {
		log.Printf("訂房欄位驗證失敗：缺少 checkout_time")
		return CreateReservationResult{Message: "Checkout time is required"}, ErrInvalidInput
	}

	checkinTime, err := time.Parse("2006-01-02 15:04:05", checkinTimeStr)
	if err != nil {
		log.Printf("訂房欄位驗證失敗：checkin_time 格式錯誤 checkin_time=%q", checkinTimeStr)
		return CreateReservationResult{Message: "Check-in time must be in format YYYY-MM-DD HH:MM:SS"}, ErrInvalidInput
	}
	checkoutTime, err := time.Parse("2006-01-02 15:04:05", checkoutTimeStr)
	if err != nil {
		log.Printf("訂房欄位驗證失敗：checkout_time 格式錯誤 checkout_time=%q", checkoutTimeStr)
		return CreateReservationResult{Message: "Checkout time must be in format YYYY-MM-DD HH:MM:SS"}, ErrInvalidInput
	}
	if !checkoutTime.After(checkinTime) {
		log.Printf("訂房欄位驗證失敗：checkin_time 必須早於 checkout_time checkin_time=%s checkout_time=%s", checkinTimeStr, checkoutTimeStr)
		return CreateReservationResult{Message: "Check-in time must be before checkout time"}, ErrInvalidInput
	}
	if checkoutTime.Before(time.Now()) {
		log.Printf("訂房欄位驗證失敗：checkout_time 必須是未來時間 checkout_time=%s", checkoutTimeStr)
		return CreateReservationResult{Message: "Checkout time must be in the future"}, ErrInvalidInput
	}

	// 依入住至退房時間計算晚數（不足 24 小時以 1 晚計，其餘向上取整）
	nights := ReservationNights(checkinTime, checkoutTime)
	if nights <= 0 {
		return CreateReservationResult{Message: "Invalid stay duration"}, ErrInvalidInput
	}

	// 查詢使用者users資訊
	var userExists bool
	err = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&userExists)
	if err != nil {
		log.Printf("查詢訂房使用者失敗：user_id=%s err=%v", userID, err)
		return CreateReservationResult{Message: "Failed to check user"}, err
	}
	if !userExists {
		log.Printf("訂房請求被拒絕：使用者不存在 user_id=%s", userID)
		return CreateReservationResult{Message: "User not found"}, ErrUserNotFound
	}

	tx, err := db.Begin()
	if err != nil {
		log.Printf("訂房交易開始失敗：user_id=%s room_id=%d err=%v", userID, roomID, err)
		return CreateReservationResult{Message: "Failed to start transaction"}, err
	}
	log.Printf("訂房交易開始：user_id=%s room_id=%d", userID, roomID)
	// 失敗 -> 訂房交易回滾
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr == nil {
			log.Printf("訂房交易已回滾：user_id=%s room_id=%d", userID, roomID)
		}
	}()

	// 查詢房間rooms資訊
	var hotelID int
	var price float64
	var quantity int
	err = tx.QueryRow(
		`SELECT hotel_id, price, quantity FROM rooms WHERE id = $1 AND status = TRUE FOR UPDATE`,
		roomID,
	).Scan(&hotelID, &price, &quantity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("訂房請求被拒絕：房間不存在或不可使用 room_id=%d", roomID)
			return CreateReservationResult{Message: "Room not found or not available"}, ErrRoomNotFound
		}
		log.Printf("查詢訂房房間失敗：room_id=%d err=%v", roomID, err)
		return CreateReservationResult{Message: "Failed to check room"}, err
	}
	log.Printf("訂房房間已鎖定：room_id=%d hotel_id=%d quantity=%d price=%.2f nights=%d booking_num=%d", roomID, hotelID, quantity, price, nights, bookingNum)

	// 檢查房間庫存是否足夠
	if quantity < bookingNum {
		log.Printf("訂房請求被拒絕：庫存不足 room_id=%d quantity=%d booking_num=%d", roomID, quantity, bookingNum)
		return CreateReservationResult{Message: "Room is fully booked"}, ErrRoomNotAvailable
	}

	// 依晚數 × 間數計算 total_price，扣減庫存
	totalPrice := ReservationTotalPrice(price, nights, bookingNum)

	// 扣減房間庫存
	_, err = tx.Exec(`UPDATE rooms SET quantity = quantity - $1 WHERE id = $2`, bookingNum, roomID)
	if err != nil {
		log.Printf("扣減房間庫存失敗：room_id=%d err=%v", roomID, err)
		return CreateReservationResult{Message: "Failed to update room quantity"}, err
	}
	log.Printf("房間庫存扣減成功：room_id=%d old_quantity=%d booking_num=%d new_quantity=%d", roomID, quantity, bookingNum, quantity-bookingNum)

	// 建立訂房紀錄
	var recordID int
	err = tx.QueryRow(
		`INSERT INTO records (user_id, hotel_id, room_id, checkin_time, checkout_time, total_price, booking_num) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		userID, hotelID, roomID, checkinTime, checkoutTime, totalPrice, bookingNum,
	).Scan(&recordID)
	if err != nil {
		log.Printf("建立訂房紀錄失敗：user_id=%s room_id=%d err=%v", userID, roomID, err)
		return CreateReservationResult{Message: "Failed to create reservation record"}, err
	}
	log.Printf("訂房紀錄建立成功：record_id=%d user_id=%s room_id=%d total_price=%.2f", recordID, userID, roomID, totalPrice)

	if err = tx.Commit(); err != nil {
		log.Printf("訂房交易提交失敗：record_id=%d err=%v", recordID, err)
		return CreateReservationResult{Message: "Failed to commit transaction"}, err
	}
	log.Printf("訂房交易提交成功：record_id=%d", recordID)

	return CreateReservationResult{
		Message:    "Reservation created successfully",
		RecordID:   strconv.Itoa(recordID),
		UserID:     userID,
		HotelID:    strconv.Itoa(hotelID),
		RoomID:     strconv.Itoa(roomID),
		BookingNum: strconv.Itoa(bookingNum),
		Nights:     strconv.Itoa(nights),
		TotalPrice: strconv.FormatFloat(totalPrice, 'f', 2, 64),
	}, nil
}
