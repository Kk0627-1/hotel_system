package httpapi

import (
	"log"
	"net/http"

	"hotelsystem/internal/web"
)

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		web.SetCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic while handling request: %v", recovered)
				web.WriteJSON(w, http.StatusInternalServerError, web.SimpleResponse{Success: false, Message: "Internal server error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
