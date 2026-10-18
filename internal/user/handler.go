package user

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

type sessionResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
	Username string `json:"username,omitempty"`
	UserID   string `json:"user_id,omitempty"`
	Token    string `json:"token,omitempty"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		web.WriteJSON(w, http.StatusBadRequest, web.SimpleResponse{Success: false, Message: "Failed to read request body: " + err.Error()})
		return
	}
	defer r.Body.Close()
	var req RegisterRequest
	if err := json.Unmarshal(bodyData, &req); err != nil {
		web.WriteJSON(w, http.StatusBadRequest, web.SimpleResponse{Success: false, Message: "Invalid JSON format: " + err.Error()})
		return
	}

	result, err := h.service.Register(req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			web.WriteJSON(w, http.StatusBadRequest, web.SimpleResponse{Success: false, Message: err.Error()})
		case errors.Is(err, ErrUsernameExists):
			web.WriteJSON(w, http.StatusConflict, web.SimpleResponse{Success: false, Message: err.Error()})
		default:
			web.WriteJSON(w, http.StatusInternalServerError, web.SimpleResponse{Success: false, Message: "Registration failed"})
		}
		return
	}

	web.WriteJSON(
		w,
		http.StatusOK,
		sessionResponse{Success: true, Message: "User registered successfully", Username: req.Username, UserID: result.UserID},
	)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {

	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		web.WriteJSON(w, http.StatusBadRequest, web.SimpleResponse{Success: false, Message: "Failed to read request body: " + err.Error()})
		return
	}
	defer r.Body.Close()
	var req LoginRequest
	if err := json.Unmarshal(bodyData, &req); err != nil {
		web.WriteJSON(w, http.StatusBadRequest, web.SimpleResponse{Success: false, Message: "Invalid JSON format: " + err.Error()})
		return
	}

	result, err := h.service.Login(req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			web.WriteJSON(w, http.StatusBadRequest, web.SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, ErrUsernameNotFound):
			web.WriteJSON(w, http.StatusUnauthorized, web.SimpleResponse{Success: false, Message: result.Message})
		default:
			web.WriteJSON(w, http.StatusInternalServerError, web.SimpleResponse{Success: false, Message: "Login failed"})
		}
		return
	}

	web.WriteJSON(
		w,
		http.StatusOK,
		sessionResponse{Success: true, Message: result.Message, Username: result.Username, UserID: result.UserID, Token: result.Token},
	)
}
