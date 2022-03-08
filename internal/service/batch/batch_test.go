package batch

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zaz600/go-musthave-shortener/internal/entity"
	"github.com/zaz600/go-musthave-shortener/internal/infrastructure/repository"
)

func TestBatchService_Add(t *testing.T) {
	batchSize := 5
	repo := repository.NewInMemoryLinksRepository(context.Background(), nil)
	batch := NewBatchService(batchSize, repo)
	for i := 0; i < batchSize; i++ {
		err := batch.Add(context.Background(), entity.NewLinkEntity(fmt.Sprintf("http://ya.ru/?%d", i), "123456"))
		require.NoError(t, err)
		if i < batchSize-1 {
			// пока предел буфера не достигнут, ничего не должно писаться в репозиторий
			count, err := repo.Count(context.TODO())
			require.NoError(t, err)
			assert.Equal(t, count, 0)
		} else {
			// после последнего элемента, батч должен сделать запись в репозиторий
			count, err := repo.Count(context.TODO())
			require.NoError(t, err)
			assert.Equal(t, count, batchSize)
			// и обнулить буфер
			assert.Len(t, batch.buffer, 0)
		}
	}
	// и еще одна попытка записи уже после сброса буфера в репозиторий
	err := batch.Add(context.Background(), entity.NewLinkEntity("http://ya.ru/?1000", "123456"))
	require.NoError(t, err)
	count, err := repo.Count(context.TODO())
	require.NoError(t, err)
	assert.Equal(t, count, batchSize)
	assert.Len(t, batch.buffer, 1)
}

func TestBatchService_Flush(t *testing.T) {
	batchSize := 5
	nRec := 2
	repo := repository.NewInMemoryLinksRepository(context.Background(), nil)
	batch := NewBatchService(batchSize, repo)
	for i := 0; i < nRec; i++ {
		err := batch.Add(context.Background(), entity.NewLinkEntity(fmt.Sprintf("http://ya.ru/?%d", i), "123456"))
		require.NoError(t, err)
	}
	// вызов Flush должен скинуть в репозиторий все, что есть в буфере, даже если предел буфера не достигнут
	err := batch.Flush(context.Background())
	require.NoError(t, err)
	count, err := repo.Count(context.TODO())
	require.NoError(t, err)
	assert.Equal(t, count, nRec)
	assert.Len(t, batch.buffer, 0)
}

func BenchmarkService_Add(b *testing.B) {
	batchSize := 100
	repo := repository.NewInMemoryLinksRepository(context.Background(), nil)
	batch := NewBatchService(batchSize, repo)

	for i := 0; i < b.N; i++ {
		_ = batch.Add(context.Background(), entity.NewLinkEntity(fmt.Sprintf("http://ya.ru/?%d", i), "123456"))
	}
}
