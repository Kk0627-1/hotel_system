package main

import (
	"net/http"
	"strings"

	"hotelsystem/UserServices"
)

const corsAllowHeaders = "Content-Type, Authorization"

func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", corsAllowHeaders)
}

func userIDFromBearer(r *http.Request) (string, error) {
	// 從 Authorization 頭部取得 Bearer token
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth == "" {
		return "", UserServices.ErrInvalidToken
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return "", UserServices.ErrInvalidToken
	}
	return UserServices.ParseAccessToken(strings.TrimSpace(auth[len(prefix):]))
}

func requireUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, err := userIDFromBearer(r)
	if err != nil {
		writeUnauthorized(w, "Authorization required")
		return "", false
	}
	return userID, true
}

// requireUserWithBodyID 驗證 Bearer token，且 body 的 user_id 須與 token 中的使用者一致。
func requireUserWithBodyID(w http.ResponseWriter, r *http.Request, bodyUserID string) bool {
	tokenUserID, ok := requireUser(w, r)
	if !ok {
		return false
	}
	return assertBodyUserIDMatches(w, tokenUserID, bodyUserID)
}

// assertBodyUserIDMatches 比對 body 的 user_id 與 token 使用者是否相同（不讀 Header）。
// body 缺少 user_id 回 400；與 token 不符回 403；一致回 true。
func assertBodyUserIDMatches(w http.ResponseWriter, tokenUserID, bodyUserID string) bool {
	bodyUserID = strings.TrimSpace(bodyUserID)
	if bodyUserID == "" {
		writeJSON(w, http.StatusBadRequest, SimpleResponse{Success: false, Message: "User ID is required"})
		return false
	}
	if bodyUserID != tokenUserID {
		writeForbidden(w, "User ID does not match token")
		return false
	}
	return true
}

// resolveOptionalBodyUserID 驗證 Bearer token，並決定查詢用的 user_id（待付款／帳單等可選欄位）。
// body 未填 user_id 時回傳 token 中的使用者；有填則須與 token 一致，否則 403。
func resolveOptionalBodyUserID(w http.ResponseWriter, r *http.Request, bodyUserID string) (string, bool) {
	tokenUserID, ok := requireUser(w, r)
	if !ok {
		return "", false
	}
	bodyUserID = strings.TrimSpace(bodyUserID)
	if bodyUserID == "" {
		return tokenUserID, true
	}
	if bodyUserID != tokenUserID {
		writeForbidden(w, "User ID does not match token")
		return "", false
	}
	return bodyUserID, true
}

// writeForbidden 回傳 403：已登入但無權限（例如 body user_id 與 token 使用者不一致）。
func writeForbidden(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Forbidden"
	}
	writeJSON(w, http.StatusForbidden, SimpleResponse{Success: false, Message: message})
}

// writeUnauthorized 回傳 401：未登入。
func writeUnauthorized(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Unauthorized"
	}
	writeJSON(w, http.StatusUnauthorized, SimpleResponse{Success: false, Message: message})
}
