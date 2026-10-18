package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"hotelsystem/PaymentServices"
)

// ConfirmPaymentHandler 確認支付處理器
type ConfirmPaymentHandler struct{}

// ListPendingPaymentsHandler 待付款列表處理器
type ListPendingPaymentsHandler struct{}

// CancelPaymentHandler 取消未付款訂單處理器
type CancelPaymentHandler struct{}

// BillReportHandler 帳單報表處理器
type BillReportHandler struct{}

// ListPurchasedBillsHandler 帳單報表列表處理器
type ListPurchasedBillsHandler struct{}

// 確認支付（pending → paid）
func (h ConfirmPaymentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("收到確認支付請求：method=%s path=%s", r.Method, r.URL.Path)
	setCORS(w)
	if r.Method == http.MethodOptions {
		log.Printf("確認支付 CORS 預檢請求完成：path=%s", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		log.Printf("確認支付請求被拒絕：不支援 method=%s", r.Method)
		writeJSON(w, http.StatusMethodNotAllowed, SimpleResponse{Success: false, Message: "Only POST requests are allowed"})
		return
	}

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("讀取確認支付請求內容失敗：err=%v", err)
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Failed to read request body: " + err.Error()})
		return
	}
	defer r.Body.Close()

	var req PaymentServices.ConfirmPaymentRequest
	// 解析 JSON，轉成 ConfirmPaymentRequest
	if err := json.Unmarshal(bodyData, &req); err != nil {
		log.Printf("確認支付請求被拒絕：JSON 格式錯誤 err=%v", err)
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Invalid JSON format: " + err.Error()})
		return
	}
	if !requireUserWithBodyID(w, r, req.UserID) {
		log.Printf("確認支付請求被拒絕：user_id 與 token 不符")
		return
	}
	log.Printf("確認支付請求解析完成：record_id=%s user_id=%s", req.RecordID, req.UserID)

	result, err := PaymentServices.ConfirmPayment(db, req)
	if err != nil {
		log.Printf("確認支付請求失敗：record_id=%s user_id=%s err=%v message=%q", req.RecordID, req.UserID, err, result.Message)
		// 根據錯誤類型返回不同狀態碼
		switch {
		case errors.Is(err, PaymentServices.ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, PaymentServices.ErrRecordNotFound):
			writeJSON(w, http.StatusNotFound, SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, PaymentServices.ErrPaymentNotPending):
			writeJSON(w, http.StatusConflict, SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, PaymentServices.ErrWalletNotFound):
			writeJSON(w, http.StatusNotFound, SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, PaymentServices.ErrInsufficientBalance):
			writeJSON(w, http.StatusConflict, SimpleResponse{Success: false, Message: result.Message})
		default:
			writeJSON(w, http.StatusInternalServerError, SimpleResponse{Success: false, Message: "Failed to confirm payment"})
		}
		return
	}
	log.Printf("確認支付請求成功：record_id=%s user_id=%s payment_status=%s", result.RecordID, result.UserID, result.PaymentStatus)

	// 返回確認支付成功資訊
	writeJSON(w, http.StatusOK, PaymentConfirmResponse{
		Success:       true,
		Message:       result.Message,
		RecordID:      result.RecordID,
		UserID:        result.UserID,
		HotelID:       result.HotelID,
		RoomID:        result.RoomID,
		TotalPrice:    result.TotalPrice,
		PaymentType:   result.PaymentType,
		PaymentStatus: result.PaymentStatus,
	})
}

// 查詢待付款列表（POST JSON：分頁與可選 user_id）
func (h ListPendingPaymentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("收到待付款列表請求：method=%s path=%s", r.Method, r.URL.Path)
	setCORS(w)
	if r.Method == http.MethodOptions {
		log.Printf("待付款列表 CORS 預檢請求完成：path=%s", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		log.Printf("待付款列表請求被拒絕：不支援 method=%s", r.Method)
		writeJSON(w, http.StatusMethodNotAllowed, SimpleResponse{Success: false, Message: "Only POST requests are allowed"})
		return
	}

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("讀取待付款列表請求內容失敗：err=%v", err)
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Failed to read request body: " + err.Error()})
		return
	}
	defer r.Body.Close()

	var req PaymentServices.ListPendingRequest
	// 解析 JSON，轉成 ListPendingRequest
	if err := json.Unmarshal(bodyData, &req); err != nil {
		log.Printf("待付款列表請求被拒絕：JSON 格式錯誤 err=%v", err)
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Invalid JSON format: " + err.Error()})
		return
	}
	userID, ok := resolveOptionalBodyUserID(w, r, req.UserID)
	if !ok {
		log.Printf("待付款列表請求被拒絕：未授權或 user_id 與 token 不符")
		return
	}
	req.UserID = userID
	result, err := PaymentServices.ListPending(db, req)
	if err != nil {
		if errors.Is(err, PaymentServices.ErrInvalidListParams) {
			writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: err.Error()})
		} else {
			writeJSON(w, http.StatusInternalServerError, SimpleResponse{Success: false, Message: "Failed to list pending payments"})
		}
		return
	}

	// 返回分頁列表
	writeJSON(w, http.StatusOK, PaymentPendingListResponse{
		Success:   true,
		Message:   "OK",
		Records:   result.Records,
		Total:     result.Total,
		Page:      result.Page,
		PageSize:  result.PageSize,
		TotalPage: result.TotalPages,
	})
}

func (h CancelPaymentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// header設置
	setCORS(w)

	// 檢查請求類型
	if r.Method == http.MethodOptions {
		log.Printf("取消訂單 CORS 預檢請求完成：path=%s", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		log.Printf("取消訂單請求被拒絕：不支援 method=%s", r.Method)
		writeJSON(w, http.StatusMethodNotAllowed, SimpleResponse{Success: false, Message: "Only POST requests are allowed"})
		return
	}

	// 讀取請求內容
	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("讀取取消訂單請求內容失敗：err=%v", err)
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Failed to read request body: " + err.Error()})
		return
	}

	defer r.Body.Close()

	var req PaymentServices.CancelPaymentRequest
	if err := json.Unmarshal(bodyData, &req); err != nil {
		log.Printf("取消訂單請求被拒絕：JSON 格式錯誤 err=%v", err)
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Invalid JSON format: " + err.Error()})
		return
	}
	if !requireUserWithBodyID(w, r, req.UserID) {
		log.Printf("取消訂單請求被拒絕：user_id 與 token 不符")
		return
	}
	log.Printf("取消訂單請求解析完成：record_id=%s user_id=%s", req.RecordID, req.UserID)

	result, err := PaymentServices.CancelPayment(db, req)
	if err != nil {
		log.Printf("取消訂單請求失敗：record_id=%s err=%v message=%q", req.RecordID, err, result.Message)
		switch {
		case errors.Is(err, PaymentServices.ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, PaymentServices.ErrRecordNotFound):
			writeJSON(w, http.StatusNotFound, SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, PaymentServices.ErrPaymentNotPending), errors.Is(err, PaymentServices.ErrOrderDisabled):
			writeJSON(w, http.StatusConflict, SimpleResponse{Success: false, Message: result.Message})
		default:
			writeJSON(w, http.StatusInternalServerError, SimpleResponse{Success: false, Message: "Failed to cancel payment"})
		}
		return
	}
	log.Printf("取消訂單請求成功：record_id=%s", req.RecordID)
	writeJSON(w, http.StatusOK, SimpleResponse{Success: true, Message: result.Message})
}

func (h ListPurchasedBillsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// header設置
	setCORS(w)
	// 檢查請求類型
	if r.Method == http.MethodOptions {
		log.Printf("訂單報表 CORS 預檢請求完成：path=%s", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		log.Printf("訂單報表請求被拒絕：不支援 method=%s", r.Method)
		writeJSON(w, http.StatusMethodNotAllowed, SimpleResponse{Success: false, Message: "Only POST requests are allowed"})
		return
	}

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("讀取訂單報表請求內容失敗：err=%v", err)
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Failed to read request body: " + err.Error()})
		return
	}
	defer r.Body.Close()

	var req PaymentServices.PurchasedBillRequest
	if err := json.Unmarshal(bodyData, &req); err != nil {
		log.Printf("訂單報表請求被拒絕：JSON 格式錯誤 err=%v", err)
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Invalid JSON format: " + err.Error()})
		return
	}
	userID, ok := resolveOptionalBodyUserID(w, r, req.UserID)
	if !ok {
		log.Printf("訂單報表請求被拒絕：未授權或 user_id 與 token 不符")
		return
	}
	req.UserID = userID
	result, err := PaymentServices.ListPurchasedBills(db, req)
	if err != nil {
		log.Printf("訂單報表請求失敗：err=%v", err)
		if errors.Is(err, PaymentServices.ErrInvalidListParams) {
			writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: err.Error()})
		} else {
			writeJSON(w, http.StatusInternalServerError, SimpleResponse{Success: false, Message: "Failed to list bills"})
		}
		return
	}

	writeJSON(w, http.StatusOK, PaymentBillListResponse{
		Success:   true,
		Message:   "OK",
		List:      result.List,
		Total:     result.Total,
		Page:      result.Page,
		PageSize:  result.PageSize,
		TotalPage: result.TotalPages,
	})
}

func (h BillReportHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// header設置
	setCORS(w)
	// 檢查請求類型
	if r.Method == http.MethodOptions {
		log.Printf("訂單報表 CORS 預檢請求完成：path=%s", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		log.Printf("訂單報表請求被拒絕：不支援 method=%s", r.Method)
		writeJSON(w, http.StatusMethodNotAllowed, SimpleResponse{Success: false, Message: "Only POST requests are allowed"})
		return
	}

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("讀取訂單報表請求內容失敗：err=%v", err)
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Failed to read request body: " + err.Error()})
		return
	}
	defer r.Body.Close()

	var req PaymentServices.PurchasedBillRequest
	if err := json.Unmarshal(bodyData, &req); err != nil {
		log.Printf("訂單報表請求被拒絕：JSON 格式錯誤 err=%v", err)
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Invalid JSON format: " + err.Error()})
		return
	}
	userID, ok := resolveOptionalBodyUserID(w, r, req.UserID)
	if !ok {
		log.Printf("訂單報表請求被拒絕：未授權或 user_id 與 token 不符")
		return
	}
	req.UserID = userID
	result, err := PaymentServices.ListPurchasedBills(db, req)
	if err != nil {
		log.Printf("訂單報表請求失敗：err=%v", err)
		if errors.Is(err, PaymentServices.ErrInvalidListParams) {
			writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: err.Error()})
		} else {
			writeJSON(w, http.StatusInternalServerError, SimpleResponse{Success: false, Message: "Failed to list bills"})
		}
		return
	}

	writeJSON(w, http.StatusOK, PaymentBillListResponse{
		Success:   true,
		Message:   "OK",
		List:      result.List,
		Total:     result.Total,
		Page:      result.Page,
		PageSize:  result.PageSize,
		TotalPage: result.TotalPages,
	})
}
