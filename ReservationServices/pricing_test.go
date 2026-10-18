package ReservationServices

import (
	"testing"
	"time"
)

func TestReservationNightsAndTotalPrice(t *testing.T) {
	checkin := time.Date(2026, 1, 1, 14, 0, 0, 0, time.UTC)
	checkout12h := checkin.Add(12 * time.Hour)
	if got := ReservationNights(checkin, checkout12h); got != 1 {
		t.Fatalf("nights(12h) = %d, want 1", got)
	}
	checkout36h := checkin.Add(36 * time.Hour)
	if got := ReservationNights(checkin, checkout36h); got != 2 {
		t.Fatalf("nights(36h) = %d, want 2", got)
	}
	total := ReservationTotalPrice(100, 2, 3)
	if total != 600 {
		t.Fatalf("total = %v, want 600", total)
	}
}
