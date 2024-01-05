package rest_test

import (
	"gotinder/infra"
	"net/http"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"
)

type AuthTestSuite struct {
	suite.Suite
}

func TestAuthTestSuite(t *testing.T) {
	suite.Run(t, new(AuthTestSuite))
}

func (s *AuthTestSuite) SetupSuite() {
	pg := newPostgresTest(s.T())
	infra.NewPgConnection(pg.connStr)
}

func (s *AuthTestSuite) SetupTest() {
	pgTest.migrate(s.T(), infra.PgConn)
}

func (s *AuthTestSuite) Test_Post_AuthDirectLogin_Success() {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("Valid1234!"), bcrypt.DefaultCost)
	s.Nil(err)
	_, err = sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Insert("users").
		Columns("email", "password", "birth_of_date").
		Values("valid@mail.com", string(hashedPassword), time.Now().Unix()).
		RunWith(infra.PgConn).
		Exec()
	s.Nil(err)

	res := newHttpTest().
		withPath("/v1/auth/direct/login").
		withMethod(http.MethodPost).
		withBody(map[string]interface{}{
			"user":   "valid@mail.com",
			"passwd": "Valid1234!",
		}).
		do()

	s.Equal(http.StatusOK, res.StatusCode)
	for _, cookie := range res.Cookies() {
		s.Contains([]string{"JWT", "XSRF-TOKEN"}, cookie.Name)
	}
}

func (s *AuthTestSuite) Test_Post_AuthRegister_Success() {
	currTime := time.Now()
	res := newHttpTest().
		withPath("/v1/auth/register").
		withMethod(http.MethodPost).
		withBody(map[string]interface{}{
			"email":         "valid@mail.com",
			"password":      "Valid1234!",
			"birth_of_date": currTime.Unix(),
		}).
		do()

	row := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Select("email", "password", "birth_of_date").
		From("users").
		Where("email = $1", "valid@mail.com").
		RunWith(infra.PgConn).
		QueryRow()

	var user struct {
		Email       string
		Password    string
		BirthOfDate int64
	}
	s.Nil(row.Scan(&user.Email, &user.Password, &user.BirthOfDate))

	s.Equal(http.StatusOK, res.StatusCode)
	s.Equal("valid@mail.com", user.Email)
	s.Equal(currTime.Unix(), user.BirthOfDate)
	s.Nil(bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("Valid1234!")))
}
