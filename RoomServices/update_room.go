package RoomServices

import (
	"database/sql"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/lib/pq"
)

type UpdateRoomRequest struct {
	RoomID   int    `json:"room_id"`
	RoomType string `json:"room_type"`
	Price    int    `json:"price"`
	Quantity int    `json:"quantity"`
	Status   bool   `json:"status"`
}

type UpdateRoomResult struct {
	Message string `json:"message"`
	RoomID  string `json:"room_id"`
	HotelID string `json:"hotel_id"`
}

// UpdateRoom 更新房間
func UpdateRoom(db *sql.DB, req UpdateRoomRequest) (UpdateRoomResult, error) {
	roomID := req.RoomID
	roomType := strings.TrimSpace(req.RoomType)
	price := req.Price
	quantity := req.Quantity
	status := req.Status
	log.Printf("更新房間開始：room_id=%d room_type=%q price=%d quantity=%d status=%t", roomID, roomType, price, quantity, status)

	if roomID <= 0 {
		log.Printf("更新房間欄位驗證失敗：缺少 room_id room_id=%d", roomID)
		return UpdateRoomResult{Message: "Room ID is required"}, ErrInvalidInput
	}
	if roomType == "" {
		log.Printf("更新房間欄位驗證失敗：缺少 room_type")
		return UpdateRoomResult{Message: "Room type is required"}, ErrInvalidInput
	}
	if len(roomType) > 45 {
		log.Printf("更新房間欄位驗證失敗：room_type 超過 45 字元 room_type=%q len=%d", roomType, len(roomType))
		return UpdateRoomResult{Message: "Room type must be less than 45 characters"}, ErrInvalidInput
	}
	if price <= 0 {
		log.Printf("更新房間欄位驗證失敗：price 必須大於 0 price=%d", price)
		return UpdateRoomResult{Message: "Price must be greater than 0"}, ErrInvalidInput
	}
	if quantity <= 0 {
		quantity = 1
	}

	var updatedID, hotelID int
	err := db.QueryRow(
		`UPDATE rooms SET room_type = $1, price = $2, quantity = $3, status = $4 WHERE id = $5 RETURNING id, hotel_id`,
		roomType, price, quantity, status, roomID,
	).Scan(&updatedID, &hotelID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("更新房間被拒絕：房間不存在 room_id=%d", roomID)
			return UpdateRoomResult{Message: "Room not found"}, ErrRoomNotFound
		}
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			log.Printf("更新房間被拒絕：房型已存在 room_id=%d room_type=%q", roomID, roomType)
			return UpdateRoomResult{Message: "Room type already exists in this hotel"}, ErrRoomTypeExists
		}
		log.Printf("更新房間失敗：room_id=%d room_type=%q err=%v", roomID, roomType, err)
		return UpdateRoomResult{Message: "Failed to update room"}, err
	}

	log.Printf("更新房間成功：room_id=%d hotel_id=%d room_type=%q price=%d quantity=%d status=%t", updatedID, hotelID, roomType, price, quantity, status)
	return UpdateRoomResult{
		Message: "Room updated successfully",
		RoomID:  strconv.Itoa(updatedID),
		HotelID: strconv.Itoa(hotelID),
	}, nil
}
