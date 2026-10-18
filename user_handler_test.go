package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRegisterCreatesWalletOverHTTPWithMockDB(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create mock database: %v", err)
	}
	defer mockDB.Close()

	previousDB := db
	db = mockDB
	t.Cleanup(func() { db = previousDB })

	const userID = "3fa85f64-5717-4562-b3fc-2c963f66afa6"
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO users \(username, password\) VALUES \(\$1, \$2\) RETURNING id`).
		WithArgs("testuser", "123456").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(userID))
	mock.ExpectExec(`INSERT INTO wallets \(user_id\) VALUES \(\$1\)`).
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	server := httptest.NewServer((&Config{}).routes())
	defer server.Close()

	response, err := http.Post(server.URL+"/register", "application/json", strings.NewReader(`{"username":"testuser","password":"123456"}`))
	if err != nil {
		t.Fatalf("POST /register: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	var result UserSessionResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !result.Success || result.UserID != userID || result.Username != "testuser" {
		t.Fatalf("unexpected response: %+v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database flow did not match expectations: %v", err)
	}
}
