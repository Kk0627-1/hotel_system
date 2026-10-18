package wallet

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

func (h *Handler) Recharge(w http.ResponseWriter, r *http.Request) {
	bodyData, err := io.ReadAll(r.Body)
	if err != nil {
		web.WriteJSON(w, http.StatusBadRequest, web.SimpleResponse{Success: false, Message: "Failed to read request body: " + err.Error()})
		return
	}
	defer r.Body.Close()

	var req RechargeRequest
	if err := json.Unmarshal(bodyData, &req); err != nil {
		web.WriteJSON(w, http.StatusBadRequest, web.SimpleResponse{Success: false, Message: "Invalid JSON format: " + err.Error()})
		return
	}
	if !web.RequireMatchingUser(w, r, req.UserID) {
		return
	}

	result, err := h.service.Recharge(req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			web.WriteJSON(w, http.StatusBadRequest, web.SimpleResponse{Success: false, Message: result.Message})
		case errors.Is(err, ErrUserNotFound), errors.Is(err, ErrWalletNotFound):
			web.WriteJSON(w, http.StatusNotFound, web.SimpleResponse{Success: false, Message: result.Message})
		default:
			web.WriteJSON(w, http.StatusInternalServerError, web.SimpleResponse{Success: false, Message: "Failed to recharge"})
		}
		return
	}
	web.WriteJSON(w, http.StatusOK, RechargeResponse{Success: result.Success, Message: result.Message})
}
