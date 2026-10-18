package httpapi

import (
	"net/http"

	"hotelsystem/internal/hotel"
	"hotelsystem/internal/payment"
	"hotelsystem/internal/reservation"
	"hotelsystem/internal/room"
	"hotelsystem/internal/user"
	"hotelsystem/internal/wallet"
)

type Handlers struct {
	User        *user.Handler
	Hotel       *hotel.Handler
	Room        *room.Handler
	Reservation *reservation.Handler
	Payment     *payment.Handler
	Wallet      *wallet.Handler
}

func NewRouter(handlers Handlers) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /register", handlers.User.Register)
	mux.HandleFunc("POST /login", handlers.User.Login)

	mux.HandleFunc("POST /hotel/create", handlers.Hotel.Create)
	mux.HandleFunc("POST /hotel/update", handlers.Hotel.Update)
	mux.HandleFunc("POST /hotel/delete", handlers.Hotel.Delete)
	mux.HandleFunc("POST /hotel/list", handlers.Hotel.List)

	mux.HandleFunc("POST /room/create", handlers.Room.Create)
	mux.HandleFunc("POST /room/update", handlers.Room.Update)
	mux.HandleFunc("POST /room/delete", handlers.Room.Delete)
	mux.HandleFunc("POST /room/list", handlers.Room.List)

	mux.HandleFunc("POST /reservation/create", handlers.Reservation.Create)

	mux.HandleFunc("POST /payment/list", handlers.Payment.ListPurchased)
	mux.HandleFunc("POST /payment/confirm", handlers.Payment.Confirm)
	mux.HandleFunc("POST /payment/cancel", handlers.Payment.Cancel)
	mux.HandleFunc("POST /payment/bill", handlers.Payment.BillReport)
	mux.HandleFunc("POST /payment/pending", handlers.Payment.ListPending)

	mux.HandleFunc("POST /wallet/recharge", handlers.Wallet.Recharge)

	return recoverPanic(cors(mux))
}
