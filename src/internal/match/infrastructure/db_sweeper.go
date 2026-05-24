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
				eventMap := map[string]any{
					"match_id":  match.ID,
					"video_url": match.VideoURL,
				}

				if err := swpr.EventPublisher.PublishEvent(ctx, match.UserID, events.EventAnalysisStart, eventMap); err != nil {
					log.Printf("Publish failure: %v", err)
					continue
				}
				if err := swpr.MatchRepo.FixStuckMatch(ctx, match.ID); err != nil {
					log.Printf("analysis flag error: %v", err)
				}
			}
		}

	}
}
