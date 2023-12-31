package rest

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"golang.org/x/sync/errgroup"
)

type (
	// Cleanup is a type to define function which has to call on shutdown
	CleanupFn func() (name string, fn func())

	// v1 is a type to group register function
	v1 struct {
		group *gin.RouterGroup
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
		time.Sleep(5 * time.Second)
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

	v1Group := v1{h.Group("/v1")}
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
