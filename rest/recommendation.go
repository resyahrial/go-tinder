package rest

import (
	"database/sql"
	"fmt"
	"gotinder/infra"
	"log"
	"net/http"

	sq "github.com/Masterminds/squirrel"
	"github.com/gin-gonic/gin"
	"github.com/go-pkgz/auth/token"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type (
	findRecommendationsQueryParam struct {
		Limit int `form:"limit" validate:"required,gte=1"`
	}

	recommendationResponse struct {
		ID          uuid.UUID `json:"id"`
		BirthOfDate int64     `json:"birth_of_date"`
		Distance    string    `json:"distance_in_meter"`
	}
)

// RegisterRecommendation register recommendation handler
func (v v1) RegisterRecommendation() {
	authMiddleware := v.auth.service.Middleware()

	locationGroup := v.group.Group("/recommendations", asGin(authMiddleware.Auth))
	locationGroup.GET("", findRecommendations)
}

// findRecommendations give list of user recommendation for current user
func findRecommendations(ctx *gin.Context) {
	var param findRecommendationsQueryParam
	if err := ctx.ShouldBind(&param); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	user := token.MustGetUserInfo(ctx.Request)

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	findUserQuery, _, err := psql.
		Select("users.id", "latest_locations.lat", "latest_locations.lng").
		From("users").
		InnerJoin("latest_locations ON users.id = latest_locations.user_id").
		Where("email = $1").
		ToSql()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": errors.Wrap(err, "failed to build find user query").Error(),
		})
		return
	}

	row := infra.PgConn.QueryRow(findUserQuery, user.Name)
	var u struct {
		ID  uuid.UUID
		Lat string
		Lng string
	}
	if err := row.Scan(&u.ID, &u.Lat, &u.Lng); err != nil {
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

	findRecommendationsQuery, _, err := psql.
		Select("users.id", "users.birth_of_date", fmt.Sprintf("st_distancesphere(latest_locations.location::geometry,ST_SetSRID(ST_MakePoint(%s,%s), 4326)) AS distance", u.Lat, u.Lng)).
		From("users").
		InnerJoin("latest_locations ON users.id = latest_locations.user_id").
		Where("users.id != $1").
		Where("ST_DWithin(latest_locations.location, ST_SetSRID(ST_MakePoint($2,$3), 4326)::geography, 150000)").
		OrderBy("latest_locations.updated_at DESC").
		Limit(uint64(param.Limit)).
		ToSql()
	log.Println(findRecommendationsQuery)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": errors.Wrap(err, "failed to build find recommendations query").Error(),
		})
		return
	}

	rows, err := infra.PgConn.Query(findRecommendationsQuery, u.ID, u.Lat, u.Lng)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	defer rows.Close()

	var recommendations []recommendationResponse
	for rows.Next() {
		var recommendation recommendationResponse
		if err := rows.Scan(
			&recommendation.ID,
			&recommendation.BirthOfDate,
			&recommendation.Distance,
		); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		recommendations = append(recommendations, recommendation)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": recommendations,
	})
}
