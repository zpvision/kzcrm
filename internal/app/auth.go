package app

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type contextKey string

const userKey contextKey = "user"

type authUser struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Name  string    `json:"name"`
	Role  string    `json:"role"`
	CSRF  string    `json:"csrfToken"`
}
type attempt struct {
	count int
	since time.Time
}
type loginLimiter struct {
	mu    sync.Mutex
	items map[string]attempt
}

func newLoginLimiter() *loginLimiter { return &loginLimiter{items: map[string]attempt{}} }
func (l *loginLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	a := l.items[key]
	if time.Since(a.since) > 15*time.Minute {
		a = attempt{since: time.Now()}
	}
	if a.count >= 8 {
		return false
	}
	a.count++
	l.items[key] = a
	return true
}
func (l *loginLimiter) clear(key string) { l.mu.Lock(); delete(l.items, key); l.mu.Unlock() }
func randomToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
func (s *Server) sessionHash(token string) []byte {
	h := hmac.New(sha256.New, []byte(s.cfg.SessionSecret))
	_, _ = h.Write([]byte(token))
	return h.Sum(nil)
}
func remoteIP(r *http.Request) string {
	h, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return h
	}
	return r.RemoteAddr
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	ip := remoteIP(r)
	if !s.attempts.allow(ip) {
		problem(w, 429, "Слишком много попыток. Повторите через 15 минут")
		return
	}
	var in struct{ Email, Password string }
	if !decode(w, r, &in) {
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	var u authUser
	var hash string
	var active bool
	err := s.db.QueryRow(r.Context(), `SELECT id,email,display_name,role,password_hash,active FROM users WHERE email=$1`, in.Email).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &hash, &active)
	if err != nil || !active || !verifyPassword(in.Password, hash) {
		time.Sleep(250 * time.Millisecond)
		problem(w, 401, "Неверный email или пароль")
		return
	}
	token := randomToken(32)
	csrf := randomToken(24)
	sum := s.sessionHash(token)
	expires := time.Now().Add(12 * time.Hour)
	_, err = s.db.Exec(r.Context(), `INSERT INTO sessions(user_id,token_hash,csrf_token,expires_at,ip_address,user_agent) VALUES($1,$2,$3,$4,$5,$6)`, u.ID, sum, csrf, expires, ip, r.UserAgent())
	if err != nil {
		dbProblem(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "kzcrm_session", Value: token, Path: "/", HttpOnly: true, Secure: s.cfg.Env == "production", SameSite: http.SameSiteStrictMode, Expires: expires, MaxAge: 43200})
	u.CSRF = csrf
	s.attempts.clear(ip)
	respond(w, 200, u)
}

func (s *Server) authenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("kzcrm_session")
		if err != nil {
			problem(w, 401, "Требуется вход")
			return
		}
		sum := s.sessionHash(c.Value)
		var u authUser
		err = s.db.QueryRow(r.Context(), `SELECT u.id,u.email,u.display_name,u.role,s.csrf_token FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=$1 AND s.expires_at>now() AND u.active`, sum).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.CSRF)
		if err != nil {
			problem(w, 401, "Сессия истекла")
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" && r.Header.Get("X-CSRF-Token") != u.CSRF {
			problem(w, 403, "Неверный CSRF-токен")
			return
		}
		ctx := context.WithValue(r.Context(), userKey, u)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
func currentUser(r *http.Request) authUser { return r.Context().Value(userKey).(authUser) }

func (s *Server) me(w http.ResponseWriter, r *http.Request) { respond(w, 200, currentUser(r)) }
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, e := r.Cookie("kzcrm_session"); e == nil {
		sum := s.sessionHash(c.Value)
		_, _ = s.db.Exec(r.Context(), `DELETE FROM sessions WHERE token_hash=$1`, sum)
	}
	http.SetCookie(w, &http.Cookie{Name: "kzcrm_session", Value: "", Path: "/", HttpOnly: true, MaxAge: -1, SameSite: http.SameSiteStrictMode})
	w.WriteHeader(204)
}
