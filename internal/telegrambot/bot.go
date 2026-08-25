package telegrambot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/requiem-glitch/personal-watcher/internal/postgres"
	"github.com/requiem-glitch/personal-watcher/internal/watch"
)

type Bot struct {
	Token  string
	ChatID string
	Client *http.Client
	Repo   postgres.Repository
}

type Chat struct {
	ID int64 `json:"id"`
}

type Message struct {
	Text string `json:"text"`
	Chat Chat   `json:"chat"`
}

type Update struct {
	UpdateID int      `json:"update_id"`
	Message  *Message `json:"message"`
}

type UpdateResponse struct {
	OK     bool     `json:"ok"`
	Result []Update `json:"result"`
}

type SendMessageRequest struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

func parseInt64(posInt string) (int64, error) {
	num, err := strconv.ParseInt(posInt, 10, 64)
	if err != nil {
		return 0, err
	}
	return num, nil
}

func parseInt(posInt string) (int, error) {
	num, err := strconv.Atoi(posInt)
	if err != nil {
		return 0, err
	}
	return num, nil
}

func (b Bot) getUpdates(ctx context.Context, offset int) ([]Update, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=%d", b.Token, offset, 30)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		return []Update{}, err
	}

	resp, err := b.Client.Do(req)
	if err != nil {
		return []Update{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return []Update{}, fmt.Errorf("telegrambot return status %d", resp.StatusCode)
	}
	var updateResponse UpdateResponse
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&updateResponse)
	if err != nil {
		return []Update{}, err
	}
	if !updateResponse.OK {
		return []Update{}, fmt.Errorf("telegrambot updateResponse false")
	}
	return updateResponse.Result, nil
}

func (b Bot) Run(ctx context.Context) {
	offset := 0
	chatID, err := parseInt64(b.ChatID)
	if err != nil {
		fmt.Printf("ParseInt chatID: %v", err)
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
			updates, err := b.getUpdates(ctx, offset)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("getUpdates: %v", err)
				continue
			}
			for _, update := range updates {
				offset = update.UpdateID + 1
				if update.Message == nil {
					continue
				}
				if update.Message.Chat.ID != chatID {
					continue
				}
				args := strings.Fields(update.Message.Text)
				if len(args) == 0 {
					continue
				}
				switch args[0] {
				case "/list":
					if len(args) != 1 {
						b.reply(ctx, "USAGE: /list")
						continue
					}
					watches, err := b.Repo.ListWatches(ctx)
					if err != nil {
						log.Printf("tgbot ListWatches: %v", err)
						continue
					}
					if len(watches) == 0 {
						b.reply(ctx, "No watches")
						continue
					}
					text := []byte{}
					for _, watch := range watches {
						toWrite := fmt.Sprintf("%d | %s | %t\n", watch.ID, watch.URL, watch.Enabled)
						text = append(text, []byte(toWrite)...)
					}
					b.reply(ctx, string(text))
				case "/delete":
					if len(args) != 2 {
						b.reply(ctx, "USAGE: /delete id")
						continue
					}
					wid, err := parseInt64(args[1])
					if err != nil || wid <= 0 {
						b.reply(ctx, "id must be numeric and greater than 0")
						continue
					}
					rowsAffected, err := b.Repo.DeleteWatch(ctx, wid)
					if err != nil {
						b.reply(ctx, "server error")
						continue
					}
					if rowsAffected == 0 {
						b.reply(ctx, "not found")
						continue
					}
					b.reply(ctx, "deleted")
				case "/enable":
					if len(args) != 2 {
						b.reply(ctx, "USAGE: /enable id")
						continue
					}
					b.setEnabled(ctx, args[1], true)
				case "/disable":
					if len(args) != 2 {
						b.reply(ctx, "USAGE: /disable id")
						continue
					}
					b.setEnabled(ctx, args[1], false)
				case "/add":
					if len(args) != 4 {
						b.reply(ctx, "USAGE: /add <url> <expected_status> <interval_seconds>")
						continue
					}
					url := args[1]
					expectedStatus, err := parseInt(args[2])
					if err != nil || expectedStatus < 100 || expectedStatus > 599 {
						b.reply(ctx, "expected status must be numeric and in interval 100..599")
						continue
					}
					intervalSeconds, err := parseInt(args[3])
					if err != nil || intervalSeconds <= 0 {
						b.reply(ctx, "interval seconds must be numeric and greater than 0")
						continue
					}
					params := watch.CreateParams{
						URL:            url,
						ExpectedStatus: expectedStatus,
						IntervalSec:    intervalSeconds,
					}
					watch, err := b.Repo.CreateWatch(ctx, params)
					if err != nil {
						b.reply(ctx, "server error")
						continue
					}
					replyStr := fmt.Sprintf("created watch id=%d", watch.ID)
					b.reply(ctx, replyStr)
				default:
					b.reply(ctx, "unknown command")
				}
			}

		}
	}
}

func (b Bot) sendMessage(ctx context.Context, text string) error {
	message := SendMessageRequest{
		ChatID: b.ChatID,
		Text:   text,
	}
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.telegram.org/bot"+b.Token+"/sendMessage",
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.Client.Do(req)
	if err != nil {

		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram returned status %d", resp.StatusCode)
	}
	return nil
}

func (b Bot) reply(ctx context.Context, text string) {
	err := b.sendMessage(ctx, text)
	if err != nil {
		log.Printf("sendMessage: %v", err)
	}
}

func (b Bot) setEnabled(ctx context.Context, id string, value bool) {
	wid, err := parseInt64(id)
	if err != nil || wid <= 0 {
		b.reply(ctx, "id must be numeric and greater than 0")
		return
	}
	enabled := value
	params := watch.UpdateParams{Enabled: &enabled}
	_, err = b.Repo.UpdateWatch(ctx, wid, params)
	if err == pgx.ErrNoRows {
		b.reply(ctx, "not found")
		return
	}
	if err != nil {
		b.reply(ctx, "server error")
		return
	}
	if value {
		b.reply(ctx, "enabled")
	} else {
		b.reply(ctx, "disabled")
	}
}
