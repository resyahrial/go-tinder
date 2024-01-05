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

type CouponTestSuite struct {
	suite.Suite
}

func TestCouponTestSuite(t *testing.T) {
	suite.Run(t, new(CouponTestSuite))
}

func (s *CouponTestSuite) SetupSuite() {
	pg := newPostgresTest(s.T())
	infra.NewPgConnection(pg.connStr)
}

func (s *CouponTestSuite) SetupTest() {
	pgTest.migrate(s.T(), infra.PgConn)
}

func (s *CouponTestSuite) Test_Post_Coupon_Success() {
	tokens := getAuthToken(s.T(), infra.PgConn)

	currTime := time.Now()
	res := newHttpTest().
		withPath("/v1/coupons").
		withMethod(http.MethodPost).
		withBody(map[string]interface{}{
			"code":               "NEWUSER123",
			"duration_in_second": 60 * 60 * 24 * 365,
			"valid_until":        currTime.Add(24 * 14 * time.Hour).Unix(),
		}).
		withHeader("Cookie", fmt.Sprintf("%s=%s", tokens[0][0], tokens[0][1])).
		withHeader("Cookie", fmt.Sprintf("%s=%s", tokens[1][0], tokens[1][1])).
		do()

	s.Equal(http.StatusOK, res.StatusCode)

	row := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Select("duration_in_second", "valid_until").
		From("coupons").
		Where("code = $1", "NEWUSER123").
		RunWith(infra.PgConn).
		QueryRow()

	var coupon struct {
		DurationInSecond int64
		ValidUntil       int64
	}
	s.Nil(row.Scan(&coupon.DurationInSecond, &coupon.ValidUntil))
	s.EqualValues(60*60*24*365, coupon.DurationInSecond)
	s.Equal(currTime.Add(24*14*time.Hour).Unix(), coupon.ValidUntil)
}

func (s *CouponTestSuite) Test_Post_CouponApply_Success() {
	tokens := getAuthToken(s.T(), infra.PgConn)

	rowCreateCoupon := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Insert("coupons").
		Columns("code", "duration_in_second", "valid_until").
		Values("NEWUSER123", 60*60*24*365, time.Now().Add(24*14*time.Hour).Unix()).
		Suffix("RETURNING id").
		RunWith(infra.PgConn).
		QueryRow()
	var couponId string
	s.Nil(rowCreateCoupon.Scan(&couponId))

	rowCreateUser := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Insert("users").
		Columns("email", "password", "birth_of_date").
		Values("sub@mail.com", "password", time.Now().Unix()).
		Suffix("RETURNING id").
		RunWith(infra.PgConn).
		QueryRow()
	var subId string
	s.Nil(rowCreateUser.Scan(&subId))

	res := newHttpTest().
		withPath("/v1/coupons/apply").
		withMethod(http.MethodPost).
		withBody(map[string]interface{}{
			"code":    "NEWUSER123",
			"user_id": subId,
		}).
		withHeader("Cookie", fmt.Sprintf("%s=%s", tokens[0][0], tokens[0][1])).
		withHeader("Cookie", fmt.Sprintf("%s=%s", tokens[1][0], tokens[1][1])).
		do()

	s.Equal(http.StatusOK, res.StatusCode)

	rowUserCoupon := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Select("user_id", "coupon_id", "used_at").
		From("user_coupons").
		Where("user_id = $1", subId).
		RunWith(infra.PgConn).
		QueryRow()

	var userCoupon struct {
		UserId   string
		CouponId string
		UsedAt   sql.NullInt64
	}
	s.Nil(rowUserCoupon.Scan(&userCoupon.UserId, &userCoupon.CouponId, &userCoupon.UsedAt))
	s.Equal(subId, userCoupon.UserId)
	s.Equal(couponId, userCoupon.CouponId)
	s.False(userCoupon.UsedAt.Valid)
}
