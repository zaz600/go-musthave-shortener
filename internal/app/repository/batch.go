package repository

import "context"

type BatchWriter interface {
	Add(ctx context.Context, e LinkEntity) error
	Flush(ctx context.Context) error
}

type Batch struct {
	batchSize  int
	buffer     []LinkEntity
	repository LinksRepository
}

func NewBatch(batchSize int, repository LinksRepository) *Batch {
	return &Batch{
		batchSize:  batchSize,
		buffer:     make([]LinkEntity, 0, batchSize),
		repository: repository,
	}
}

func (b Batch) Add(ctx context.Context, e LinkEntity) error {
	b.buffer = append(b.buffer, e)
	if cap(b.buffer) == len(b.buffer) {
		if err := b.Flush(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (b Batch) Flush(ctx context.Context) error {
	err := b.repository.PutBatch(ctx, b.buffer)
	if err != nil {
		return err
	}
	b.buffer = b.buffer[:0]
	return nil
}
