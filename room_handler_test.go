package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestDeleteRoomOverHTTPWithMockDB(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create mock database: %v", err)
	}
	defer mockDB.Close()

	previousDB := db
	db = mockDB
	t.Cleanup(func() { db = previousDB })

	mock.ExpectQuery(`DELETE FROM rooms WHERE id = \$1 RETURNING id, hotel_id`).
		WithArgs("1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "hotel_id"}).AddRow(1, 7))

	server := httptest.NewServer((&Config{}).routes())
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/room/delete", bytes.NewBufferString(`{"room_id":"1"}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearerTokenForTest(t, testUserID))

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /room/delete: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	var result RoomWriteResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !result.Success || result.RoomID != "1" || result.Message != "Room deleted successfully" {
		t.Fatalf("unexpected response: %+v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database flow did not match expectations: %v", err)
	}
}

func TestDeleteRoomRejectsInvalidRequestOverHTTP(t *testing.T) {
	server := httptest.NewServer((&Config{}).routes())
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/room/delete", bytes.NewBufferString(`{"room_id":""}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearerTokenForTest(t, testUserID))

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /room/delete: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusBadRequest)
	}

	var result SimpleResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Success || result.Message != "Room ID is required" {
		t.Fatalf("unexpected response: %+v", result)
	}
}
