package reservation

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"hotelsystem/internal/web"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

type createResponse struct {
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

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		web.WriteJSON(w, http.StatusBadRequest, web.SimpleResponse{Success: false, Message: "Failed to read request body: " + err.Error()})
		return
	}
	defer r.Body.Close()

	var req CreateReservationRequest
	if err := json.Unmarshal(bodyData, &req); err != nil {
		web.WriteJSON(w, http.StatusBadRequest, web.SimpleResponse{Success: false, Message: "Invalid JSON format: " + err.Error()})
		return
	}
	if !web.RequireMatchingUser(w, r, req.UserID) {
		return
	}

	result, err := h.service.Create(req)
	if err != nil {
		log.Printf("create reservation failed: user_id=%s room_id=%d err=%v", req.UserID, req.RoomID, err)
		switch {
		case errors.Is(err, ErrInvalidInput):
			web.WriteJSON(w, http.StatusBadRequest, web.SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, ErrUserNotFound), errors.Is(err, ErrRoomNotFound):
			web.WriteJSON(w, http.StatusNotFound, web.SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, ErrRoomNotAvailable):
			web.WriteJSON(w, http.StatusConflict, web.SimpleResponse{Success: false, Message: result.Message})
		default:
			web.WriteJSON(w, http.StatusInternalServerError, web.SimpleResponse{Success: false, Message: "Failed to create reservation"})
		}
		return
	}
	web.WriteJSON(w, http.StatusOK, createResponse{
		Success: true, Message: result.Message, RecordID: result.RecordID, UserID: result.UserID,
		HotelID: result.HotelID, RoomID: result.RoomID, BookingNum: result.BookingNum,
		Nights: result.Nights, TotalPrice: result.TotalPrice,
	})
}
