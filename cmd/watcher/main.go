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
	"github.com/requiem-glitch/personal-watcher/internal/checker"
	"github.com/requiem-glitch/personal-watcher/internal/httpapi"
	"github.com/requiem-glitch/personal-watcher/internal/postgres"
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

	repo := postgres.Repository{Pool: pool}

	mux := httpapi.NewMux(httpapi.Handler{Repo: repo})
	server.Handler = mux

	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	/*checker test START*/
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	siteChecker := checker.Checker{
		Client: client,
	}
	toCheck, err := repo.GetWatch(appCtx, 3)
	if err != nil {
		log.Printf("GetWatch: %v", err)
		return
	}
	result := siteChecker.Check(appCtx, toCheck)
	err = repo.SaveCheck(appCtx, result)
	if err != nil {
		log.Printf("SaveCheck: %v", err)
		return
	}
	log.Printf("%+v", result)
	/*checker test FINISH*/
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
