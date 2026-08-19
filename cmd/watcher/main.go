package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/requiem-glitch/personal-watcher/internal/httpapi"
)

func main() {

	/* POSTGRES BLOCK START */
	dbURL, ok := os.LookupEnv("DATABASE_URL")
	if !ok || dbURL == "" {
		log.Fatal("DATABASE_URL NOT FOUND OR EMPTY")
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatal(err, "POOL CREATING ERROR")
	}
	defer pool.Close()
	pingCtx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()
	err = pool.Ping(pingCtx)
	if err != nil {
		log.Fatal("POOL PING ERROR")
	}
	/* POSTGRES BLOCK END */

	appCtx, stop := signal.NotifyContext(
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

	mux := httpapi.NewMux(httpapi.Handler{Pool: pool})
	server.Handler = mux

	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-appCtx.Done()
	log.Println("ctrl+c found")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()
	err = server.Shutdown(shutdownCtx)
	if err != nil {
		log.Fatal(err)
	}
}
