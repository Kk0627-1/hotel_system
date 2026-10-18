package httpapi_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hotelsystem/internal/config"
	"hotelsystem/internal/platform/database"

	_ "github.com/lib/pq"
)

// TestCreateReservationIntegration verifies the reservation flow against PostgreSQL.
// It skips when the database from the environment configuration is unreachable.
func TestCreateReservationIntegration(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	connectionString := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)
	testDB, err := sql.Open("postgres", connectionString)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer testDB.Close()
	if err := testDB.Ping(); err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}

	previousDB := db
	db = testDB
	t.Cleanup(func() { db = previousDB })

	if err := database.InitSchema(testDB); err != nil {
		t.Fatalf("initialize schema: %v", err)
	}

	uniqueSuffix := time.Now().UnixNano()
	userID := fmt.Sprintf("00000000-0000-4000-8000-%012x", uniqueSuffix&0xffffffffffff)
	username := fmt.Sprintf("reservation_test_%d", uniqueSuffix)
	hotelName := fmt.Sprintf("Reservation Test Hotel %d", uniqueSuffix)

	if _, err := testDB.Exec(`INSERT INTO users (id, username, password) VALUES ($1, $2, $3)`, userID, username, "test-password"); err != nil {
		t.Fatalf("create test user: %v", err)
	}
	var hotelID, roomID int
	if err := testDB.QueryRow(`INSERT INTO hotels (name, status) VALUES ($1, TRUE) RETURNING id`, hotelName).Scan(&hotelID); err != nil {
		t.Fatalf("create test hotel: %v", err)
	}
	if err := testDB.QueryRow(`INSERT INTO rooms (hotel_id, room_type, price, quantity, status) VALUES ($1, $2, $3, $4, TRUE) RETURNING id`, hotelID, "integration-test", 1280.50, 2).Scan(&roomID); err != nil {
		t.Fatalf("create test room: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testDB.Exec(`DELETE FROM records WHERE room_id = $1`, roomID)
		_, _ = testDB.Exec(`DELETE FROM rooms WHERE id = $1`, roomID)
		_, _ = testDB.Exec(`DELETE FROM hotels WHERE id = $1`, hotelID)
		_, _ = testDB.Exec(`DELETE FROM users WHERE id = $1`, userID)
	})

	server := httptest.NewServer((&Config{}).routes())
	defer server.Close()
	checkinTime := time.Now().Add(12 * time.Hour).Format("2006-01-02 15:04:05")
	checkoutTime := time.Now().Add(24 * time.Hour).Format("2006-01-02 15:04:05")
	payload, err := json.Marshal(map[string]any{
		"user_id":       userID,
		"room_id":       roomID,
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
	req.Header.Set("Authorization", bearerTokenForTest(t, userID))
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /reservation/create: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	var result ReservationResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !result.Success || result.UserID != userID || result.RoomID != fmt.Sprint(roomID) {
		t.Fatalf("unexpected API response: %+v", result)
	}

	var savedUserID, totalPrice string
	var savedHotelID, savedRoomID, bookingNum int
	if err := testDB.QueryRow(`SELECT user_id::text, hotel_id, room_id, total_price::text, booking_num FROM records WHERE id = $1`, result.RecordID).
		Scan(&savedUserID, &savedHotelID, &savedRoomID, &totalPrice, &bookingNum); err != nil {
		t.Fatalf("query created reservation: %v", err)
	}
	if savedUserID != userID || savedHotelID != hotelID || savedRoomID != roomID || totalPrice != "1280.50" || bookingNum != 1 {
		t.Fatalf("unexpected record: user=%s hotel=%d room=%d price=%s booking_num=%d", savedUserID, savedHotelID, savedRoomID, totalPrice, bookingNum)
	}

	var quantity int
	if err := testDB.QueryRow(`SELECT quantity FROM rooms WHERE id = $1`, roomID).Scan(&quantity); err != nil {
		t.Fatalf("query room quantity: %v", err)
	}
	if quantity != 1 {
		t.Fatalf("room quantity = %d, want 1", quantity)
	}
}
