package RoomServices

import (
	"database/sql"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/lib/pq"
)

var (
	ErrInvalidInput      = errors.New("invalid input")
	ErrRoomTypeExists    = errors.New("room type already exists in this hotel")
	ErrRoomNotFound      = errors.New("room not found")
	ErrHotelNotFound     = errors.New("hotel not found")
)

type CreateRoomRequest struct {
	HotelID  int    `json:"hotel_id"`
	RoomType string `json:"room_type"`
	Price    int    `json:"price"`
	Quantity int    `json:"quantity"`
}

type CreateRoomResult struct {
	Message  string `json:"message"`
	RoomID   string `json:"room_id"`
	HotelID  string `json:"hotel_id"`
}

// CreateRoom 創建房間
func CreateRoom(db *sql.DB, req CreateRoomRequest) (CreateRoomResult, error) {
	hotelID := req.HotelID
	roomType := strings.TrimSpace(req.RoomType)
	price := req.Price
	quantity := req.Quantity
	log.Printf("創建房間開始：hotel_id=%d room_type=%q price=%d quantity=%d", hotelID, roomType, price, quantity)

	if hotelID <= 0 {
		log.Printf("創建房間欄位驗證失敗：缺少 hotel_id hotel_id=%d", hotelID)
		return CreateRoomResult{Message: "Hotel ID is required"}, ErrInvalidInput
	}
	if roomType == "" {
		log.Printf("創建房間欄位驗證失敗：缺少 room_type")
		return CreateRoomResult{Message: "Room type is required"}, ErrInvalidInput
	}
	if len(roomType) > 45 {
		log.Printf("創建房間欄位驗證失敗：room_type 超過 45 字元 room_type=%q len=%d", roomType, len(roomType))
		return CreateRoomResult{Message: "Room type must be less than 45 characters"}, ErrInvalidInput
	}
	if price <= 0 {
		log.Printf("創建房間欄位驗證失敗：price 必須大於 0 price=%d", price)
		return CreateRoomResult{Message: "Price must be greater than 0"}, ErrInvalidInput
	}

	if quantity <= 0 {
		quantity = 1
	}

	var hotelExists bool
	err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM hotels WHERE id = $1)`, hotelID).Scan(&hotelExists)
	if err != nil {
		log.Printf("創建房間失敗：查詢酒店失敗 hotel_id=%d err=%v", hotelID, err)
		return CreateRoomResult{Message: "Failed to create room"}, err
	}
	if !hotelExists {
		log.Printf("創建房間被拒絕：酒店不存在 hotel_id=%d", hotelID)
		return CreateRoomResult{Message: "Hotel not found"}, ErrHotelNotFound
	}

	var roomID int
	err = db.QueryRow(
		`INSERT INTO rooms (hotel_id, room_type, price, quantity) VALUES ($1, $2, $3, $4) RETURNING id`,
		hotelID, roomType, price, quantity,
	).Scan(&roomID)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			log.Printf("創建房間被拒絕：房型已存在 hotel_id=%d room_type=%q", hotelID, roomType)
			return CreateRoomResult{Message: "Room type already exists in this hotel"}, ErrRoomTypeExists
		}
		log.Printf("創建房間失敗：hotel_id=%d room_type=%q err=%v", hotelID, roomType, err)
		return CreateRoomResult{Message: "Failed to create room"}, err
	}

	log.Printf("創建房間成功：room_id=%d hotel_id=%d room_type=%q price=%d quantity=%d", roomID, hotelID, roomType, price, quantity)
	return CreateRoomResult{
		Message: "Room created successfully",
		RoomID:  strconv.Itoa(roomID),
		HotelID: strconv.Itoa(hotelID),
	}, nil
}
