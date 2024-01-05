package rest_test

import (
	"fmt"
	"gotinder/infra"
	"net/http"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"
)

type ActionTestSuite struct {
	suite.Suite
}

func TestActionTestSuite(t *testing.T) {
	suite.Run(t, new(ActionTestSuite))
}

func (s *ActionTestSuite) SetupSuite() {
	pg := newPostgresTest(s.T())
	infra.NewPgConnection(pg.connStr)
}

func (s *ActionTestSuite) SetupTest() {
	pgTest.migrate(s.T(), infra.PgConn)
	redis := newRedisTest(s.T())
	infra.NewRedisPool(redis.connStr, "", 0)
}

func (s *ActionTestSuite) Test_Post_ActionLike_Success() {
	tokens := getAuthToken(s.T(), infra.PgConn)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("Secret1234!"), bcrypt.DefaultCost)
	s.Nil(err)
	userRow := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Insert("users").
		Columns("email", "password", "birth_of_date").
		Values("target@mail.com", string(hashedPassword), time.Now().Unix()).
		Suffix("RETURNING id").
		RunWith(infra.PgConn).
		QueryRow()
	s.Nil(err)
	var targetId string
	s.Nil(userRow.Scan(&targetId))

	res := newHttpTest().
		withPath("/v1/actions/likes").
		withMethod(http.MethodPost).
		withBody(map[string]interface{}{
			"id": targetId,
		}).
		withHeader("Cookie", fmt.Sprintf("%s=%s", tokens[0][0], tokens[0][1])).
		withHeader("Cookie", fmt.Sprintf("%s=%s", tokens[1][0], tokens[1][1])).
		do()

	s.Equal(http.StatusOK, res.StatusCode)
	row := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Select("likes.target_id", "likes.self_id").
		From("likes").
		Join("users ON users.id = likes.self_id").
		Where("users.email = $1", "base@mail.com").
		RunWith(infra.PgConn).
		QueryRow()

	var recordedTargetId, selfId string
	s.Nil(row.Scan(&recordedTargetId, &selfId))
	s.Equal(targetId, recordedTargetId)

	conn := infra.RedisPool.Get()
	defer conn.Close()
	actionKey := fmt.Sprintf("action-%s", selfId)
	cached, err := redis.Strings(conn.Do("SMEMBERS", actionKey))
	s.Nil(err)
	s.Len(cached, 1)
	s.Contains(cached, targetId)
}

func (s *ActionTestSuite) Test_Post_ActionPass_Success() {
	tokens := getAuthToken(s.T(), infra.PgConn)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("Secret1234!"), bcrypt.DefaultCost)
	s.Nil(err)
	userRow := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Insert("users").
		Columns("email", "password", "birth_of_date").
		Values("target@mail.com", string(hashedPassword), time.Now().Unix()).
		Suffix("RETURNING id").
		RunWith(infra.PgConn).
		QueryRow()
	s.Nil(err)
	var targetId string
	s.Nil(userRow.Scan(&targetId))

	res := newHttpTest().
		withPath("/v1/actions/passes").
		withMethod(http.MethodPost).
		withBody(map[string]interface{}{
			"id": targetId,
		}).
		withHeader("Cookie", fmt.Sprintf("%s=%s", tokens[0][0], tokens[0][1])).
		withHeader("Cookie", fmt.Sprintf("%s=%s", tokens[1][0], tokens[1][1])).
		do()

	s.Equal(http.StatusOK, res.StatusCode)
	row := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Select("passes.target_id", "passes.self_id").
		From("passes").
		Join("users ON users.id = passes.self_id").
		Where("users.email = $1", "base@mail.com").
		RunWith(infra.PgConn).
		QueryRow()

	var recordedTargetId, selfId string
	s.Nil(row.Scan(&recordedTargetId, &selfId))
	s.Equal(targetId, recordedTargetId)

	conn := infra.RedisPool.Get()
	defer conn.Close()
	actionKey := fmt.Sprintf("action-%s", selfId)
	cached, err := redis.Strings(conn.Do("SMEMBERS", actionKey))
	s.Nil(err)
	s.Len(cached, 1)
	s.Contains(cached, targetId)
}
