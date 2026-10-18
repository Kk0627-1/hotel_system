package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

const testUserID = "3fa85f64-5717-4562-b3fc-2c963f66afa6"

func TestCreateReservationOverHTTPWithMockDB(t *testing.T) {
	t.Log("[1/5] 建立模擬資料庫，準備訂房資料")
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create mock database: %v", err)
	}
	defer mockDB.Close()

	previousDB := db
	db = mockDB
	t.Cleanup(func() { db = previousDB })

	checkinTime := time.Now().Add(12 * time.Hour).Format("2006-01-02 15:04:05")
	checkoutTime := time.Now().Add(24 * time.Hour).Format("2006-01-02 15:04:05")
	t.Logf("[2/5] 設定預期流程：使用者=%s、房間=12、入住時間=%s、退房時間=%s", testUserID, checkinTime, checkoutTime)
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM users WHERE id = \$1\)`).
		WithArgs(testUserID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT hotel_id, price, quantity FROM rooms WHERE id = \$1 AND status = TRUE FOR UPDATE`).
		WithArgs(12).
		WillReturnRows(sqlmock.NewRows([]string{"hotel_id", "price", "quantity"}).AddRow(7, 1280.50, 2))
	mock.ExpectExec(`UPDATE rooms SET quantity = quantity - \$1 WHERE id = \$2`).
		WithArgs(1, 12).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`INSERT INTO records`).
		WithArgs(testUserID, 7, 12, sqlmock.AnyArg(), sqlmock.AnyArg(), 1280.50, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(99))
	mock.ExpectCommit()

	server := httptest.NewServer((&Config{}).routes())
	defer server.Close()
	t.Logf("[3/5] 啟動測試 API 並 POST %s/reservation/create", server.URL)

	payload := map[string]any{
		"user_id":       testUserID,
		"room_id":       12,
		"checkin_time":  checkinTime,
		"checkout_time": checkoutTime,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, server.URL+"/reservation/create", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearerTokenForTest(t, testUserID))

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /reservation/create: %v", err)
	}
	defer response.Body.Close()

	t.Logf("[4/5] 收到 HTTP %d，驗證訂房成功回應", response.StatusCode)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	var result ReservationResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !result.Success || result.RecordID != "99" || result.HotelID != "7" || result.RoomID != "12" || result.TotalPrice != "1280.50" || result.Nights != "1" || result.BookingNum != "1" {
		t.Fatalf("unexpected response: %+v", result)
	}

	t.Log("[5/5] 驗證資料庫流程：查詢使用者、鎖定房間、扣減庫存、建立訂單、提交交易")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database flow did not match expectations: %v", err)
	}
	t.Log("訂房成功流程測試完成")
}

func TestCreateReservationRejectsMismatchedUserIDOverHTTP(t *testing.T) {
	server := httptest.NewServer((&Config{}).routes())
	defer server.Close()

	checkinTime := time.Now().Add(12 * time.Hour).Format("2006-01-02 15:04:05")
	checkoutTime := time.Now().Add(24 * time.Hour).Format("2006-01-02 15:04:05")
	payload, err := json.Marshal(map[string]any{
		"user_id":       "00000000-0000-0000-0000-000000000099",
		"room_id":       12,
		"checkin_time":  checkinTime,
		"checkout_time": checkoutTime,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, server.URL+"/reservation/create", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearerTokenForTest(t, testUserID))

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /reservation/create: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusForbidden)
	}

	var result SimpleResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Success || result.Message != "User ID does not match token" {
		t.Fatalf("unexpected response: %+v", result)
	}
}

func TestCreateReservationRejectsInvalidRequestOverHTTP(t *testing.T) {
	t.Log("[1/3] 啟動測試 API，準備送出缺少 Authorization 的訂房請求")
	server := httptest.NewServer((&Config{}).routes())
	defer server.Close()

	t.Logf("[2/3] POST %s/reservation/create", server.URL)
	response, err := http.Post(server.URL+"/reservation/create", "application/json", bytes.NewBufferString(`{"room_id":0,"checkout_time":""}`))
	if err != nil {
		t.Fatalf("POST /reservation/create: %v", err)
	}
	defer response.Body.Close()

	t.Logf("[3/3] 收到 HTTP %d，驗證未授權回應", response.StatusCode)
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusUnauthorized)
	}

	var result SimpleResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Success || result.Message != "Authorization required" {
		t.Fatalf("unexpected response: %+v", result)
	}
	t.Log("未授權訂房請求測試完成")
}
