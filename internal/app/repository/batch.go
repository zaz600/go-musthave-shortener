package repository

import (
	"context"
)

type BatchWriter interface {
	Add(ctx context.Context, e LinkEntity) error
	Flush(ctx context.Context) error
}

type BatchService struct {
	batchSize  int
	buffer     []LinkEntity
	repository LinksRepository
}

func NewBatchService(batchSize int, repository LinksRepository) *BatchService {
	return &BatchService{
		batchSize:  batchSize,
		buffer:     make([]LinkEntity, 0, batchSize),
		repository: repository,
	}
}

func (b *BatchService) Add(ctx context.Context, e LinkEntity) error {
	b.buffer = append(b.buffer, e)
	if cap(b.buffer) == len(b.buffer) {
		if err := b.Flush(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (b *BatchService) Flush(ctx context.Context) error {
	if len(b.buffer) == 0 {
		return nil
	}
	err := b.repository.PutBatch(ctx, b.buffer)
	if err != nil {
		return err
	}
	b.buffer = b.buffer[:0]
	return nil
}
