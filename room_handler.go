package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"hotelsystem/RoomServices"
)

// CreateRoomHandler 創建房間處理器
type CreateRoomHandler struct{}

// UpdateRoomHandler 更新房間處理器
type UpdateRoomHandler struct{}

// DeleteRoomHandler 刪除房間處理器
type DeleteRoomHandler struct{}

// 創建房間
func (h CreateRoomHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	var req RoomServices.CreateRoomRequest
	if err := json.Unmarshal(bodyData, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Invalid JSON format: " + err.Error()})
		return
	}
	result, err := RoomServices.CreateRoom(db, req)
	if err != nil {
		switch {
		case errors.Is(err, RoomServices.ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, RoomServices.ErrHotelNotFound):
			writeJSON(w, http.StatusNotFound, SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, RoomServices.ErrRoomTypeExists):
			writeJSON(w, http.StatusConflict, SimpleResponse{Success: false, Message: result.Message})
		default:
			writeJSON(w, http.StatusInternalServerError, SimpleResponse{Success: false, Message: "Failed to create room"})
		}
		return
	}
	writeJSON(w, http.StatusOK, RoomWriteResponse{Success: true, Message: result.Message, RoomID: result.RoomID, HotelID: result.HotelID})
}

// 更新房間
func (h UpdateRoomHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	var req RoomServices.UpdateRoomRequest
	if err := json.Unmarshal(bodyData, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Invalid JSON format: " + err.Error()})
		return
	}
	result, err := RoomServices.UpdateRoom(db, req)
	if err != nil {
		switch {
		case errors.Is(err, RoomServices.ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, RoomServices.ErrRoomNotFound):
			writeJSON(w, http.StatusNotFound, SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, RoomServices.ErrRoomTypeExists):
			writeJSON(w, http.StatusConflict, SimpleResponse{Success: false, Message: result.Message})
		default:
			writeJSON(w, http.StatusInternalServerError, SimpleResponse{Success: false, Message: "Failed to update room"})
		}
		return
	}
	writeJSON(w, http.StatusOK, RoomWriteResponse{Success: true, Message: result.Message, RoomID: result.RoomID, HotelID: result.HotelID})
}

// 刪除房間
func (h DeleteRoomHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	var req RoomServices.DeleteRoomRequest
	if err := json.Unmarshal(bodyData, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Invalid JSON format: " + err.Error()})
		return
	}
	result, err := RoomServices.DeleteRoom(db, req)
	if err != nil {
		switch {
		case errors.Is(err, RoomServices.ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, RoomServices.ErrRoomNotFound):
			writeJSON(w, http.StatusNotFound, SimpleResponse{Success: false, Message: result.Message})
		default:
			writeJSON(w, http.StatusInternalServerError, SimpleResponse{Success: false, Message: "Failed to delete room"})
		}
		return
	}
	writeJSON(w, http.StatusOK, RoomWriteResponse{Success: true, Message: result.Message, RoomID: result.RoomID})
}

// ListRoomsHandler 查詢／列表房間（POST JSON：分頁與篩選條件）
type ListRoomsHandler struct{}

func (h ListRoomsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	var req RoomServices.ListRoomsRequest
	if err := json.Unmarshal(bodyData, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Invalid JSON format: " + err.Error()})
		return
	}

	result, err := RoomServices.ListRooms(db, req)
	if err != nil {
		if errors.Is(err, RoomServices.ErrInvalidListParams) {
			writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: err.Error()})
		} else {
			writeJSON(w, http.StatusInternalServerError, SimpleResponse{Success: false, Message: "Failed to list rooms"})
		}
		return
	}

	writeJSON(w, http.StatusOK, RoomListResponse{
		Success:   true,
		Message:   "OK",
		Rooms:     result.Rooms,
		Total:     result.Total,
		Page:      result.Page,
		PageSize:  result.PageSize,
		TotalPage: result.TotalPages,
	})
}
