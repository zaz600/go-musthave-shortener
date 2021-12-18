package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4"
)

type PgLinksRepository struct {
	conn *pgx.Conn
}

func NewPgLinksRepository(databaseDSN string) (*PgLinksRepository, error) {
	conn, err := pgx.Connect(context.Background(), databaseDSN)
	if err != nil {
		return nil, err
	}
	return &PgLinksRepository{conn: conn}, nil
}

func (p *PgLinksRepository) Get(linkID string) (LinkEntity, error) {
	// TODO implement me
	panic("implement me")
}

func (p *PgLinksRepository) Put(linkEntity LinkEntity) (string, error) {
	// TODO implement me
	panic("implement me")
}

func (p *PgLinksRepository) Count() int {
	// TODO implement me
	panic("implement me")
}

func (p *PgLinksRepository) FindLinksByUID(uuid string) []LinkEntity {
	// TODO implement me
	panic("implement me")
}

func (p *PgLinksRepository) Status() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return p.conn.Ping(ctx)
}

func (p *PgLinksRepository) Close() error {
	return p.conn.Close(context.Background())
}
