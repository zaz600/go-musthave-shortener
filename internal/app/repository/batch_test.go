package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchService_Add(t *testing.T) {
	batchSize := 5
	repo := NewInMemoryLinksRepository(nil)
	batch := NewBatchService(batchSize, repo)
	for i := 0; i < batchSize; i++ {
		err := batch.Add(context.Background(), NewLinkEntity(fmt.Sprintf("http://ya.ru/?%d", i), "123456"))
		require.NoError(t, err)
		if i < batchSize-1 {
			// пока предел буфера не достигнут, ничего не должно писаться в репозиторий
			assert.Len(t, repo.db, 0)
		} else {
			// после последнего элемента, батч должен сделать запись в репозиторий
			assert.Len(t, repo.db, batchSize)
			// и обнулить буфер
			assert.Len(t, batch.buffer, 0)
		}
	}
	// и еще одна попытка записи уже после сброса буфера в репозиторий
	err := batch.Add(context.Background(), NewLinkEntity("http://ya.ru/?1000", "123456"))
	require.NoError(t, err)
	assert.Len(t, repo.db, batchSize)
	assert.Len(t, batch.buffer, 1)
}

func TestBatchService_Flush(t *testing.T) {
	batchSize := 5
	nRec := 2
	repo := NewInMemoryLinksRepository(nil)
	batch := NewBatchService(batchSize, repo)
	for i := 0; i < nRec; i++ {
		err := batch.Add(context.Background(), NewLinkEntity(fmt.Sprintf("http://ya.ru/?%d", i), "123456"))
		require.NoError(t, err)
	}
	// вызов Flush должен скинуть в репозиторий все, что есть в буфере, даже если предел буфера не достигнут
	err := batch.Flush(context.Background())
	require.NoError(t, err)
	assert.Len(t, repo.db, nRec)
	assert.Len(t, batch.buffer, 0)
}
