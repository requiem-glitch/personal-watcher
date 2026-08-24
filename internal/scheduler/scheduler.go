package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/requiem-glitch/personal-watcher/internal/checker"
	"github.com/requiem-glitch/personal-watcher/internal/postgres"
)

type Scheduler struct {
	Repo    postgres.Repository
	Checker checker.Checker
	Every   time.Duration
}

func (s Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.Every)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			readyToRun, err := s.Repo.ListDueWatches(ctx)
			if err != nil {
				log.Printf("ListDueWatches: %v", err)
				continue
			}

			for _, watch := range readyToRun {
				result := s.Checker.Check(ctx, watch)
				err = s.Repo.SaveCheck(ctx, result)
				if err != nil {
					log.Printf("SaveCheck watch %d: %v", watch.ID, err)
					continue
				}
			}

		case <-ctx.Done():
			return
		}
	}
}
