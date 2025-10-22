package api

import (
	"net/http"

	"go.uber.org/zap"
)

// Recovery middleware
func RecoveryMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("panic recovered", zap.Any("error", err))
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// Request logger middleware
func RequestLoggerMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Info("request",
				zap.String("method", r.Method),
				zap.String("url", r.URL.String()),
				zap.String("remote_addr", r.RemoteAddr),
			)
			next.ServeHTTP(w, r)
		})
	}
}

// CORS middleware
func CORSMiddleware() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
            w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
            w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
            w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
            w.Header().Set("Access-Control-Allow-Credentials", "true")

            if req.Method == "OPTIONS" {
                w.WriteHeader(http.StatusNoContent)
                return
            }

            next.ServeHTTP(w, req)
        })
    }
}