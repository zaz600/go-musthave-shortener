package batch

import (
	"context"

	"github.com/zaz600/go-musthave-shortener/internal/entity"
	"github.com/zaz600/go-musthave-shortener/internal/infrastructure/repository"
)

type BatchWriter interface {
	Add(ctx context.Context, e entity.LinkEntity) error
	Flush(ctx context.Context) error
}

type Service struct {
	batchSize       int
	buffer          []entity.LinkEntity
	linksRepository repository.LinksRepository
}

func NewBatchService(batchSize int, repository repository.LinksRepository) *Service {
	return &Service{
		batchSize:       batchSize,
		buffer:          make([]entity.LinkEntity, 0, batchSize),
		linksRepository: repository,
	}
}

func (b *Service) Add(ctx context.Context, e entity.LinkEntity) error {
	b.buffer = append(b.buffer, e)
	if cap(b.buffer) == len(b.buffer) {
		if err := b.Flush(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (b *Service) Flush(ctx context.Context) error {
	if len(b.buffer) == 0 {
		return nil
	}
	err := b.linksRepository.PutBatch(ctx, b.buffer)
	if err != nil {
		return err
	}
	b.buffer = b.buffer[:0]
	return nil
}
