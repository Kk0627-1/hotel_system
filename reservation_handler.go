package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"hotelsystem/ReservationServices"
)

// CreateReservationHandler 創建訂房處理器
type CreateReservationHandler struct{}

// 創建訂房
func (h CreateReservationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("收到訂房請求：method=%s path=%s", r.Method, r.URL.Path)
	setCORS(w)
	if r.Method == http.MethodOptions {
		log.Printf("訂房 CORS 預檢請求完成：path=%s", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		log.Printf("訂房請求被拒絕：不支援 method=%s", r.Method)
		writeJSON(w, http.StatusMethodNotAllowed, SimpleResponse{Success: false, Message: "Only POST requests are allowed"})
		return
	}

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("讀取訂房請求內容失敗：err=%v", err)
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Failed to read request body: " + err.Error()})
		return
	}
	defer r.Body.Close()

	var req ReservationServices.CreateReservationRequest
	if err := json.Unmarshal(bodyData, &req); err != nil {
		log.Printf("訂房請求被拒絕：JSON 格式錯誤 err=%v", err)
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Invalid JSON format: " + err.Error()})
		return
	}
	if !requireUserWithBodyID(w, r, req.UserID) {
		log.Printf("訂房請求被拒絕：user_id 與 token 不符")
		return
	}
	log.Printf("訂房請求解析完成：user_id=%s room_id=%d booking_num=%d checkin_time=%s checkout_time=%s", req.UserID, req.RoomID, req.BookingNum, req.CheckinTime, req.CheckoutTime)

	result, err := ReservationServices.CreateReservation(db, req)
	if err != nil {
		log.Printf("訂房請求失敗：user_id=%s room_id=%d err=%v message=%q", req.UserID, req.RoomID, err, result.Message)
		switch {
		case errors.Is(err, ReservationServices.ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, ReservationServices.ErrUserNotFound):
			writeJSON(w, http.StatusNotFound, SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, ReservationServices.ErrRoomNotFound):
			writeJSON(w, http.StatusNotFound, SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, ReservationServices.ErrRoomNotAvailable):
			writeJSON(w, http.StatusConflict, SimpleResponse{Success: false, Message: result.Message})
		default:
			writeJSON(w, http.StatusInternalServerError, SimpleResponse{Success: false, Message: "Failed to create reservation"})
		}
		return
	}
	log.Printf("訂房請求成功：record_id=%s user_id=%s room_id=%s total_price=%s", result.RecordID, result.UserID, result.RoomID, result.TotalPrice)

	writeJSON(w, http.StatusOK, ReservationResponse{
		Success:    true,
		Message:    result.Message,
		RecordID:   result.RecordID,
		UserID:     result.UserID,
		HotelID:    result.HotelID,
		RoomID:     result.RoomID,
		BookingNum: result.BookingNum,
		Nights:     result.Nights,
		TotalPrice: result.TotalPrice,
	})
}
