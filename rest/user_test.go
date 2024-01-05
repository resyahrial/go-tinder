package rest_test

import (
	"database/sql"
	"fmt"
	"gotinder/infra"
	"net/http"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/suite"
)

type UserTestSuite struct {
	suite.Suite
}

func TestUserTestSuite(t *testing.T) {
	suite.Run(t, new(UserTestSuite))
}

func (s *UserTestSuite) SetupSuite() {
	pg := newPostgresTest(s.T())
	infra.NewPgConnection(pg.connStr)
}

func (s *UserTestSuite) SetupTest() {
	pgTest.migrate(s.T(), infra.PgConn)
}

func (s *UserTestSuite) Test_Post_UserSubscription_Success() {
	tokens := getAuthToken(s.T(), infra.PgConn)

	rowFindUser := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Select("id").
		From("users").
		Where("email = ?", "base@mail.com").
		RunWith(infra.PgConn).
		QueryRow()
	var userId string
	s.Nil(rowFindUser.Scan(&userId))

	couponDuration := 60 * 60 * 24 * 365
	rowCreateCoupon := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Insert("coupons").
		Columns("code", "duration_in_second", "valid_until").
		Values("NEWUSER123", couponDuration, time.Now().Add(24*14*time.Hour).Unix()).
		Suffix("RETURNING id").
		RunWith(infra.PgConn).
		QueryRow()
	var couponId string
	s.Nil(rowCreateCoupon.Scan(&couponId))

	rowCreateUserCoupon := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Insert("user_coupons").
		Columns("user_id", "coupon_id").
		Values(userId, couponId).
		Suffix("RETURNING id").
		RunWith(infra.PgConn).
		QueryRow()
	var userCouponId string
	s.Nil(rowCreateUserCoupon.Scan(&userCouponId))

	res := newHttpTest().
		withPath("/v1/users/subscribe").
		withMethod(http.MethodPost).
		withBody(map[string]interface{}{
			"coupon_code": "NEWUSER123",
		}).
		withHeader("Cookie", fmt.Sprintf("%s=%s", tokens[0][0], tokens[0][1])).
		withHeader("Cookie", fmt.Sprintf("%s=%s", tokens[1][0], tokens[1][1])).
		do()

	s.Equal(http.StatusOK, res.StatusCode)

	rowUpdatedUser := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Select("subscribe_until").
		From("users").
		Where("id = ?", userId).
		RunWith(infra.PgConn).
		QueryRow()

	var subscribeUntil sql.NullInt64
	s.Nil(rowUpdatedUser.Scan(&subscribeUntil))
	s.Equal(time.Now().Add(time.Duration(couponDuration)*time.Second).Unix(), subscribeUntil.Int64)

	rowUpdatedUserCoupon := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Select("used_at").
		From("user_coupons").
		Where("id = ?", userCouponId).
		RunWith(infra.PgConn).
		QueryRow()

	var usedAt sql.NullInt64
	s.Nil(rowUpdatedUserCoupon.Scan(&usedAt))
	s.Equal(time.Now().Unix(), usedAt.Int64)
}

func (s *UserTestSuite) Test_Post_UserSubscription_SubscribedBefore_Success() {
	tokens := getAuthToken(s.T(), infra.PgConn)

	rowFindUser := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Update("users").
		Set("subscribe_until", time.Now().Add(24*14*time.Hour).Unix()).
		Where("email = ?", "base@mail.com").
		Suffix("RETURNING id").
		RunWith(infra.PgConn).
		QueryRow()
	var userId string
	s.Nil(rowFindUser.Scan(&userId))

	couponDuration := 60 * 60 * 24 * 365
	rowCreateCoupon := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Insert("coupons").
		Columns("code", "duration_in_second", "valid_until").
		Values("NEWUSER123", couponDuration, time.Now().Add(24*14*time.Hour).Unix()).
		Suffix("RETURNING id").
		RunWith(infra.PgConn).
		QueryRow()
	var couponId string
	s.Nil(rowCreateCoupon.Scan(&couponId))

	rowCreateUserCoupon := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Insert("user_coupons").
		Columns("user_id", "coupon_id").
		Values(userId, couponId).
		Suffix("RETURNING id").
		RunWith(infra.PgConn).
		QueryRow()
	var userCouponId string
	s.Nil(rowCreateUserCoupon.Scan(&userCouponId))

	res := newHttpTest().
		withPath("/v1/users/subscribe").
		withMethod(http.MethodPost).
		withBody(map[string]interface{}{
			"coupon_code": "NEWUSER123",
		}).
		withHeader("Cookie", fmt.Sprintf("%s=%s", tokens[0][0], tokens[0][1])).
		withHeader("Cookie", fmt.Sprintf("%s=%s", tokens[1][0], tokens[1][1])).
		do()

	s.Equal(http.StatusOK, res.StatusCode)

	rowUpdatedUser := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Select("subscribe_until").
		From("users").
		Where("id = ?", userId).
		RunWith(infra.PgConn).
		QueryRow()

	var subscribeUntil sql.NullInt64
	s.Nil(rowUpdatedUser.Scan(&subscribeUntil))
	s.Equal(time.Now().Add(24*(14+365)*time.Hour).Unix(), subscribeUntil.Int64)

	rowUpdatedUserCoupon := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Select("used_at").
		From("user_coupons").
		Where("id = ?", userCouponId).
		RunWith(infra.PgConn).
		QueryRow()

	var usedAt sql.NullInt64
	s.Nil(rowUpdatedUserCoupon.Scan(&usedAt))
	s.Equal(time.Now().Unix(), usedAt.Int64)
}
