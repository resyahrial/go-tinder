package rest_test

import (
	"database/sql"
	"fmt"
	"gotinder/infra"
	"net/http"
	"testing"

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

func (s *UserTestSuite) Test_Patch_UserSubscription_Success() {
	tokens := getAuthToken(s.T(), infra.PgConn)

	res := newHttpTest().
		withPath("/v1/users/subscribe").
		withMethod(http.MethodPatch).
		withHeader("Cookie", fmt.Sprintf("%s=%s", tokens[0][0], tokens[0][1])).
		withHeader("Cookie", fmt.Sprintf("%s=%s", tokens[1][0], tokens[1][1])).
		do()

	s.Equal(http.StatusOK, res.StatusCode)

	row := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Select("subscribe_until").
		From("users").
		Where("users.email = $1", "base@mail.com").
		RunWith(infra.PgConn).
		QueryRow()

	var subscribeUntil sql.NullInt64
	s.Nil(row.Scan(&subscribeUntil))
	s.True(subscribeUntil.Valid)
}
