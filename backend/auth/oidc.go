package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

type OIDC struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	CookieSecret []byte
	Logger       *zap.Logger

	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
	oauth2Config oauth2.Config
}

func (o *OIDC) Init(ctx context.Context) error {
	provider, err := oidc.NewProvider(ctx, o.IssuerURL)
	if err != nil {
		return fmt.Errorf("oidc provider discovery: %w", err)
	}
	o.provider = provider
	o.verifier = provider.Verifier(&oidc.Config{ClientID: o.ClientID})
	o.oauth2Config = oauth2.Config{
		ClientID:     o.ClientID,
		ClientSecret: o.ClientSecret,
		RedirectURL:  o.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "groups"},
	}
	return nil
}

const (
	stateCookie    = "navidrome_oidc_state"
	verifierCookie = "navidrome_oidc_verifier"
	sessionCookie  = "navidrome_session"
	sessionTTL     = 12 * time.Hour
)

func (o *OIDC) Login(w http.ResponseWriter, r *http.Request) {
	state, err := randBase64(16)
	if err != nil {
		http.Error(w, "state", http.StatusInternalServerError)
		return
	}
	verifier, err := randBase64(32)
	if err != nil {
		http.Error(w, "verifier", http.StatusInternalServerError)
		return
	}

	setCookie(w, stateCookie, state, 10*time.Minute)
	setCookie(w, verifierCookie, verifier, 10*time.Minute)

	authURL := o.oauth2Config.AuthCodeURL(
		state,
		oauth2.AccessTypeOnline,
		oauth2.SetAuthURLParam("code_challenge", pkceChallenge(verifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (o *OIDC) Callback(w http.ResponseWriter, r *http.Request) {
	stateCk, err := r.Cookie(stateCookie)
	if err != nil {
		http.Error(w, "state cookie missing", http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("state") != stateCk.Value {
		http.Error(w, "state mismatch", http.StatusBadRequest)
		return
	}
	verifierCk, err := r.Cookie(verifierCookie)
	if err != nil {
		http.Error(w, "verifier cookie missing", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	token, err := o.oauth2Config.Exchange(
		ctx, r.URL.Query().Get("code"),
		oauth2.SetAuthURLParam("code_verifier", verifierCk.Value),
	)
	if err != nil {
		o.Logger.Error("oauth2 exchange", zap.Error(err))
		http.Error(w, "exchange failed", http.StatusBadGateway)
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "id_token missing", http.StatusBadGateway)
		return
	}
	idToken, err := o.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		o.Logger.Error("id token verify", zap.Error(err))
		http.Error(w, "id token invalid", http.StatusUnauthorized)
		return
	}

	var claims struct {
		Sub               string `json:"sub"`
		Email             string `json:"email"`
		PreferredUsername string `json:"preferred_username"`
	}
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, "claims", http.StatusBadGateway)
		return
	}

	username := claims.PreferredUsername
	if username == "" {
		userInfo, err := o.provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
		if err != nil {
			o.Logger.Error("userinfo", zap.Error(err))
			http.Error(w, "userinfo failed", http.StatusBadGateway)
			return
		}
		var ui struct {
			PreferredUsername string `json:"preferred_username"`
		}
		if err := userInfo.Claims(&ui); err != nil {
			http.Error(w, "userinfo claims", http.StatusBadGateway)
			return
		}
		username = ui.PreferredUsername
	}
	if username == "" {
		username = claims.Email
	}
	if username == "" {
		o.Logger.Error("no username claim in token")
		http.Error(w, "no username", http.StatusBadGateway)
		return
	}

	sess := session{User: username, Exp: time.Now().Add(sessionTTL).Unix()}
	cookieVal, err := o.encodeSession(sess)
	if err != nil {
		http.Error(w, "encode session", http.StatusInternalServerError)
		return
	}

	clearCookie(w, stateCookie)
	clearCookie(w, verifierCookie)
	setCookie(w, sessionCookie, cookieVal, sessionTTL)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (o *OIDC) Logout(w http.ResponseWriter, r *http.Request) {
	clearCookie(w, sessionCookie)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (o *OIDC) SessionUser(r *http.Request) (string, bool) {
	ck, err := r.Cookie(sessionCookie)
	if err != nil {
		return "", false
	}
	return o.validSession(ck.Value)
}

type session struct {
	User string `json:"user"`
	Exp  int64  `json:"exp"`
}

func (o *OIDC) encodeSession(s session) (string, error) {
	body, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, o.CookieSecret)
	mac.Write(body)
	sig := mac.Sum(nil)
	return base64.URLEncoding.EncodeToString(body) + "." + base64.URLEncoding.EncodeToString(sig), nil
}

func (o *OIDC) validSession(raw string) (string, bool) {
	parts := strings.SplitN(raw, ".", 2)
	if len(parts) != 2 {
		return "", false
	}
	body, err := base64.URLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", false
	}
	sig, err := base64.URLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", false
	}
	mac := hmac.New(sha256.New, o.CookieSecret)
	mac.Write(body)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return "", false
	}
	var s session
	if err := json.Unmarshal(body, &s); err != nil {
		return "", false
	}
	if time.Now().Unix() >= s.Exp {
		return "", false
	}
	return s.User, true
}

func setCookie(w http.ResponseWriter, name, value string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()),
	})
}

func clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
	})
}

func randBase64(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func pkceChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
