package batch

import (
	"context"

	"github.com/zaz600/go-musthave-shortener/internal/entity"
	"github.com/zaz600/go-musthave-shortener/internal/infrastructure/repository"
)

// BatchWriter интерфейс для сервиса, сокращающего ссылки порциями
type BatchWriter interface {
	// Add добавление ссылки
	Add(ctx context.Context, e entity.LinkEntity) error
	// Flush запись в хранилище и очистка буфера
	Flush(ctx context.Context) error
}

// Service сервис, сокращающий ссылки заданными порциями
type Service struct {
	// batchSize размер пачки. Когда в буфере накапливается указанное число ссылок,
	// происходит сброс буфера и сохранение ссылок в хранилище.
	batchSize int
	// buffer буфер для временного хранения ссылок перед их записью в хранилище.
	buffer []entity.LinkEntity
	// linksRepository хранилище ссылок, в которое производится запись при заполнении буфера
	linksRepository repository.LinksRepository
}

func NewBatchService(batchSize int, repository repository.LinksRepository) *Service {
	return &Service{
		batchSize:       batchSize,
		buffer:          make([]entity.LinkEntity, 0, batchSize),
		linksRepository: repository,
	}
}

// Add добавление ссылки
func (b *Service) Add(ctx context.Context, e entity.LinkEntity) error {
	b.buffer = append(b.buffer, e)
	if cap(b.buffer) == len(b.buffer) {
		if err := b.Flush(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Flush запись в хранилище и очистка буфера
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
