package main

import (
	"encoding/hex"
	"net/http"
	"strings"

	"backend/auth"
)

func subsonicHandler(ldap *auth.LDAP, proxy http.Handler, ensurer *userEnsurer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := subsonicCredentials(r)
		if ok && ldap.Authenticate(user, pass) {
			ensurer.ensure(user)
			proxy.ServeHTTP(w, withUser(r, user))
			return
		}
		proxy.ServeHTTP(w, r)
	})
}

func subsonicCredentials(r *http.Request) (string, string, bool) {
	if user, pass, ok := r.BasicAuth(); ok && pass != "" {
		return user, pass, true
	}

	q := r.URL.Query()
	user := q.Get("u")
	if user == "" {
		return "", "", false
	}
	pass := q.Get("p")
	if pass == "" {
		return "", "", false
	}
	if decoded, ok := strings.CutPrefix(pass, "enc:"); ok {
		b, err := hex.DecodeString(decoded)
		if err != nil {
			return "", "", false
		}
		pass = string(b)
	}
	return user, pass, true
}
