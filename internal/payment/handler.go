package payment

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

type confirmResponse struct {
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

type pendingListResponse struct {
	Success   bool            `json:"success"`
	Message   string          `json:"message,omitempty"`
	Records   []PendingRecord `json:"records"`
	Total     int             `json:"total,omitempty"`
	Page      int             `json:"page,omitempty"`
	PageSize  int             `json:"page_size,omitempty"`
	TotalPage int             `json:"total_page,omitempty"`
}

type billListResponse struct {
	Success   bool            `json:"success"`
	Message   string          `json:"message,omitempty"`
	List      []PurchasedBill `json:"list"`
	Total     int             `json:"total,omitempty"`
	Page      int             `json:"page,omitempty"`
	PageSize  int             `json:"page_size,omitempty"`
	TotalPage int             `json:"total_page,omitempty"`
}

func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	var req ConfirmPaymentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !web.RequireMatchingUser(w, r, req.UserID) {
		return
	}
	result, err := h.service.Confirm(req)
	if err != nil {
		log.Printf("confirm payment failed: record_id=%s err=%v", req.RecordID, err)
		switch {
		case errors.Is(err, ErrInvalidInput):
			web.WriteJSON(w, http.StatusBadRequest, web.SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, ErrRecordNotFound), errors.Is(err, ErrWalletNotFound):
			web.WriteJSON(w, http.StatusNotFound, web.SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, ErrPaymentNotPending), errors.Is(err, ErrInsufficientBalance):
			web.WriteJSON(w, http.StatusConflict, web.SimpleResponse{Success: false, Message: result.Message})
		default:
			web.WriteJSON(w, http.StatusInternalServerError, web.SimpleResponse{Success: false, Message: "Failed to confirm payment"})
		}
		return
	}
	web.WriteJSON(w, http.StatusOK, confirmResponse{
		Success: true, Message: result.Message, RecordID: result.RecordID, UserID: result.UserID,
		HotelID: result.HotelID, RoomID: result.RoomID, TotalPrice: result.TotalPrice,
		PaymentType: result.PaymentType, PaymentStatus: result.PaymentStatus,
	})
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	var req CancelPaymentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !web.RequireMatchingUser(w, r, req.UserID) {
		return
	}
	result, err := h.service.Cancel(req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			web.WriteJSON(w, http.StatusBadRequest, web.SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, ErrRecordNotFound):
			web.WriteJSON(w, http.StatusNotFound, web.SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, ErrPaymentNotPending), errors.Is(err, ErrOrderDisabled):
			web.WriteJSON(w, http.StatusConflict, web.SimpleResponse{Success: false, Message: result.Message})
		default:
			web.WriteJSON(w, http.StatusInternalServerError, web.SimpleResponse{Success: false, Message: "Failed to cancel payment"})
		}
		return
	}
	web.WriteJSON(w, http.StatusOK, web.SimpleResponse{Success: true, Message: result.Message})
}

func (h *Handler) ListPending(w http.ResponseWriter, r *http.Request) {
	var req ListPendingRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, ok := web.ResolveUserID(w, r, req.UserID)
	if !ok {
		return
	}
	req.UserID = userID
	result, err := h.service.ListPending(req)
	if err != nil {
		if errors.Is(err, ErrInvalidListParams) {
			web.WriteJSON(w, http.StatusBadRequest, web.SimpleResponse{Success: false, Message: err.Error()})
		} else {
			web.WriteJSON(w, http.StatusInternalServerError, web.SimpleResponse{Success: false, Message: "Failed to list pending payments"})
		}
		return
	}
	web.WriteJSON(w, http.StatusOK, pendingListResponse{
		Success: true, Message: "OK", Records: result.Records,
		Total: result.Total, Page: result.Page, PageSize: result.PageSize, TotalPage: result.TotalPages,
	})
}

func (h *Handler) ListPurchased(w http.ResponseWriter, r *http.Request) {
	h.listBills(w, r)
}

func (h *Handler) BillReport(w http.ResponseWriter, r *http.Request) {
	h.listBills(w, r)
}

func (h *Handler) listBills(w http.ResponseWriter, r *http.Request) {
	var req PurchasedBillRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, ok := web.ResolveUserID(w, r, req.UserID)
	if !ok {
		return
	}
	req.UserID = userID
	result, err := h.service.ListPurchasedBills(req)
	if err != nil {
		if errors.Is(err, ErrInvalidListParams) {
			web.WriteJSON(w, http.StatusBadRequest, web.SimpleResponse{Success: false, Message: err.Error()})
		} else {
			web.WriteJSON(w, http.StatusInternalServerError, web.SimpleResponse{Success: false, Message: "Failed to list bills"})
		}
		return
	}
	web.WriteJSON(w, http.StatusOK, billListResponse{
		Success: true, Message: "OK", List: result.List,
		Total: result.Total, Page: result.Page, PageSize: result.PageSize, TotalPage: result.TotalPages,
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dest any) bool {
	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		web.WriteJSON(w, http.StatusBadRequest, web.SimpleResponse{Success: false, Message: "Failed to read request body: " + err.Error()})
		return false
	}
	defer r.Body.Close()
	if err := json.Unmarshal(bodyData, dest); err != nil {
		web.WriteJSON(w, http.StatusBadRequest, web.SimpleResponse{Success: false, Message: "Invalid JSON format: " + err.Error()})
		return false
	}
	return true
}
