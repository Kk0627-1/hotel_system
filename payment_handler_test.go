package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// TestConfirmPaymentOverHTTPWithMockDB 模擬資料庫，驗證確認支付成功（pending → paid）
func TestConfirmPaymentOverHTTPWithMockDB(t *testing.T) {
	t.Log("[1/5] 建立模擬資料庫，準備待付款訂單")
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create mock database: %v", err)
	}
	defer mockDB.Close()

	previousDB := db
	db = mockDB
	t.Cleanup(func() { db = previousDB })

	t.Logf("[2/5] 設定預期流程：record_id=99、使用者=%s", testUserID)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT user_id::text, hotel_id, room_id, total_price, payment_type, payment_status, status FROM records WHERE id = \$1 FOR UPDATE`).
		WithArgs(99).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "hotel_id", "room_id", "total_price", "payment_type", "payment_status", "status"}).
			AddRow(testUserID, 7, 12, 1280.50, "cash", "pending", true))
	mock.ExpectExec(`INSERT INTO wallets \(user_id\) VALUES \(\$1\) ON CONFLICT \(user_id\) DO NOTHING`).
		WithArgs(testUserID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT balance FROM wallets WHERE user_id = \$1 FOR UPDATE`).
		WithArgs(testUserID).
		WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(500.00))
	mock.ExpectQuery(`UPDATE records SET payment_status = 'paid' WHERE id = \$1 AND user_id = \$2 AND payment_status = 'pending' AND status = TRUE RETURNING id`).
		WithArgs(99, testUserID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(99))
	mock.ExpectCommit()

	server := httptest.NewServer((&Config{}).routes())
	defer server.Close()
	t.Logf("[3/5] 啟動測試 API 並 POST %s/payment/confirm", server.URL)

	payload := map[string]any{
		"record_id": "99",
		"user_id":   testUserID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, server.URL+"/payment/confirm", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearerTokenForTest(t, testUserID))
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /payment/confirm: %v", err)
	}
	defer response.Body.Close()

	t.Logf("[4/5] 收到 HTTP %d，驗證確認支付成功回應", response.StatusCode)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	var result PaymentConfirmResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !result.Success || result.RecordID != "99" || result.HotelID != "7" || result.RoomID != "12" || result.TotalPrice != "1280.50" || result.PaymentStatus != "paid" {
		t.Fatalf("unexpected response: %+v", result)
	}

	t.Log("[5/5] 驗證資料庫流程：查詢訂單、更新付款狀態")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database flow did not match expectations: %v", err)
	}
	t.Log("確認支付成功流程測試完成")
}

// TestConfirmPaymentRejectsInvalidRequestOverHTTP 缺少 record_id 應回 400
func TestConfirmPaymentRejectsInvalidRequestOverHTTP(t *testing.T) {
	t.Log("[1/3] 啟動測試 API，準備送出缺少必要欄位的確認支付請求")
	server := httptest.NewServer((&Config{}).routes())
	defer server.Close()

	t.Logf("[2/3] POST %s/payment/confirm", server.URL)
	req, err := http.NewRequest(http.MethodPost, server.URL+"/payment/confirm", bytes.NewBufferString(`{"record_id":"","user_id":"`+testUserID+`"}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearerTokenForTest(t, testUserID))
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /payment/confirm: %v", err)
	}
	defer response.Body.Close()

	t.Logf("[3/3] 收到 HTTP %d，驗證輸入錯誤回應", response.StatusCode)
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusBadRequest)
	}

	var result SimpleResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Success || result.Message != "Record ID is required" {
		t.Fatalf("unexpected response: %+v", result)
	}
	t.Log("無效確認支付請求測試完成")
}

// TestListPendingPaymentsOverHTTPWithMockDB 模擬資料庫，驗證待付款列表查詢
func TestListPendingPaymentsOverHTTPWithMockDB(t *testing.T) {
	t.Log("[1/5] 建立模擬資料庫，準備待付款列表")
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create mock database: %v", err)
	}
	defer mockDB.Close()

	previousDB := db
	db = mockDB
	t.Cleanup(func() { db = previousDB })

	bookingTime := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	t.Logf("[2/5] 設定預期流程：使用者=%s、page=1", testUserID)
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM records WHERE payment_status = 'pending' AND status = TRUE AND user_id = \$1`).
		WithArgs(testUserID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT id, user_id::text, hotel_id, room_id, total_price, payment_type, payment_status, booking_time FROM records WHERE payment_status = 'pending' AND status = TRUE AND user_id = \$1 ORDER BY booking_time DESC LIMIT \$2 OFFSET \$3`).
		WithArgs(testUserID, 10, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "hotel_id", "room_id", "total_price", "payment_type", "payment_status", "booking_time"}).
			AddRow(99, testUserID, 7, 12, 1280.50, "cash", "pending", bookingTime))

	server := httptest.NewServer((&Config{}).routes())
	defer server.Close()
	t.Logf("[3/5] 啟動測試 API 並 POST %s/payment/pending", server.URL)

	payload := map[string]any{
		"page":      1,
		"page_size": 10,
		"user_id":   testUserID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, server.URL+"/payment/pending", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearerTokenForTest(t, testUserID))
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /payment/pending: %v", err)
	}
	defer response.Body.Close()

	t.Logf("[4/5] 收到 HTTP %d，驗證待付款列表回應", response.StatusCode)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	var result PaymentPendingListResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !result.Success || result.Total != 1 || len(result.Records) != 1 || result.Records[0].RecordID != "99" || result.Records[0].PaymentStatus != "pending" {
		t.Fatalf("unexpected response: %+v", result)
	}

	t.Log("[5/5] 驗證資料庫流程：COUNT、SELECT 待付款")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database flow did not match expectations: %v", err)
	}
	t.Log("待付款列表成功流程測試完成")
}

// TestListPendingPaymentsRejectsInvalidRequestOverHTTP page < 1 應回 400
func TestListPendingPaymentsRejectsInvalidRequestOverHTTP(t *testing.T) {
	t.Log("[1/3] 啟動測試 API，準備送出無效分頁參數")
	server := httptest.NewServer((&Config{}).routes())
	defer server.Close()

	t.Logf("[2/3] POST %s/payment/pending", server.URL)
	req, err := http.NewRequest(http.MethodPost, server.URL+"/payment/pending", bytes.NewBufferString(`{"page":0,"page_size":10}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearerTokenForTest(t, testUserID))
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /payment/pending: %v", err)
	}
	defer response.Body.Close()

	t.Logf("[3/3] 收到 HTTP %d，驗證輸入錯誤回應", response.StatusCode)
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusBadRequest)
	}

	var result SimpleResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Success {
		t.Fatalf("unexpected response: %+v", result)
	}
	t.Log("無效待付款列表請求測試完成")
}
