package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/go-ldap/ldap/v3"
	"go.uber.org/zap"
)

type LDAP struct {
	URL    string
	Logger *zap.Logger

	cacheTTL time.Duration
	mu       sync.Mutex
	cache    map[string]time.Time
}

func NewLDAP(url string, logger *zap.Logger) *LDAP {
	return &LDAP{
		URL:      url,
		Logger:   logger,
		cacheTTL: 5 * time.Minute,
		cache:    map[string]time.Time{},
	}
}

func (l *LDAP) Authenticate(username, password string) bool {
	if !validUsername(username) || password == "" {
		return false
	}
	key := cacheKey(username, password)
	if l.cached(key) {
		return true
	}

	conn, err := ldap.DialURL(l.URL)
	if err != nil {
		l.Logger.Error("ldap dial", zap.Error(err))
		return false
	}
	defer conn.Close()

	dn := fmt.Sprintf("cn=%s,ou=users,dc=syncloud,dc=org", username)
	if err := conn.Bind(dn, password); err != nil {
		l.Logger.Warn("ldap bind failed", zap.String("user", username), zap.Error(err))
		return false
	}

	l.store(key)
	return true
}

func (l *LDAP) cached(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	exp, ok := l.cache[key]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(l.cache, key)
		return false
	}
	return true
}

func (l *LDAP) store(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cache[key] = time.Now().Add(l.cacheTTL)
}

func validUsername(username string) bool {
	if username == "" || len(username) > 64 {
		return false
	}
	for _, r := range username {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '.' || r == '_' || r == '-' || r == '@':
		default:
			return false
		}
	}
	return true
}

func cacheKey(username, password string) string {
	h := sha256.Sum256([]byte(username + "\x00" + password))
	return username + ":" + hex.EncodeToString(h[:])
}
