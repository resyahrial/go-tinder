package rest

import (
	"database/sql"
	"gotinder/infra"
	"net/http"

	sq "github.com/Masterminds/squirrel"
	"github.com/gin-gonic/gin"
	"github.com/go-pkgz/auth/token"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type (
	passRequest struct {
		ID string `json:"id" validate:"required,uuid"`
	}
)

// RegisterPass register pass handler
func (v v1) Registerpass() {
	authMiddleware := v.auth.service.Middleware()

	locationGroup := v.group.Group("/passes", asGin(authMiddleware.Auth))
	locationGroup.POST("", pass)
}

func pass(ctx *gin.Context) {
	var req passRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	user := token.MustGetUserInfo(ctx.Request)

	findUserQuery, _, err := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Select("id").
		From("users").
		Where("email = $1").
		ToSql()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": errors.Wrap(err, "failed to build find user query").Error(),
		})
		return
	}

	row := infra.PgConn.QueryRow(findUserQuery, user.Name)
	var userId uuid.UUID
	if err := row.Scan(&userId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": "user not found",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": errors.Wrap(err, "failed to find user").Error(),
		})
		return
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	query, _, err := psql.
		Insert("passes").
		Columns("self_id", "target_id").
		Values("$1", "$2").
		Suffix("ON CONFLICT (self_id,target_id) DO NOTHING").
		ToSql()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": errors.Wrap(err, "failed to build create passes query").Error(),
		})
		return
	}

	if _, err := infra.PgConn.Exec(query, userId, req.ID); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": errors.Wrap(err, "failed to record request").Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "success pass user",
	})
}
