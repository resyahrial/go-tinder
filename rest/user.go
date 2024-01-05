package rest

import (
	"gotinder/infra"
	"net/http"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gin-gonic/gin"
	"github.com/go-pkgz/auth/token"
)

// RegisterUser register user handler
func (v v1) RegisterUser() {
	authMiddleware := v.auth.service.Middleware()

	locationGroup := v.group.Group("/users", asGin(authMiddleware.Auth), enrichActor)
	locationGroup.PATCH("/subscribe", subscribe)
}

// subscribe do process user subscribption
func subscribe(ctx *gin.Context) {
	user := token.MustGetUserInfo(ctx.Request)

	_, err := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Update("users").
		Set("subscribe_until", time.Now().Add(24*365*time.Hour).Unix()).
		Where("id = ?", user.StrAttr("user_id")).
		RunWith(infra.PgConn).
		Exec()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "success record subscription",
	})
}
