package repository

import (
	"context"
	"database/sql"
	"embed"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib" // pgx for database/sql
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.*
var embedMigrations embed.FS

type PgLinksRepository struct {
	db *sql.DB
}

func NewPgLinksRepository(databaseDSN string) (*PgLinksRepository, error) {
	db, err := sql.Open("pgx", databaseDSN)
	if err != nil {
		return nil, err
	}

	repo := PgLinksRepository{db: db}
	err = repo.migrate()
	if err != nil {
		return nil, err
	}
	return &repo, nil
}

func (p *PgLinksRepository) Get(linkID string) (LinkEntity, error) {
	query := `select uid, original_url, link_id  from shortener.links where link_id = $1`
	var entity LinkEntity
	result := p.db.QueryRowContext(context.Background(), query, linkID)
	err := result.Scan(&entity.UID, &entity.OriginalURL, &entity.ID)
	if err != nil {
		return LinkEntity{}, err
	}
	return entity, nil
}

func (p *PgLinksRepository) Put(linkEntity LinkEntity) (string, error) {
	query := `insert into shortener.links(link_id, original_url, uid) values($1, $2, $3)`
	_, err := p.conn.Exec(context.Background(), query, linkEntity.ID, linkEntity.OriginalURL, linkEntity.UID)
	if err != nil {
		return "", err
	}
	return linkEntity.ID, nil
}

func (p *PgLinksRepository) PutBatch(linkEntities []LinkEntity) error {
	batch := make([]LinkEntity, 0, 100)
	for i, entity := range linkEntities {
		batch = append(batch, entity)
		if cap(batch) == len(batch) || i == len(linkEntities)-1 {
			if err := p.flush(batch); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *PgLinksRepository) flush(linkEntities []LinkEntity) error {
	query := `insert into shortener.links(link_id, original_url, uid) values($1, $2, $3)`

	tx, err := p.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	stmt, err := tx.PrepareContext(context.Background(), query)
	if err != nil {
		return err
	}
	defer stmt.Close() //nolint:errcheck

	for _, entity := range linkEntities {
		if _, err := stmt.ExecContext(context.Background(), entity.ID, entity.OriginalURL, entity.UID); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (p *PgLinksRepository) Count() (int, error) {
	query := `select count(*) from shortener.links`
	var count int
	err := p.db.QueryRowContext(context.Background(), query).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (p *PgLinksRepository) FindLinksByUID(uid string) ([]LinkEntity, error) {
	query := `select uid, original_url, link_id  from shortener.links where uid=$1`
	var result []LinkEntity
	rows, err := p.db.QueryContext(context.Background(), query, uid)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var entity LinkEntity
		err = rows.Scan(&entity.UID, &entity.OriginalURL, &entity.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, entity)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (p *PgLinksRepository) Status() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return p.db.PingContext(ctx)
}

func (p *PgLinksRepository) Close() error {
	return p.db.Close()
}

func (p *PgLinksRepository) migrate() error {
	goose.SetBaseFS(embedMigrations)
	goose.SetTableName("shortener.goose_db_version")
	return goose.Up(p.db, "migrations")
}
