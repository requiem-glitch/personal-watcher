package notifier

import (
	"context"

	"github.com/requiem-glitch/personal-watcher/internal/watch"
)

type Notifier interface {
	Notify(ctx context.Context, watch watch.Watch, healthy bool) error
}
