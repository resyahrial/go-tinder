package rest_test

import (
	"encoding/json"
	"fmt"
	"gotinder/infra"
	"io"
	"net/http"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"
)

type RecommendationTestSuite struct {
	suite.Suite
}

func TestRecommendationTestSuite(t *testing.T) {
	suite.Run(t, new(RecommendationTestSuite))
}

func (s *RecommendationTestSuite) SetupSuite() {
	pg := newPostgresTest(s.T())
	infra.NewPgConnection(pg.connStr)
}

func (s *RecommendationTestSuite) SetupTest() {
	pgTest.migrate(s.T(), infra.PgConn)
}

func (s *RecommendationTestSuite) Test_Get_Recommendation_Success() {
	tokens := getAuthToken(s.T(), infra.PgConn)

	password := "Secret1234!"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	s.Nil(err)
	rows, err := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Insert("users").
		Columns("email", "password", "birth_of_date").
		Values("malang.1@mail.com", string(hashedPassword), time.Now().Unix()).
		Values("jakarta@mail.com", string(hashedPassword), time.Now().Unix()).
		Values("malang.2@mail.com", string(hashedPassword), time.Now().Unix()).
		Values("malang.3@mail.com", string(hashedPassword), time.Now().Unix()).
		Suffix("RETURNING id").
		RunWith(infra.PgConn).
		Query()
	s.Nil(err)

	userIds := make([]string, 0)
	for rows.Next() {
		var userId string
		s.Nil(rows.Scan(&userId))
		userIds = append(userIds, userId)
	}

	row := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Select("id").
		From("users").
		Where("email = $1", "base@mail.com").
		RunWith(infra.PgConn).
		QueryRow()

	var userId string
	s.Nil(row.Scan(&userId))
	userIds = append(userIds, userId)

	_, err = sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Insert("latest_locations").
		Columns("user_id", "updated_at", "lat", "lng").
		Values(userIds[0], time.Now().Add(-1*time.Hour).Unix(), "-7.96447", "112.687").
		Values(userIds[1], time.Now().Unix(), "-6.22956", "106.747").
		Values(userIds[2], time.Now().Add(-3*time.Hour).Unix(), "-7.95349", "112.630").
		Values(userIds[3], time.Now().Add(-6*time.Hour).Unix(), "-7.95349", "112.610").
		Values(userIds[4], time.Now().Unix(), "-7.94447", "112.647").
		RunWith(infra.PgConn).
		Exec()
	s.Nil(err)

	res := newHttpTest().
		withPath("/v1/recommendations?limit=2").
		withHeader("Cookie", fmt.Sprintf("%s=%s", tokens[0][0], tokens[0][1])).
		withHeader("Cookie", fmt.Sprintf("%s=%s", tokens[1][0], tokens[1][1])).
		do()

	s.Equal(http.StatusOK, res.StatusCode)
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	s.Nil(err)
	var response map[string]interface{}
	s.Nil(json.Unmarshal(body, &response))
	resData, ok := response["data"].([]interface{})
	s.True(ok)
	expectedResult := []string{userIds[0], userIds[2]}
	for _, rec := range resData {
		recMap, ok := rec.(map[string]interface{})
		s.True(ok)
		s.Contains(expectedResult, recMap["id"])
	}
}
