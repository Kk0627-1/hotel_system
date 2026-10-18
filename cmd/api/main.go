package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hotelsystem/internal/config"
	"hotelsystem/internal/hotel"
	"hotelsystem/internal/httpapi"
	"hotelsystem/internal/payment"
	"hotelsystem/internal/platform/database"
	"hotelsystem/internal/reservation"
	"hotelsystem/internal/room"
	"hotelsystem/internal/user"
	"hotelsystem/internal/wallet"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	log.Println("diff test")

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	startupCtx, cancelStartup := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelStartup()
	db, err := database.Open(startupCtx, cfg.Database)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := database.InitSchema(db); err != nil {
		return err
	}

	userHandler := user.NewHandler(user.NewService(user.NewRepository(db)))
	hotelHandler := hotel.NewHandler(hotel.NewService(hotel.NewRepository(db)))
	roomHandler := room.NewHandler(room.NewService(room.NewRepository(db)))
	reservationHandler := reservation.NewHandler(reservation.NewService(reservation.NewRepository(db)))
	paymentHandler := payment.NewHandler(payment.NewService(payment.NewRepository(db)))
	walletHandler := wallet.NewHandler(wallet.NewService(wallet.NewRepository(db)))

	router := httpapi.NewRouter(httpapi.Handlers{
		User: userHandler, Hotel: hotelHandler, Room: roomHandler,
		Reservation: reservationHandler, Payment: paymentHandler, Wallet: walletHandler,
	})

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() { serverErr <- server.ListenAndServe() }()
	log.Printf("server listening on %s", server.Addr)

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-signalCtx.Done():
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	return server.Shutdown(shutdownCtx)
}
