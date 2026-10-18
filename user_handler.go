package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"hotelsystem/UserServices"
)

// RegisterHandler 註冊處理器
type RegisterHandler struct {}
// LoginHandler 登入處理器
type LoginHandler struct {}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	b, _ := json.Marshal(body)
	_, _ = w.Write(b)
}

func (h RegisterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	var req UserServices.RegisterRequest
	if err := json.Unmarshal(bodyData, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Invalid JSON format: " + err.Error()})
		return
	}

	result, err := UserServices.RegisterUser(db, req)
	if err != nil {
		switch {
		case errors.Is(err, UserServices.ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: err.Error()})
		case errors.Is(err, UserServices.ErrUsernameExists):
			writeJSON(w, http.StatusConflict, SimpleResponse{Success: false, Message: err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, SimpleResponse{Success: false, Message: "Registration failed"})
		}
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		UserSessionResponse{Success: true, Message: "User registered successfully", Username: req.Username, UserID: result.UserID},
	)
}

func (h LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	var req UserServices.LoginRequest
	if err := json.Unmarshal(bodyData, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "Invalid JSON format: " + err.Error()})
		return
	}

	result, err := UserServices.LoginUser(db, req)
	if err != nil {
		switch {
		case errors.Is(err, UserServices.ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, UserServices.ErrUsernameNotFound):
			writeJSON(w, http.StatusUnauthorized, SimpleResponse{Success: false, Message: result.Message})
		default:
			writeJSON(w, http.StatusInternalServerError, SimpleResponse{Success: false, Message: "Login failed"})
		}
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		UserSessionResponse{Success: true, Message: result.Message, Username: result.Username, UserID: result.UserID, Token: result.Token},
	)
}