package rest

import (
	"database/sql"
	"gotinder/infra"
	"log"
	"net/http"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gin-gonic/gin"
	"github.com/go-pkgz/auth/token"
	"github.com/pkg/errors"
)

type (
	// subcribeRequest is a type for "/users/subscribe" request
	subcribeRequest struct {
		CouponCode string `json:"coupon_code" validate:"required"`
	}
)

// RegisterUser register user handler
func (v v1) RegisterUser() {
	authMiddleware := v.auth.service.Middleware()

	locationGroup := v.group.Group("/users", asGin(authMiddleware.Auth), enrichActor)
	locationGroup.POST("/subscribe", subscribe)
}

// subscribe do process user subscribption
func subscribe(ctx *gin.Context) {
	var req subcribeRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	user := token.MustGetUserInfo(ctx.Request)

	row := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Select("user_coupons.id", "coupons.duration_in_second", "users.subscribe_until").
		From("user_coupons").
		InnerJoin("coupons ON coupons.id = user_coupons.coupon_id").
		InnerJoin("users ON users.id = user_coupons.user_id").
		Where("user_coupons.user_id = ?", user.StrAttr("user_id")).
		Where("coupons.code = ?", req.CouponCode).
		Where("user_coupons.used_at IS NULL").
		RunWith(infra.PgConn).
		QueryRow()
	var userCoupon struct {
		Id               string
		DurationInSecond int64
		UserSubscribedAt sql.NullInt64
	}
	if err := row.Scan(&userCoupon.Id, &userCoupon.DurationInSecond, &userCoupon.UserSubscribedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": "coupon not found or already applied",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	tx, err := infra.PgConn.Begin()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	var isCommitted bool
	defer func() {
		if !isCommitted {
			if err := tx.Rollback(); err != nil {
				log.Println(err)
			}
		}
	}()

	subscribeUntil := time.Now()
	if subAt := userCoupon.UserSubscribedAt; subAt.Valid && subscribeUntil.Before(time.Unix(subAt.Int64, 0)) {
		subscribeUntil = time.Unix(subAt.Int64, 0)
	}
	subscribeUntil = subscribeUntil.Add(time.Duration(userCoupon.DurationInSecond) * time.Second)

	if _, err := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Update("users").
		Set("subscribe_until", subscribeUntil.Unix()).
		Where("id = ?", user.StrAttr("user_id")).
		RunWith(tx).
		Exec(); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	if _, err := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Update("user_coupons").
		Set("used_at", time.Now().Unix()).
		Where("id = ?", userCoupon.Id).
		RunWith(tx).
		Exec(); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := tx.Commit(); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	isCommitted = true

	ctx.JSON(http.StatusOK, gin.H{
		"message": "success record subscription",
	})
}
