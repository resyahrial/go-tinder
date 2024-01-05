package rest_test

import (
	"fmt"
	"gotinder/infra"
	"net/http"
	"strconv"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/suite"
)

type LocationTestSuite struct {
	suite.Suite
}

func TestLocationTestSuite(t *testing.T) {
	suite.Run(t, new(LocationTestSuite))
}

func (s *LocationTestSuite) SetupSuite() {
	pg := newPostgresTest(s.T())
	infra.NewPgConnection(pg.connStr)
}

func (s *LocationTestSuite) SetupTest() {
	pgTest.migrate(s.T(), infra.PgConn)
}

func (s *LocationTestSuite) Test_Post_Location_Success() {
	tokens := getAuthToken(s.T(), infra.PgConn)

	lat, lng := "-7.97727", "112.6341"
	res := newHttpTest().
		withPath("/v1/locations").
		withMethod(http.MethodPost).
		withBody(map[string]interface{}{
			"lat": lat,
			"lng": lng,
		}).
		withHeader("Cookie", fmt.Sprintf("%s=%s", tokens[0][0], tokens[0][1])).
		withHeader("Cookie", fmt.Sprintf("%s=%s", tokens[1][0], tokens[1][1])).
		do()

	s.Equal(http.StatusOK, res.StatusCode)
	row := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Select("latest_locations.lat", "latest_locations.lng", fmt.Sprintf("st_distancesphere(latest_locations.location::geometry,ST_SetSRID(ST_MakePoint(%s,%s), 4326)) AS distance", lat, lng)).
		From("latest_locations").
		Join("users ON users.id = latest_locations.user_id").
		Where("users.email = $1", "base@mail.com").
		RunWith(infra.PgConn).
		QueryRow()

	var loc struct {
		Lat      string
		Lng      string
		Location string
	}
	s.Nil(row.Scan(&loc.Lat, &loc.Lng, &loc.Location))
	s.Equal(lat, loc.Lat)
	s.Equal(lng, loc.Lng)
	parsedLocation, err := strconv.ParseFloat(loc.Location, 64)
	s.Nil(err)
	s.Less(parsedLocation, float64(1))
}
