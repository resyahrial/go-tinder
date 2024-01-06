package rest

import (
	"database/sql"
	"fmt"
	"gotinder/infra"
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

	recommendations := fetchRecommendation(ctx, u.ID.String(), u.Lat, u.Lng, param.Limit)
	if recommendations == nil {
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": recommendations,
	})
}

func fetchRecommendation(ctx *gin.Context, userID, lat, lng string, limit int) []recommendationResponse {
	const maxDistanceInMeter = 150000
	findRecommendationsQuery, _, err := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Select(
			"users.id",
			"users.birth_of_date",
			fmt.Sprintf(
				"st_distancesphere(latest_locations.location::geometry,ST_SetSRID(ST_MakePoint(%s,%s), 4326)) AS distance",
				lat,
				lng,
			),
		).
		From("users").
		LeftJoin("passes ON passes.target_id = users.id").
		LeftJoin("likes ON likes.target_id = users.id").
		InnerJoin("latest_locations ON users.id = latest_locations.user_id").
		Where("users.id != $1").
		Where("passes.self_id IS NULL").
		Where("likes.self_id IS NULL").
		Where(fmt.Sprintf(
			"ST_DWithin(latest_locations.location, ST_SetSRID(ST_MakePoint($2,$3), 4326)::geography, %v)",
			maxDistanceInMeter,
		)).
		OrderBy("latest_locations.updated_at DESC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": errors.Wrap(err, "failed to build find recommendations query").Error(),
		})
		return nil
	}

	rows, err := infra.PgConn.Query(findRecommendationsQuery, userID, lat, lng)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return nil
	}
	defer rows.Close()
	if err := rows.Err(); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return nil
	}

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
			return nil
		}
		recommendations = append(recommendations, recommendation)
	}
	return recommendations
}
