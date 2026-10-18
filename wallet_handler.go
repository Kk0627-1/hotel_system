package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"hotelsystem/WalletService"
)

// WalletHandler 錢包儲值處理器
type WalletRechargeHandler struct{}

// 錢包列表
func (h WalletRechargeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, SimpleResponse{Success: false, Message: "Only POST requests are allowed"})
		return
	}

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Failed to read request body: " + err.Error()})
		return
	}
	defer r.Body.Close()
	var req WalletService.RechargeRequest
	if err := json.Unmarshal(bodyData, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Invalid JSON format: " + err.Error()})
		return
	}
	if !requireUserWithBodyID(w, r, req.UserID) {
		return
	}
	result, err := WalletService.Recharge(db, req)
	if err != nil {
		switch {
		case errors.Is(err, WalletService.ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, WalletService.ErrUserNotFound), errors.Is(err, WalletService.ErrWalletNotFound):
			writeJSON(w, http.StatusNotFound, SimpleResponse{Success: false, Message: result.Message})
		default:
			writeJSON(w, http.StatusInternalServerError, SimpleResponse{Success: false, Message: "Failed to recharge"})
		}
		return
	}

	writeJSON(w, http.StatusOK, WalletService.RechargeResponse{Success: result.Success, Message: result.Message})
}
