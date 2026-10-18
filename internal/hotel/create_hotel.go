package hotel

import (
	"database/sql"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/lib/pq"
)

var (
	ErrInvalidInput    = errors.New("invalid hotel name")
	ErrHotelNameExists = errors.New("hotel name already exists")
	ErrHotelNotFound   = errors.New("hotel not found")
)

type CreateHotelRequest struct {
	HotelName string `json:"hotel_name"`
}

type CreateHotelResult struct {
	Message string `json:"message"`
	HotelID string `json:"hotel_id"`
}

// CreateHotel 創建酒店
func CreateHotel(db *sql.DB, req CreateHotelRequest) (CreateHotelResult, error) {
	hotelName := strings.TrimSpace(req.HotelName)
	log.Printf("創建酒店開始：hotel_name=%q", hotelName)

	if hotelName == "" {
		log.Printf("創建酒店欄位驗證失敗：缺少 hotel_name")
		return CreateHotelResult{Message: "Hotel name is required"}, ErrInvalidInput
	}
	if len(hotelName) > 45 {
		log.Printf("創建酒店欄位驗證失敗：hotel_name 超過 45 字元 hotel_name=%q len=%d", hotelName, len(hotelName))
		return CreateHotelResult{Message: "Hotel name must be less than 45 characters"}, ErrInvalidInput
	}

	var hotelID int
	err := db.QueryRow(`INSERT INTO hotels (name) VALUES ($1) RETURNING id`, hotelName).Scan(&hotelID)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			log.Printf("創建酒店被拒絕：名稱已存在 hotel_name=%q", hotelName)
			return CreateHotelResult{Message: "Hotel name already exists"}, ErrHotelNameExists
		}
		log.Printf("創建酒店失敗：hotel_name=%q err=%v", hotelName, err)
		return CreateHotelResult{Message: "Failed to create hotel"}, err
	}

	log.Printf("創建酒店成功：hotel_id=%d hotel_name=%q", hotelID, hotelName)
	return CreateHotelResult{Message: "Hotel created successfully", HotelID: strconv.Itoa(hotelID)}, nil
}
