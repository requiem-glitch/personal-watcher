package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/requiem-glitch/personal-watcher/internal/watch"
)

type TelegramNotifier struct {
	Token  string
	ChatID string
	Client *http.Client
}

type TelegramMessage struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

func (tn TelegramNotifier) Notify(ctx context.Context, watch watch.Watch, healthy bool) error {
	var msg string
	if healthy {
		msg = "UP: " + watch.URL
	} else {
		msg = "DOWN: " + watch.URL
	}

	message := TelegramMessage{
		ChatID: tn.ChatID,
		Text:   msg,
	}
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.telegram.org/bot"+tn.Token+"/sendMessage",
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := tn.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram returned status %d", resp.StatusCode)
	}
	return nil
}
