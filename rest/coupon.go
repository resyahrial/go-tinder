package rest

import (
	"database/sql"
	"errors"
	"gotinder/infra"
	"net/http"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gin-gonic/gin"
)

type (
	// couponRequest is a type of "/coupons" request body
	couponRequest struct {
		Code             string `json:"code" validate:"required,alphanum,min=5"`
		DurationInSecond int64  `json:"duration_in_second" validate:"required,gte=0"`
		ValidUntil       int64  `json:"valid_until" validate:"required,gte=0"`
	}

	// applyCouponRequest is a type of "/coupons/apply" request body
	applyCouponRequest struct {
		Code   string `json:"code" validate:"required"`
		UserID string `json:"user_id" validate:"required,uuid"`
	}
)

// CouponLocation register location handler
func (v v1) CouponLocation() {
	authMiddleware := v.auth.service.Middleware()

	locationGroup := v.group.Group("/coupons", asGin(authMiddleware.Auth))
	locationGroup.POST("", createCoupon)
	locationGroup.POST("/apply", applyCoupon)
}

// createCoupon creating coupon
func createCoupon(ctx *gin.Context) {
	var req couponRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	_, err := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Insert("coupons").
		Columns("code", "duration_in_second", "valid_until").
		Values(req.Code, req.DurationInSecond, req.ValidUntil).
		RunWith(infra.PgConn).
		Exec()
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "success create coupon",
	})
}

// applyCoupon applying coupon to user
func applyCoupon(ctx *gin.Context) {
	var req applyCouponRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	row := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Select("id", "valid_until").
		From("coupons").
		Where("code = ?", req.Code).
		RunWith(infra.PgConn).
		QueryRow()
	var coupon struct {
		ID         string
		ValidUntil int64
	}
	if err := row.Scan(&coupon.ID, &coupon.ValidUntil); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": "coupon not found",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	if time.Now().After(time.Unix(coupon.ValidUntil, 0)) {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "coupon expired",
		})
		return
	}

	_, err := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Insert("user_coupons").
		Columns("user_id", "coupon_id").
		Values(req.UserID, coupon.ID).
		RunWith(infra.PgConn).
		Exec()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "success apply coupon",
	})
}
