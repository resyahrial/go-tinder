package rest

import (
	"net/http"
	"time"

	"github.com/go-pkgz/auth"
	"github.com/go-pkgz/auth/avatar"
	"github.com/go-pkgz/auth/token"
)

func newAuthService() *auth.Service {
	opt := auth.Opts{
		SecretReader: token.SecretFunc(func(aud string) (string, error) {
			return "secret_key", nil
		}),
		SecureCookies:  true,
		TokenDuration:  5 * time.Minute,
		CookieDuration: 24 * 14 * time.Hour,
		DisableXSRF:    false,
		DisableIAT:     false,
		Issuer:         "gotinder",
		Validator: token.ValidatorFunc(func(token string, claims token.Claims) bool {
			return claims.Issuer == "gotinder"
		}),
		AvatarStore: avatar.NewNoOp(),
	}

	service := auth.NewService(opt)
	service.AddDirectProviderWithUserIDFunc("direct", credChecker{}, func(user string, r *http.Request) string {
		return user
	})

	return service
}

type credChecker struct{}

func (c credChecker) Check(user, password string) (bool, error) {
	return true, nil
}
