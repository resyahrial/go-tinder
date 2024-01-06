package rest

import (
	"database/sql"
	"gotinder/infra"
	"net/http"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gin-gonic/gin"
	"github.com/go-pkgz/auth/token"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type (
	// registerRequest is a type of "/register" request body
	locationRequest struct {
		Lat string `json:"lat" validate:"required,latitude"`
		Lng string `json:"lng" validate:"required,longitude"`
	}
)

// RegisterLocation register location handler
func (v v1) RegisterLocation() {
	authMiddleware := v.auth.service.Middleware()

	locationGroup := v.group.Group("/locations", asGin(authMiddleware.Auth))
	locationGroup.POST("", updateLocation)
}

// updateLocation do process to update user current location
func updateLocation(ctx *gin.Context) {
	var req locationRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	user := token.MustGetUserInfo(ctx.Request)

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	findUserQuery, _, err := psql.Select("id").From("users").Where("email = $1").ToSql()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": errors.Wrap(err, "failed to build find user query").Error(),
		})
		return
	}

	row := infra.PgConn.QueryRow(findUserQuery, user.Name)
	var userID uuid.UUID
	if err := row.Scan(&userID); err != nil {
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

	upsertLatestLocation, _, err := psql.
		Insert("latest_locations").
		Columns("updated_at", "lat", "lng", "user_id").
		Values("$1", "$2", "$3", "$4").
		Suffix(`
			ON CONFLICT (user_id) DO UPDATE SET 
			updated_at=EXCLUDED.updated_at, 
			lat=EXCLUDED.lat, 
			lng=EXCLUDED.lng, 
			location=EXCLUDED.location
		`).
		ToSql()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": errors.Wrap(err, "failed to build upsert latest location query").Error(),
		})
		return
	}

	if _, err := infra.PgConn.Exec(upsertLatestLocation, time.Now().Unix(), req.Lat, req.Lng, userID); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": errors.Wrap(err, "failed to record request").Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "success update location",
	})
}
