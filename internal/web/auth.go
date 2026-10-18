package web

import (
	"net/http"
	"strings"

	"hotelsystem/internal/auth"
)

func UserIDFromBearer(r *http.Request) (string, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", auth.ErrInvalidToken
	}
	return auth.ParseAccessToken(strings.TrimSpace(header[len(prefix):]))
}

func RequireUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, err := UserIDFromBearer(r)
	if err != nil {
		WriteJSON(w, http.StatusUnauthorized, SimpleResponse{Success: false, Message: "Authorization required"})
		return "", false
	}
	return userID, true
}

func RequireMatchingUser(w http.ResponseWriter, r *http.Request, bodyUserID string) bool {
	tokenUserID, ok := RequireUser(w, r)
	if !ok {
		return false
	}
	bodyUserID = strings.TrimSpace(bodyUserID)
	if bodyUserID == "" {
		WriteJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "User ID is required"})
		return false
	}
	if bodyUserID != tokenUserID {
		WriteJSON(w, http.StatusForbidden, SimpleResponse{Success: false, Message: "User ID does not match token"})
		return false
	}
	return true
}

func ResolveUserID(w http.ResponseWriter, r *http.Request, bodyUserID string) (string, bool) {
	tokenUserID, ok := RequireUser(w, r)
	if !ok {
		return "", false
	}
	bodyUserID = strings.TrimSpace(bodyUserID)
	if bodyUserID == "" {
		return tokenUserID, true
	}
	if bodyUserID != tokenUserID {
		WriteJSON(w, http.StatusForbidden, SimpleResponse{Success: false, Message: "User ID does not match token"})
		return "", false
	}
	return bodyUserID, true
}
