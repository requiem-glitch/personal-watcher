package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/requiem-glitch/personal-watcher/internal/httpapi"
)

func main() {

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	server := http.Server{}
	if val, ok := os.LookupEnv("PORT"); ok && val != "" {
		server.Addr = ":" + val
	} else {
		server.Addr = ":8080"
	}

	mux := httpapi.NewMux()
	server.Handler = mux

	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Println("ctrl+c found")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()
	err := server.Shutdown(shutdownCtx)
	if err != nil {
		log.Fatal(err)
	}
}
