package room

import (
	"database/sql"
	"errors"
	"log"
	"strconv"
	"strings"
)

type DeleteRoomRequest struct {
	RoomID string `json:"room_id"`
}

type DeleteRoomResult struct {
	Message string `json:"message"`
	RoomID  string `json:"room_id"`
	HotelID string `json:"hotel_id"`
}

// DeleteRoom 刪除房間
func DeleteRoom(db *sql.DB, req DeleteRoomRequest) (DeleteRoomResult, error) {
	roomID := strings.TrimSpace(req.RoomID)
	log.Printf("刪除房間開始：room_id=%q", roomID)

	if roomID == "" {
		log.Printf("刪除房間欄位驗證失敗：缺少 room_id")
		return DeleteRoomResult{Message: "Room ID is required"}, ErrInvalidInput
	}
	if _, err := strconv.Atoi(roomID); err != nil {
		log.Printf("刪除房間欄位驗證失敗：room_id 無效 room_id=%q", roomID)
		return DeleteRoomResult{Message: "Invalid room ID"}, ErrInvalidInput
	}

	var deletedID, hotelID int
	err := db.QueryRow(`DELETE FROM rooms WHERE id = $1 RETURNING id, hotel_id`, roomID).Scan(&deletedID, &hotelID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("刪除房間被拒絕：房間不存在 room_id=%s", roomID)
			return DeleteRoomResult{Message: "Room not found"}, ErrRoomNotFound
		}
		log.Printf("刪除房間失敗：room_id=%s err=%v", roomID, err)
		return DeleteRoomResult{Message: "Failed to delete room"}, err
	}

	log.Printf("刪除房間成功：room_id=%d hotel_id=%d", deletedID, hotelID)
	return DeleteRoomResult{
		Message: "Room deleted successfully",
		RoomID:  strconv.Itoa(deletedID),
		HotelID: strconv.Itoa(hotelID),
	}, nil
}
