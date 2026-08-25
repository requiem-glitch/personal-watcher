package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/requiem-glitch/personal-watcher/internal/checker"
	"github.com/requiem-glitch/personal-watcher/internal/notifier"
	"github.com/requiem-glitch/personal-watcher/internal/postgres"
	"github.com/requiem-glitch/personal-watcher/internal/watch"
)

type Scheduler struct {
	Repo     postgres.Repository
	Checker  checker.Checker
	Every    time.Duration
	Notifier notifier.Notifier
	Workers  int
}

func (s Scheduler) Run(ctx context.Context) {
	if s.Workers <= 0 {
		log.Println("workers must be greater than 0")
		return
	}
	if s.Every <= 0 {
		log.Println("every must be greater than 0")
		return
	}

	ticker := time.NewTicker(s.Every)
	defer ticker.Stop()
	jobs := make(chan watch.Watch)
	wg := sync.WaitGroup{}

	/*solution of in progress problem start*/
	mu := sync.Mutex{}
	inProgress := make(map[int64]bool)
	/*solution of in progress problem end*/

	for range s.Workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case watch := <-jobs:
					func() {
						defer func() {
							mu.Lock()
							delete(inProgress, watch.ID)
							mu.Unlock()
						}()
						healthy, found, err := s.Repo.GetLastHealth(ctx, watch.ID)
						if err != nil {
							log.Printf("GetLastHealth %d: %v", watch.ID, err)
							return
						}

						result := s.Checker.Check(ctx, watch)

						err = s.Repo.SaveCheck(ctx, result)
						if err != nil {
							log.Printf("SaveCheck watch %d: %v", watch.ID, err)
							return
						}

						if found && healthy != result.Healthy {
							err = s.Notifier.Notify(ctx, watch, result.Healthy)
							if err != nil {
								log.Printf("Notify: %v", err)
							}
						}
					}()

				case <-ctx.Done():
					return
				}

			}

		}()
	}
	defer wg.Wait()
	for {
		select {
		case <-ticker.C:
			readyToRun, err := s.Repo.ListDueWatches(ctx)
			if err != nil {
				log.Printf("ListDueWatches: %v", err)
				continue
			}

			for _, watch := range readyToRun {
				mu.Lock()
				_, exists := inProgress[watch.ID]
				if exists {
					mu.Unlock()
					continue
				} else {
					inProgress[watch.ID] = true
				}
				mu.Unlock()

				select {
				case jobs <- watch:
				case <-ctx.Done():
					return
				}

			}

		case <-ctx.Done():
			return
		}
	}
}
