package rest

import (
	"database/sql"
	"fmt"
	"gotinder/infra"
	"net/http"

	sq "github.com/Masterminds/squirrel"
	"github.com/gin-gonic/gin"
	"github.com/go-pkgz/auth/token"
	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type (
	actionRequest struct {
		ID string `json:"id" validate:"required,uuid"`
	}
)

// RegisterAction register like handler
func (v v1) RegisterAction() {
	authMiddleware := v.auth.service.Middleware()

	locationGroup := v.group.Group("/actions", asGin(authMiddleware.Auth))
	locationGroup.POST("/likes", like)
	locationGroup.POST("/passes", pass)
}

// like will record that the actor is liking the target
func like(ctx *gin.Context) {
	self, target, ok := getSelfAndTargetAction(ctx)
	if !ok {
		return
	}

	actionKey := fmt.Sprintf("action-%s", self.String())
	if isActionAllowed(ctx, actionKey) {
		return
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	query, _, err := psql.
		Insert("likes").
		Columns("self_id", "target_id").
		Values("$1", "$2").
		Suffix("ON CONFLICT (self_id,target_id) DO NOTHING").
		ToSql()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": errors.Wrap(err, "failed to build create likes query").Error(),
		})
		return
	}

	if _, err := infra.PgConn.Exec(query, self, target); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": errors.Wrap(err, "failed to record request").Error(),
		})
		return
	}

	if cacheAction(ctx, actionKey, target) {
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "success like user",
	})
}

// -as will record that the actor is passing the target
func pass(ctx *gin.Context) {
	self, target, ok := getSelfAndTargetAction(ctx)
	if !ok {
		return
	}

	actionKey := fmt.Sprintf("action-%s", self.String())
	if isActionAllowed(ctx, actionKey) {
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

	if _, err := infra.PgConn.Exec(query, self, target); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": errors.Wrap(err, "failed to record request").Error(),
		})
		return
	}

	if cacheAction(ctx, actionKey, target) {
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "success pass user",
	})
}

// getSelfAndTargetAction validate and fetch action's actor and its target
func getSelfAndTargetAction(ctx *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	var req actionRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return uuid.Nil, uuid.Nil, false
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
		return uuid.Nil, uuid.Nil, false
	}

	row := infra.PgConn.QueryRow(findUserQuery, user.Name)
	var self uuid.UUID
	if err := row.Scan(&self); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": "user not found",
			})
			return uuid.Nil, uuid.Nil, false
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": errors.Wrap(err, "failed to find user").Error(),
		})
		return uuid.Nil, uuid.Nil, false
	}

	return self, uuid.MustParse(req.ID), true
}

// isActionAllowed check if action's actor is allowed to do the action
func isActionAllowed(ctx *gin.Context, actionKey string) bool {
	cacheConn := infra.RedisPool.Get()
	defer cacheConn.Close()

	existsResponse, err := redis.Int(cacheConn.Do("EXISTS", actionKey))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return false
	}

	if existsResponse <= 0 {
		return true
	}

	const maxActionAllowed int = 10
	actionCount, err := redis.Int(cacheConn.Do("SCARD", actionKey))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return false
	}
	if actionCount >= maxActionAllowed {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "exceed max action allowed",
		})
		return false
	}

	return true
}

// cacheAction will record current action
func cacheAction(ctx *gin.Context, actionKey string, target uuid.UUID) bool {
	cacheConn := infra.RedisPool.Get()
	defer cacheConn.Close()

	if _, err := cacheConn.Do("SADD", actionKey, target.String()); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return false
	}

	const aDayInSecond = 60 * 60 * 24
	if _, err := cacheConn.Do("EXPIRE", actionKey, aDayInSecond, "NX"); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return false
	}

	return true
}
