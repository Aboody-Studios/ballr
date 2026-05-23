package infrastructure

import (
	"context"

	"github.com/Aboody-Studios/ballr/src/internal/match/domain"
)

type ChannelEventPublisher struct {
	Events chan domain.VideoUpload
}

func (cep ChannelEventPublisher) Publish(ctx context.Context, vu domain.VideoUpload) error {
	cep.Events <- vu

}
