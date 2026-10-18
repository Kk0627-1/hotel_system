package hotel

import (
	"database/sql"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/lib/pq"
)

type UpdateHotelRequest struct {
	HotelID   string `json:"hotel_id"`
	HotelName string `json:"hotel_name"`
	Status    bool   `json:"status"`
}

type UpdateHotelResult struct {
	Message string `json:"message"`
	HotelID string `json:"hotel_id"`
}

// UpdateHotel 更新酒店
func UpdateHotel(db *sql.DB, req UpdateHotelRequest) (UpdateHotelResult, error) {
	hotelID := strings.TrimSpace(req.HotelID)
	hotelName := strings.TrimSpace(req.HotelName)
	status := req.Status
	log.Printf("更新酒店開始：hotel_id=%q hotel_name=%q status=%t", hotelID, hotelName, status)

	if hotelID == "" {
		log.Printf("更新酒店欄位驗證失敗：缺少 hotel_id")
		return UpdateHotelResult{Message: "Hotel ID is required"}, ErrInvalidInput
	}
	if _, err := strconv.Atoi(hotelID); err != nil {
		log.Printf("更新酒店欄位驗證失敗：hotel_id 無效 hotel_id=%q", hotelID)
		return UpdateHotelResult{Message: "Invalid hotel ID"}, ErrInvalidInput
	}
	if hotelName == "" {
		log.Printf("更新酒店欄位驗證失敗：缺少 hotel_name")
		return UpdateHotelResult{Message: "Hotel name is required"}, ErrInvalidInput
	}
	if len(hotelName) > 45 {
		log.Printf("更新酒店欄位驗證失敗：hotel_name 超過 45 字元 hotel_name=%q len=%d", hotelName, len(hotelName))
		return UpdateHotelResult{Message: "Hotel name must be less than 45 characters"}, ErrInvalidInput
	}

	var updatedID int
	err := db.QueryRow(`UPDATE hotels SET name = $1, status = $2 WHERE id = $3 RETURNING id`, hotelName, status, hotelID).Scan(&updatedID)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && string(pgErr.Code) == "23505" {
			log.Printf("更新酒店被拒絕：名稱已存在 hotel_id=%s hotel_name=%q", hotelID, hotelName)
			return UpdateHotelResult{Message: "Hotel name already exists"}, ErrHotelNameExists
		}
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("更新酒店被拒絕：酒店不存在 hotel_id=%s", hotelID)
			return UpdateHotelResult{Message: "Failed to update hotel"}, err
		}
		log.Printf("更新酒店失敗：hotel_id=%s hotel_name=%q err=%v", hotelID, hotelName, err)
		return UpdateHotelResult{Message: "Failed to update hotel"}, err
	}

	log.Printf("更新酒店成功：hotel_id=%d hotel_name=%q status=%t", updatedID, hotelName, status)
	return UpdateHotelResult{Message: "Hotel updated successfully", HotelID: strconv.Itoa(updatedID)}, nil
}
