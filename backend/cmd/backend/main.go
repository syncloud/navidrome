package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"backend/auth"
	"backend/config"
)

const (
	app           = "navidrome"
	dataDir       = "/var/snap/" + app + "/current"
	backendSock   = dataDir + "/backend.sock"
	navidromeSock = dataDir + "/navidrome.sock"
	secretPath    = dataDir + "/.secret"
	ldapURL       = "ldap://localhost:389"
	userHeader    = "Remote-User"
)

type ctxKey int

const userKey ctxKey = 0

func withUser(r *http.Request, user string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), userKey, user))
}

func main() {
	cmd := &cobra.Command{
		Use:          "backend",
		Short:        "Navidrome Syncloud auth gateway — OIDC web SSO + LDAP Subsonic auth in front of navidrome",
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			logger, err := buildLogger()
			if err != nil {
				return err
			}
			return run(logger)
		},
	}
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(logger *zap.Logger) error {
	cfg := &config.Config{DataDir: dataDir}
	if err := cfg.Load(); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	cookieSecret, err := os.ReadFile(secretPath)
	if err != nil {
		return fmt.Errorf("read cookie secret: %w", err)
	}

	oidc := &auth.OIDC{
		IssuerURL:    cfg.AuthBaseURL,
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURI,
		CookieSecret: cookieSecret,
		Logger:       logger,
	}
	if err := oidc.Init(context.Background()); err != nil {
		return fmt.Errorf("oidc init: %w", err)
	}

	ldap := auth.NewLDAP(ldapURL, logger)
	proxy := newNavidromeProxy(navidromeSock, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /syncloud-oidc/login", oidc.Login)
	mux.HandleFunc("GET /syncloud-oidc/callback", oidc.Callback)
	mux.HandleFunc("GET /syncloud-oidc/logout", oidc.Logout)
	mux.Handle("/rest/", subsonicHandler(ldap, proxy))
	mux.Handle("/", webHandler(oidc, proxy))

	_ = os.Remove(backendSock)
	listener, err := net.Listen("unix", backendSock)
	if err != nil {
		return fmt.Errorf("listen %s: %w", backendSock, err)
	}
	if err := os.Chmod(backendSock, 0666); err != nil {
		return fmt.Errorf("chmod socket: %w", err)
	}

	logger.Info("backend listening", zap.String("socket", backendSock))
	return (&http.Server{Handler: mux}).Serve(listener)
}

func buildLogger() (*zap.Logger, error) {
	c := zap.NewProductionConfig()
	c.Encoding = "console"
	c.EncoderConfig.TimeKey = ""
	c.OutputPaths = []string{"stdout"}
	c.ErrorOutputPaths = []string{"stderr"}
	return c.Build()
}
