package installer

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path"

	cp "github.com/otiai10/copy"
	"github.com/syncloud/golib/config"
	"github.com/syncloud/golib/linux"
	"github.com/syncloud/golib/platform"
	"go.uber.org/zap"
)

type Variables struct {
	App              string
	AppDir           string
	DataDir          string
	CommonDir        string
	StorageDir       string
	AppDomain        string
	AppUrl           string
	AuthUrl          string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURI  string
	Socket           string
}

const (
	App         = "navidrome"
	AppDir      = "/snap/navidrome/current"
	DataDir     = "/var/snap/navidrome/current"
	CommonDir   = "/var/snap/navidrome/common"
	OIDCCallback = "/syncloud-oidc/callback"
)

type Installer struct {
	newVersionFile     string
	currentVersionFile string
	platformClient     *platform.Client
	installFile        string
	secretFile         string
	logger             *zap.Logger
}

func New(logger *zap.Logger) *Installer {
	return &Installer{
		newVersionFile:     path.Join(AppDir, "version"),
		currentVersionFile: path.Join(DataDir, "version"),
		platformClient:     platform.New(),
		installFile:        path.Join(CommonDir, "installed"),
		secretFile:         path.Join(DataDir, ".secret"),
		logger:             logger,
	}
}

func (i *Installer) Install() error {
	return i.UpdateConfigs()
}

func (i *Installer) Configure() error {
	if i.IsInstalled() {
		return i.Upgrade()
	}
	return i.Initialize()
}

func (i *Installer) IsInstalled() bool {
	_, err := os.Stat(i.installFile)
	return err == nil
}

func (i *Installer) Initialize() error {
	if err := i.StorageChange(); err != nil {
		return err
	}
	if err := os.WriteFile(i.installFile, []byte("installed"), 0644); err != nil {
		return err
	}
	return i.UpdateVersion()
}

func (i *Installer) Upgrade() error {
	if err := i.StorageChange(); err != nil {
		return err
	}
	return i.UpdateVersion()
}

func (i *Installer) PreRefresh() error {
	return nil
}

func (i *Installer) PostRefresh() error {
	if err := i.UpdateConfigs(); err != nil {
		return err
	}
	if err := i.ClearVersion(); err != nil {
		return err
	}
	return i.FixPermissions()
}

func (i *Installer) AccessChange() error {
	return i.UpdateConfigs()
}

func (i *Installer) StorageChange() error {
	storageDir, err := i.platformClient.InitStorage(App, App)
	if err != nil {
		return err
	}
	return linux.Chown(storageDir, App)
}

func (i *Installer) ClearVersion() error {
	return os.RemoveAll(i.currentVersionFile)
}

func (i *Installer) UpdateVersion() error {
	return cp.Copy(i.newVersionFile, i.currentVersionFile)
}

func (i *Installer) UpdateConfigs() error {
	if err := linux.CreateUser(App); err != nil {
		return err
	}
	if err := i.StorageChange(); err != nil {
		return err
	}
	if err := linux.CreateMissingDirs(
		path.Join(DataDir, "config"),
		path.Join(DataDir, "nginx"),
		path.Join(DataDir, "data"),
		path.Join(DataDir, "cache"),
	); err != nil {
		return err
	}

	storageDir, err := i.platformClient.InitStorage(App, App)
	if err != nil {
		return err
	}
	if err := i.GenerateConfig(storageDir); err != nil {
		return fmt.Errorf("generate config: %w", err)
	}

	return i.FixPermissions()
}

func (i *Installer) GenerateConfig(storageDir string) error {
	oidcSecret, err := i.platformClient.RegisterOIDCClient(App, OIDCCallback, true, "client_secret_basic")
	if err != nil {
		return fmt.Errorf("register oidc client: %w", err)
	}
	authUrl, err := i.platformClient.GetAppUrl("auth")
	if err != nil {
		return err
	}
	appUrl, err := i.platformClient.GetAppUrl(App)
	if err != nil {
		return err
	}
	appDomain, err := i.platformClient.GetAppDomainName(App)
	if err != nil {
		return err
	}
	if _, err := i.cookieSecret(); err != nil {
		return err
	}

	variables := Variables{
		App:              App,
		AppDir:           AppDir,
		DataDir:          DataDir,
		CommonDir:        CommonDir,
		StorageDir:       storageDir,
		AppDomain:        appDomain,
		AppUrl:           appUrl,
		AuthUrl:          authUrl,
		OIDCClientID:     App,
		OIDCClientSecret: oidcSecret,
		OIDCRedirectURI:  trimRightSlash(appUrl) + OIDCCallback,
		Socket:           path.Join(DataDir, "backend.sock"),
	}

	return config.Generate(
		path.Join(AppDir, "config"),
		path.Join(DataDir, "config"),
		variables,
	)
}

func (i *Installer) cookieSecret() (string, error) {
	existing, err := os.ReadFile(i.secretFile)
	if err == nil && len(existing) > 0 {
		return string(existing), nil
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	secret := hex.EncodeToString(buf)
	if err := os.WriteFile(i.secretFile, []byte(secret), 0640); err != nil {
		return "", err
	}
	return secret, nil
}

func (i *Installer) FixPermissions() error {
	if err := linux.Chown(DataDir, App); err != nil {
		return err
	}
	return linux.Chown(CommonDir, App)
}

func (i *Installer) BackupPreStop() error    { return i.PreRefresh() }
func (i *Installer) RestorePreStart() error  { return i.PostRefresh() }
func (i *Installer) RestorePostStart() error { return i.Configure() }

func trimRightSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
