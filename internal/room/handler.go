package room

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"hotelsystem/internal/web"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

type writeResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	RoomID  string `json:"room_id,omitempty"`
	HotelID string `json:"hotel_id,omitempty"`
}

type listResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	Rooms     []Room `json:"rooms,omitempty"`
	Total     int    `json:"total,omitempty"`
	Page      int    `json:"page,omitempty"`
	PageSize  int    `json:"page_size,omitempty"`
	TotalPage int    `json:"total_page,omitempty"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if _, ok := web.RequireUser(w, r); !ok {
		return
	}
	var req CreateRoomRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := h.service.Create(req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			web.WriteJSON(w, http.StatusBadRequest, web.SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, ErrHotelNotFound):
			web.WriteJSON(w, http.StatusNotFound, web.SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, ErrRoomTypeExists):
			web.WriteJSON(w, http.StatusConflict, web.SimpleResponse{Success: false, Message: result.Message})
		default:
			web.WriteJSON(w, http.StatusInternalServerError, web.SimpleResponse{Success: false, Message: "Failed to create room"})
		}
		return
	}
	web.WriteJSON(w, http.StatusOK, writeResponse{Success: true, Message: result.Message, RoomID: result.RoomID, HotelID: result.HotelID})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if _, ok := web.RequireUser(w, r); !ok {
		return
	}
	var req UpdateRoomRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := h.service.Update(req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			web.WriteJSON(w, http.StatusBadRequest, web.SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, ErrRoomNotFound):
			web.WriteJSON(w, http.StatusNotFound, web.SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, ErrRoomTypeExists):
			web.WriteJSON(w, http.StatusConflict, web.SimpleResponse{Success: false, Message: result.Message})
		default:
			web.WriteJSON(w, http.StatusInternalServerError, web.SimpleResponse{Success: false, Message: "Failed to update room"})
		}
		return
	}
	web.WriteJSON(w, http.StatusOK, writeResponse{Success: true, Message: result.Message, RoomID: result.RoomID, HotelID: result.HotelID})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if _, ok := web.RequireUser(w, r); !ok {
		return
	}
	var req DeleteRoomRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := h.service.Delete(req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			web.WriteJSON(w, http.StatusBadRequest, web.SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, ErrRoomNotFound):
			web.WriteJSON(w, http.StatusNotFound, web.SimpleResponse{Success: false, Message: result.Message})
		default:
			web.WriteJSON(w, http.StatusInternalServerError, web.SimpleResponse{Success: false, Message: "Failed to delete room"})
		}
		return
	}
	web.WriteJSON(w, http.StatusOK, writeResponse{Success: true, Message: result.Message, RoomID: result.RoomID})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if _, ok := web.RequireUser(w, r); !ok {
		return
	}
	var req ListRoomsRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := h.service.List(req)
	if err != nil {
		if errors.Is(err, ErrInvalidListParams) {
			web.WriteJSON(w, http.StatusBadRequest, web.SimpleResponse{Success: false, Message: err.Error()})
		} else {
			web.WriteJSON(w, http.StatusInternalServerError, web.SimpleResponse{Success: false, Message: "Failed to list rooms"})
		}
		return
	}
	web.WriteJSON(w, http.StatusOK, listResponse{
		Success: true, Message: "OK", Rooms: result.Rooms,
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
