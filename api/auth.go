package api

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/sessions"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Password string `json:"password"`
}

var Store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))

func init() {
	Store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   24 * 60 * 60, 
		HttpOnly: true, 
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	}
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (h *PhotoHandlers) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.Log.Error("unsupported HTTP method for login", zap.String("method", r.Method))
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Log.Error("failed to decode login request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	pwHash := os.Getenv("PW")
	if !CheckPasswordHash(req.Password, pwHash) {
		h.Log.Warn("invalid login credentials")
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	session, _ := Store.Get(r, "session-name")
	session.Values["authenticated"] = true
	session.Values["userId"] = "user123"
	session.Values["createdAt"] = time.Now().Unix()

	if err := session.Save(r, w); err != nil {
		h.Log.Error("failed to save session", zap.Error(err))
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	h.Log.Info("login successful")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Login successful"})
}

func AuthMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := Store.Get(r, "session-name")
			if err != nil {
				logger.Warn("failed to get session", zap.Error(err), zap.String("path", r.URL.Path))
				http.Error(w, "Invalid session", http.StatusUnauthorized)
				return
			}

			if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
				logger.Warn("session not authenticated", zap.String("path", r.URL.Path))
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			createdAt, ok := session.Values["createdAt"].(int64)
			if !ok || time.Now().Unix()-createdAt > 24*60*60 {
				logger.Warn("session expired", zap.String("path", r.URL.Path))
				http.Error(w, "Session expired", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}