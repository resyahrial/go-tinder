package rest

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

type CleanupFn func() (name string, fn func())

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

func NewHandler() *gin.Engine {
	h := gin.Default()

	h.GET("/", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})

	v1 := h.Group("/v1")

	authService := newAuthService()
	authHandler, _ := authService.Handlers()
	v1.Match([]string{http.MethodGet, http.MethodPost}, "/auth/*provider", gin.WrapH(authHandler))

	return h
}
