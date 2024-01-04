package rest

import (
	"context"
	"database/sql"
	"fmt"
	"gotinder/infra"
	"log"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-pkgz/auth/token"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

type (
	// Cleanup is a type to define function which has to call on shutdown
	CleanupFn func() (name string, fn func())

	// v1 is a type to group register function
	v1 struct {
		group *gin.RouterGroup
		auth  *authService
	}
)

// New run server with graceful shutdown
func New(port int, cleanupFns ...CleanupFn) {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
		<-stop
		cancel()
	}()

	if port == 0 {
		port = 3000
	}
	address := fmt.Sprintf(":%d", port)
	srv := &http.Server{
		Handler:           NewHandler(),
		ReadHeaderTimeout: 1 * time.Minute,
	}
	srv.Addr = address
	for _, cleanupFn := range cleanupFns {
		srv.RegisterOnShutdown(func() {
			name, fn := cleanupFn()
			log.Println(name)
			fn()
		})
	}

	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		fmt.Println("server listening on port", port)
		err := srv.ListenAndServe()
		return err
	})

	eg.Go(func() error {
		<-egCtx.Done()
		log.Println("shutting down server")
		err := srv.Shutdown(context.Background())
		log.Println("server shutted down gracefully")
		return err
	})

	if err := eg.Wait(); err != nil {
		fmt.Printf("fail to exit server: %s\n", err)
	}
}

// NewHandler register handler on its path for restful API
func NewHandler() *gin.Engine {
	binding.Validator = new(bindValidator)
	h := gin.Default()

	h.GET("/", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	authSvc := new(authService)
	authSvc.init()
	v1Group := v1{
		group: h.Group("/v1"),
		auth:  authSvc,
	}
	registerHandler[v1](v1Group)

	return h
}

// registerHandler register all handler on group routing
func registerHandler[T any](group T) {
	methodFinder := reflect.TypeOf(&group)
	for i := 0; i < methodFinder.NumMethod(); i++ {
		method := methodFinder.Method(i)
		method.Func.Call([]reflect.Value{reflect.ValueOf(&group)})
	}
}

// asGin converts middleware to the gin middleware handler.
func asGin(middleware func(next http.Handler) http.Handler) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		var skip = true
		var handler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
			gctx.Request = r
			skip = false
		}
		middleware(handler).ServeHTTP(gctx.Writer, gctx.Request)
		switch {
		case skip:
			gctx.Abort()
		default:
			gctx.Next()
		}
	}
}

// enrichActor will enrich current user information on context
func enrichActor(ctx *gin.Context) {
	u := token.MustGetUserInfo(ctx.Request)
	user := &u

	findUserQuery, _, err := sq.
		StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Select("id", "subscribe_until").
		From("users").
		Where("email = $1").
		ToSql()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": errors.Wrap(err, "failed to build find user query").Error(),
		})
		return
	}

	row := infra.PgConn.QueryRow(findUserQuery, user.Name)
	var userID string
	var subscribeUntil sql.NullInt64
	if err := row.Scan(&userID, &subscribeUntil); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": "user not found",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": errors.Wrap(err, "failed to find user").Error(),
		})
		return
	}

	user.SetStrAttr("user_id", userID)
	user.SetPaidSub(subscribeUntil.Valid && time.Now().Before(time.Unix(subscribeUntil.Int64, 0)))

	ctx.Request = token.SetUserInfo(ctx.Request, u)
}
