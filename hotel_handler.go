package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"hotelsystem/HotelServices"
)

// CreateHotelHandler 創建酒店處理器
type CreateHotelHandler struct{}

// UpdateHotelHandler 更新酒店處理器
type UpdateHotelHandler struct{}

// DeleteHotelHandler 刪除酒店處理器
type DeleteHotelHandler struct{}

// ListHotelsHandler 列出酒店處理器
type ListHotelsHandler struct{}

// DeleteHotelHandler 刪除酒店處理器（業務尚未實作）
func (h DeleteHotelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, SimpleResponse{Success: false, Message: "Only POST requests are allowed"})
		return
	}

	if _, ok := requireUser(w, r); !ok {
		return
	}

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Failed to read request body: " + err.Error()})
		return
	}
	defer r.Body.Close()
	// DELETE Hotel Request
	var req HotelServices.DeleteHotelRequest
	// 解析 JSON 數據, 將 JSON 數據轉換為 CreateHotelRequest 結構體
	if err := json.Unmarshal(bodyData, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Invalid JSON format: " + err.Error()})
		return
	}
	result, err := HotelServices.DeleteHotel(db, req)
	// 根據錯誤類型返回不同的錯誤信息
	if err != nil {
		switch {
		case errors.Is(err, HotelServices.ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, HotelServices.ErrHotelNotFound):
			writeJSON(w, http.StatusNotFound, SimpleResponse{Success: false, Message: result.Message})
		default:
			writeJSON(w, http.StatusInternalServerError, SimpleResponse{Success: false, Message: "Failed to delete hotel"})
		}
		return
	}
	// 返回成功信息
	writeJSON(w, http.StatusOK, HotelWriteResponse{Success: true, Message: result.Message, HotelID: result.HotelID})
}

// CreateHotelHandler 創建飯店處理器
func (h CreateHotelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setCORS(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, SimpleResponse{Success: false, Message: "Only POST requests are allowed"})
		return
	}

	if _, ok := requireUser(w, r); !ok {
		return
	}

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Failed to read request body: " + err.Error()})
		return
	}
	defer r.Body.Close()
	var req HotelServices.CreateHotelRequest
	// 解析 JSON 數據, 將 JSON 數據轉換為 CreateHotelRequest 結構體
	if err := json.Unmarshal(bodyData, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Invalid JSON format: " + err.Error()})
		return
	}
	// 創建酒店
	result, err := HotelServices.CreateHotel(db, req)
	if err != nil {
		switch {
		case errors.Is(err, HotelServices.ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, HotelServices.ErrHotelNameExists):
			writeJSON(w, http.StatusConflict, SimpleResponse{Success: false, Message: result.Message})
		default:
			writeJSON(w, http.StatusInternalServerError, SimpleResponse{Success: false, Message: "Failed to create hotel"})
		}
		return
	}
	writeJSON(w, http.StatusOK, HotelWriteResponse{Success: true, Message: result.Message, HotelID: result.HotelID})
}

// UpdateHotelHandler 更新飯店處理器
func (h UpdateHotelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setCORS(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, SimpleResponse{Success: false, Message: "Only POST requests are allowed"})
		return
	}

	if _, ok := requireUser(w, r); !ok {
		return
	}

	bodyData, err := io.ReadAll(r.Body)
	// 檢查讀取錯誤
	if err != nil {
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Failed to read request body: " + err.Error()})
		return
	}
	defer r.Body.Close()
	var req HotelServices.UpdateHotelRequest
	if err := json.Unmarshal(bodyData, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Invalid JSON format: " + err.Error()})
		return
	}
	// 更新酒店
	result, err := HotelServices.UpdateHotel(db, req)
	if err != nil {
		switch {
		case errors.Is(err, HotelServices.ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, HotelServices.ErrHotelNameExists):
			writeJSON(w, http.StatusConflict, SimpleResponse{Success: false, Message: result.Message})
		default:
			writeJSON(w, http.StatusInternalServerError, SimpleResponse{Success: false, Message: "Failed to update hotel"})
		}
		return
	}
	writeJSON(w, http.StatusOK, HotelWriteResponse{Success: true, Message: result.Message, HotelID: result.HotelID})
}

// ListHotelsHandler 查詢／列表飯店（POST JSON：分頁與篩選條件）
func (h ListHotelsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setCORS(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, SimpleResponse{Success: false, Message: "Only POST requests are allowed"})
		return
	}

	if _, ok := requireUser(w, r); !ok {
		return
	}

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Failed to read request body: " + err.Error()})
		return
	}
	defer r.Body.Close()

	var req HotelServices.ListHotelsRequest
	if err := json.Unmarshal(bodyData, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Invalid JSON format: " + err.Error()})
		return
	}

	result, err := HotelServices.ListHotels(db, req)
	if err != nil {
		if errors.Is(err, HotelServices.ErrInvalidListParams) {
			writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: err.Error()})
		} else {
			writeJSON(w, http.StatusInternalServerError, SimpleResponse{Success: false, Message: "Failed to list hotels"})
		}
		return
	}

	writeJSON(w, http.StatusOK, HotelListResponse{
		Success:   true,
		Message:   "OK",
		Hotels:    result.Hotels,
		Total:     result.Total,
		Page:      result.Page,
		PageSize:  result.PageSize,
		TotalPage: result.TotalPages,
	})
}
