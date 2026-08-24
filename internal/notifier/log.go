package notifier

import (
	"context"
	"log"

	"github.com/requiem-glitch/personal-watcher/internal/watch"
)

type LogNotifier struct{}

func (ln LogNotifier) Notify(_ context.Context, watch watch.Watch, healthy bool) error {
	if healthy {
		log.Printf("watch %d RECOVERED: %s", watch.ID, watch.URL)
	} else {
		log.Printf("watch %d DOWN: %s", watch.ID, watch.URL)
	}
	return nil
}
