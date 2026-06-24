package infrastructure

import (
	"context"
	"log"
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/match/domain"
	"github.com/Aboody-Studios/ballr/src/pkg/events"
)

type Sweeper struct {
	MatchRepo      domain.MatchRepository
	EventPublisher events.Publisher
}

func (swpr *Sweeper) SweepStuckMatches(ctx context.Context) error {
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			cutOffTime := time.Now().Add(-2 * time.Minute)

			matches, err := swpr.MatchRepo.GetStuckMatches(ctx, cutOffTime)
			if err != nil {
				log.Printf("Query failure: %v", err)
			}

			for _, match := range matches {
				// Attempt to atomically claim the match so only one sweeper/process enqueues the job.
				claimable, err := swpr.MatchRepo.ClaimStuckMatch(ctx, match.ID)
				if err != nil {
					log.Printf("claim failure: %v", err)
					continue
				}
				if !claimable {
					// Someone else already claimed it.
					continue
				}

				// TODO!: See if metadata is necessary

				eventMap := map[string]any{
					"match_id":  match.ID,
					"video_url": match.VideoURL,
				}

				if err := swpr.EventPublisher.PublishEvent(ctx, events.Event{Type: events.EventAnalysisStart, UserID: match.UserID, Metadata: eventMap}); err != nil {
					// Publish failed; revert claim so another sweeper can retry later.
					if err2 := swpr.MatchRepo.UnclaimMatch(ctx, match.ID); err2 != nil {
						log.Printf("revert claim failed: %v", err2)
					}
					log.Printf("Publish failure: %v", err)
					continue
				}
			}
		}

	}
}
