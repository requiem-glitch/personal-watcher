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
	"github.com/requiem-glitch/personal-watcher/internal/notifier"
	"github.com/requiem-glitch/personal-watcher/internal/postgres"
	"github.com/requiem-glitch/personal-watcher/internal/scheduler"
	"github.com/requiem-glitch/personal-watcher/internal/telegrambot"
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

	//Scheduler start
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	siteChecker := checker.Checker{
		Client: client,
	}

	tgAPI, exists := os.LookupEnv("TELEGRAM_BOT_TOKEN")
	if !exists || tgAPI == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN NOT FOUND OR EMPTY")
	}
	tgChatID, exists := os.LookupEnv("TELEGRAM_CHAT_ID")
	if !exists || tgChatID == "" {
		log.Fatal("TELEGRAM_CHAT_ID NOT FOUND OR EMPTY")
	}
	telegramNotifier := notifier.TelegramNotifier{
		Token:  tgAPI,
		ChatID: tgChatID,
		Client: client,
	}
	siteScheduler := scheduler.Scheduler{
		Repo:     repo,
		Checker:  siteChecker,
		Every:    5 * time.Second,
		Notifier: telegramNotifier,
	}

	go siteScheduler.Run(appCtx)

	// TelegramBot START
	botClient := &http.Client{
		Timeout: 40 * time.Second,
	}
	bot := telegrambot.Bot{
		Token:  tgAPI,
		ChatID: tgChatID,
		Client: botClient,
		Repo:   repo,
	}

	go bot.Run(appCtx)

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
