package rest

import (
	"database/sql"
	"errors"
	"gotinder/infra"
	"net/http"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gin-gonic/gin"
	"github.com/go-pkgz/auth"
	"github.com/go-pkgz/auth/avatar"
	"github.com/go-pkgz/auth/provider"
	"github.com/go-pkgz/auth/token"
	passwordvalidator "github.com/wagslane/go-password-validator"
	"golang.org/x/crypto/bcrypt"
)

type (
	// registerRequest is a type of "/register" request body
	registerRequest struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required"`
	}
)

// RegisterAuth register auth handler
func (v v1) RegisterAuth() {
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
	service.AddDirectProviderWithUserIDFunc("direct", provider.CredCheckerFunc(checkCred), func(user string, r *http.Request) string {
		return user
	})

	authHandler, _ := service.Handlers()
	v.group.Match([]string{http.MethodGet, http.MethodPost}, "/auth/*provider", func(ctx *gin.Context) {
		provider := ctx.Param("provider")
		if provider == "/register" && ctx.Request.Method == http.MethodPost {
			register(ctx)
			return
		}
		authHandler.ServeHTTP(ctx.Writer, ctx.Request)
	})
}

// checkCred validate user's credential
func checkCred(email, password string) (bool, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	query, _, err := psql.Select("password").From("users").Where("email = $1").ToSql()
	if err != nil {
		return false, err
	}

	row := infra.PgConn.QueryRow(query, email)
	var recordedPassword string
	if err := row.Scan(&recordedPassword); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(recordedPassword), []byte(password)); err != nil {
		return false, err
	}

	return true, nil
}

func register(ctx *gin.Context) {
	var req registerRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := passwordvalidator.Validate(req.Password, 35); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "register success",
	})
}
