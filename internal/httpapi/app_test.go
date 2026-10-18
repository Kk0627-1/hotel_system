package httpapi_test

import (
	"database/sql"
	"net/http"

	"hotelsystem/internal/hotel"
	"hotelsystem/internal/httpapi"
	"hotelsystem/internal/payment"
	"hotelsystem/internal/reservation"
	"hotelsystem/internal/room"
	"hotelsystem/internal/user"
	"hotelsystem/internal/wallet"
	"hotelsystem/internal/web"
)

var db *sql.DB

type Config struct{}

func (*Config) routes() http.Handler {
	return httpapi.NewRouter(httpapi.Handlers{
		User:        user.NewHandler(user.NewService(user.NewRepository(db))),
		Hotel:       hotel.NewHandler(hotel.NewService(hotel.NewRepository(db))),
		Room:        room.NewHandler(room.NewService(room.NewRepository(db))),
		Reservation: reservation.NewHandler(reservation.NewService(reservation.NewRepository(db))),
		Payment:     payment.NewHandler(payment.NewService(payment.NewRepository(db))),
		Wallet:      wallet.NewHandler(wallet.NewService(wallet.NewRepository(db))),
	})
}

type SimpleResponse = web.SimpleResponse
type UserSessionResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
	Username string `json:"username,omitempty"`
	UserID   string `json:"user_id,omitempty"`
	Token    string `json:"token,omitempty"`
}
type RoomWriteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	RoomID  string `json:"room_id,omitempty"`
	HotelID string `json:"hotel_id,omitempty"`
}
type ReservationResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message,omitempty"`
	RecordID   string `json:"record_id,omitempty"`
	UserID     string `json:"user_id,omitempty"`
	HotelID    string `json:"hotel_id,omitempty"`
	RoomID     string `json:"room_id,omitempty"`
	BookingNum string `json:"booking_num,omitempty"`
	Nights     string `json:"nights,omitempty"`
	TotalPrice string `json:"total_price,omitempty"`
}
type PaymentConfirmResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message,omitempty"`
	RecordID      string `json:"record_id,omitempty"`
	UserID        string `json:"user_id,omitempty"`
	HotelID       string `json:"hotel_id,omitempty"`
	RoomID        string `json:"room_id,omitempty"`
	TotalPrice    string `json:"total_price,omitempty"`
	PaymentType   string `json:"payment_type,omitempty"`
	PaymentStatus string `json:"payment_status,omitempty"`
}
type PaymentPendingListResponse struct {
	Success   bool                    `json:"success"`
	Message   string                  `json:"message,omitempty"`
	Records   []payment.PendingRecord `json:"records"`
	Total     int                     `json:"total,omitempty"`
	Page      int                     `json:"page,omitempty"`
	PageSize  int                     `json:"page_size,omitempty"`
	TotalPage int                     `json:"total_page,omitempty"`
}
