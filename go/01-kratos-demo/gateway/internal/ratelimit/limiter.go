package ratelimit

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gin-demo/gateway/internal/httpx"

	"golang.org/x/time/rate"
)

type Limiter struct {
	rps     rate.Limit
	burst   int
	ttl     time.Duration
	clients sync.Map
}

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen atomic.Int64
}

func NewPerIP(rps float64, burst int) *Limiter {
	if rps <= 0 {
		rps = 1
	}
	if burst <= 0 {
		burst = 1
	}

	limiter := &Limiter{
		rps:   rate.Limit(rps),
		burst: burst,
		ttl:   3 * time.Minute,
	}
	go limiter.cleanup()
	return limiter
}

func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(r) {
			httpx.WriteError(w, http.StatusTooManyRequests, "too many requests")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *Limiter) Handler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(r) {
			httpx.WriteError(w, http.StatusTooManyRequests, "too many requests")
			return
		}
		next(w, r)
	}
}

func (l *Limiter) allow(r *http.Request) bool {
	key := clientIP(r)
	value, ok := l.clients.Load(key)
	if !ok {
		client := &clientLimiter{limiter: rate.NewLimiter(l.rps, l.burst)}
		client.lastSeen.Store(time.Now().UnixNano())
		value, _ = l.clients.LoadOrStore(key, client)
	}

	client := value.(*clientLimiter)
	client.lastSeen.Store(time.Now().UnixNano())
	return client.limiter.Allow()
}

func (l *Limiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		l.clients.Range(func(key, value interface{}) bool {
			client := value.(*clientLimiter)
			lastSeen := time.Unix(0, client.lastSeen.Load())
			if now.Sub(lastSeen) > l.ttl {
				l.clients.Delete(key)
			}
			return true
		})
	}
}

func clientIP(r *http.Request) string {
	if value := r.Header.Get("X-Forwarded-For"); value != "" {
		parts := strings.Split(value, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	if value := r.Header.Get("X-Real-IP"); value != "" {
		return strings.TrimSpace(value)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}
