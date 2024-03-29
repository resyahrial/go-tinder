package rest

import (
	"fmt"
	"gotinder/infra"
	"net/http"

	sq "github.com/Masterminds/squirrel"
	"github.com/gin-gonic/gin"
	"github.com/go-pkgz/auth/token"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
)

type (
	// actionRequest is a type of action (like/pass) request body
	actionRequest struct {
		ID string `json:"id" validate:"required,uuid"`
	}

	// actionType is a type of available actions
	actionType string
)

const (
	actionLike actionType = "likes"
	actionPass actionType = "passes"
)

// RegisterAction register like handler
func (v v1) RegisterAction() {
	authMiddleware := v.auth.service.Middleware()

	locationGroup := v.group.Group("/actions", asGin(authMiddleware.Auth), enrichActor)
	locationGroup.POST("/likes", like)
	locationGroup.POST("/passes", pass)
}

// like will record that the actor is liking the target
func like(ctx *gin.Context) {
	if !action(ctx, actionLike) {
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "success like user",
	})
}

// pass will record that the actor is passing the target
func pass(ctx *gin.Context) {
	if !action(ctx, actionPass) {
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "success pass user",
	})
}

// action is a common functionality of like and pass
func action(ctx *gin.Context, actType actionType) bool {
	var req actionRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return false
	}

	user := token.MustGetUserInfo(ctx.Request)
	self := user.StrAttr("user_id")

	actionKey := fmt.Sprintf("action-%s", self)
	if !user.IsPaidSub() && !isActionAllowed(ctx, actionKey) {
		return false
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	query, _, err := psql.
		Insert(string(actType)).
		Columns("self_id", "target_id").
		Values("$1", "$2").
		Suffix("ON CONFLICT (self_id,target_id) DO NOTHING").
		ToSql()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": errors.Wrap(err, fmt.Sprintf("failed to build create %s query", string(actType))).Error(),
		})
		return false
	}

	if _, err := infra.PgConn.Exec(query, self, req.ID); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": errors.Wrap(err, "failed to record request").Error(),
		})
		return false
	}

	if !cacheAction(ctx, actionKey, req.ID) {
		return false
	}

	return true
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

	actionCount, err := redis.Int(cacheConn.Do("SCARD", actionKey))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return false
	}

	const maxActionAllowed int = 10
	if actionCount >= maxActionAllowed {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "exceed max action allowed",
		})
		return false
	}

	return true
}

// cacheAction will record current action
func cacheAction(ctx *gin.Context, actionKey string, target string) bool {
	cacheConn := infra.RedisPool.Get()
	defer cacheConn.Close()

	if _, err := cacheConn.Do("SADD", actionKey, target); err != nil {
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
