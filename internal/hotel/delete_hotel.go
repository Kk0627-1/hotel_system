package hotel

import (
	"database/sql"
	"errors"
	"log"
	"strconv"
	"strings"
)

type DeleteHotelRequest struct {
	HotelID string `json:"hotel_id"`
}

type DeleteHotelResult struct {
	Message string `json:"message"`
	HotelID string `json:"hotel_id"`
}

// DeleteHotel 刪除飯店
func DeleteHotel(db *sql.DB, req DeleteHotelRequest) (DeleteHotelResult, error) {
	hotelID := strings.TrimSpace(req.HotelID)
	log.Printf("刪除酒店開始：hotel_id=%q", hotelID)

	if hotelID == "" {
		log.Printf("刪除酒店欄位驗證失敗：缺少 hotel_id")
		return DeleteHotelResult{Message: "Hotel ID is required"}, ErrInvalidInput
	}
	if _, err := strconv.Atoi(hotelID); err != nil {
		log.Printf("刪除酒店欄位驗證失敗：hotel_id 無效 hotel_id=%q", hotelID)
		return DeleteHotelResult{Message: "Invalid hotel ID"}, ErrInvalidInput
	}

	var deletedID int
	err := db.QueryRow(`DELETE FROM hotels WHERE id = $1 RETURNING id`, hotelID).Scan(&deletedID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("刪除酒店被拒絕：酒店不存在 hotel_id=%s", hotelID)
			return DeleteHotelResult{Message: "Hotel not found"}, ErrHotelNotFound
		}
		log.Printf("刪除酒店失敗：hotel_id=%s err=%v", hotelID, err)
		return DeleteHotelResult{Message: "Failed to delete hotel"}, err
	}

	log.Printf("刪除酒店成功：hotel_id=%d", deletedID)
	return DeleteHotelResult{Message: "Hotel deleted successfully", HotelID: strconv.Itoa(deletedID)}, nil
}
