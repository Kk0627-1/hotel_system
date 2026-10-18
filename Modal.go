package main

import (
	"hotelsystem/HotelServices"
	"hotelsystem/PaymentServices"
	"hotelsystem/RoomServices"
)

// SimpleResponse 通用：錯誤提示或僅含說明文字的成功回應
type SimpleResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// UserSessionResponse 註冊／登入成功
type UserSessionResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
	Username string `json:"username,omitempty"`
	UserID   string `json:"user_id,omitempty"`
	Token    string `json:"token,omitempty"`
}

// HotelWriteResponse 建立／更新酒店成功
type HotelWriteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	HotelID string `json:"hotel_id,omitempty"`
}

// RoomWriteResponse 建立／更新房間成功
type RoomWriteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	RoomID  string `json:"room_id,omitempty"`
	HotelID string `json:"hotel_id,omitempty"`
}

// RoomListResponse 房間分頁列表
type RoomListResponse struct {
	Success   bool                `json:"success"`
	Message   string              `json:"message,omitempty"`
	Rooms     []RoomServices.Room `json:"rooms,omitempty"`
	Total     int                 `json:"total,omitempty"`
	Page      int                 `json:"page,omitempty"`
	PageSize  int                 `json:"page_size,omitempty"`
	TotalPage int                 `json:"total_page,omitempty"`
}

// HotelListResponse 酒店分頁列表
type HotelListResponse struct {
	Success   bool                  `json:"success"`
	Message   string                `json:"message,omitempty"`
	Hotels    []HotelServices.Hotel `json:"hotels,omitempty"`
	Total     int                   `json:"total,omitempty"`
	Page      int                   `json:"page,omitempty"`
	PageSize  int                   `json:"page_size,omitempty"`
	TotalPage int                   `json:"total_page,omitempty"`
}

// ReservationResponse 訂房成功回應
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

// PaymentConfirmResponse 確認支付成功回應
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

// PaymentBillListResponse 帳單報表分頁列表
type PaymentBillListResponse struct {
	Success   bool                            `json:"success"`
	Message   string                          `json:"message,omitempty"`
	List      []PaymentServices.PurchasedBill `json:"list"`
	Total     int                             `json:"total,omitempty"`
	Page      int                             `json:"page,omitempty"`
	PageSize  int                             `json:"page_size,omitempty"`
	TotalPage int                             `json:"total_page,omitempty"`
}

// PaymentPendingListResponse 待付款分頁列表
type PaymentPendingListResponse struct {
	Success   bool                            `json:"success"`
	Message   string                          `json:"message,omitempty"`
	Records   []PaymentServices.PendingRecord `json:"records"`
	Total     int                             `json:"total,omitempty"`
	Page      int                             `json:"page,omitempty"`
	PageSize  int                             `json:"page_size,omitempty"`
	TotalPage int                             `json:"total_page,omitempty"`
}
