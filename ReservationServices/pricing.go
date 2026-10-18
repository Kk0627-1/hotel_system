package ReservationServices

import (
	"math"
	"time"
)

// ReservationNights 依入住至退房時間計算晚數（不足 24 小時以 1 晚計，其餘向上取整）
func ReservationNights(checkin, checkout time.Time) int {
	hours := checkout.Sub(checkin).Hours()
	if hours <= 0 {
		return 0
	}
	nights := int(math.Ceil(hours / 24))
	if nights < 1 {
		nights = 1
	}
	return nights
}

// ReservationTotalPrice 總價 = 單價 × 晚數 × 間數
func ReservationTotalPrice(nightlyPrice float64, nights, bookingNum int) float64 {
	return nightlyPrice * float64(nights) * float64(bookingNum)
}
