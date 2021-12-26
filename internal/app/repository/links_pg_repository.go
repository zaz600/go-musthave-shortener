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
	repo := PgLinksRepository{conn: conn}
	err = repo.migrate()
	if err != nil {
		return nil, err
	}
	return &repo, nil
}

func (p *PgLinksRepository) Get(linkID string) (LinkEntity, error) {
	query := `select uid, original_url, link_id  from shortener.links where link_id = $1`
	var entity LinkEntity
	result := p.conn.QueryRow(context.Background(), query, linkID)
	err := result.Scan(&entity.UID, &entity.OriginalURL, &entity.ID)
	if err != nil {
		return LinkEntity{}, err
	}
	return entity, nil
}

// Put сохраняет ссылку в БД. Если оригинальная ссылка уже есть среди сокращенных, возвращает ошибку LinkExistsError
func (p *PgLinksRepository) Put(linkEntity LinkEntity) (string, error) {
	query := `
WITH new_link AS (
    INSERT INTO shortener.links(link_id, original_url, uid) VALUES ($1, $2, $3)
    ON CONFLICT(original_url) DO NOTHING
    RETURNING link_id
) SELECT COALESCE(
    (SELECT link_id FROM new_link),
    (SELECT link_id FROM shortener.links WHERE original_url = $2)
);`
	var linkID string
	err := p.conn.QueryRow(context.Background(), query, linkEntity.ID, linkEntity.OriginalURL, linkEntity.UID).Scan(&linkID)
	if err != nil {
		return "", err
	}
	if linkEntity.ID != linkID {
		// хотели положить в бд ссылку с одним коротким айди,
		// а вернулся айди ранее сохкращеной ссылки
		return "", NewLinkExistsError(linkID)
	}
	return linkID, nil
}

func (p *PgLinksRepository) PutBatch(linkEntities []LinkEntity) error {
	batch := make([]LinkEntity, 0, 100)
	for i, entity := range linkEntities {
		batch = append(batch, entity)
		if cap(batch) == len(batch) || i == len(linkEntities)-1 {
			if err := p.Flush(batch); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *PgLinksRepository) Flush(linkEntities []LinkEntity) error {
	query := `insert into shortener.links(link_id, original_url, uid) values($1, $2, $3)`

	tx, err := p.conn.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck

	stmt, err := tx.Prepare(context.Background(), "insert link", query)
	if err != nil {
		return err
	}
	defer p.conn.Deallocate(context.Background(), stmt.Name) //nolint:errcheck

	for _, entity := range linkEntities {
		if _, err := tx.Exec(context.Background(), stmt.Name, entity.ID, entity.OriginalURL, entity.UID); err != nil {
			return err
		}
	}
	if err := tx.Commit(context.Background()); err != nil {
		return err
	}
	return nil
}

func (p *PgLinksRepository) Count() (int, error) {
	query := `select count(*) from shortener.links`
	var count int
	err := p.conn.QueryRow(context.Background(), query).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (p *PgLinksRepository) FindLinksByUID(uid string) ([]LinkEntity, error) {
	query := `select uid, original_url, link_id  from shortener.links where uid=$1`
	var result []LinkEntity
	rows, err := p.conn.Query(context.Background(), query, uid)
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
	return p.conn.Ping(ctx)
}

func (p *PgLinksRepository) Close() error {
	return p.conn.Close(context.Background())
}

func (p *PgLinksRepository) migrate() error {
	// TODO нужен отдельный пакет для миграций из sql файлов, но с гусем падают теты в PR по дедлайну доступности порта
	migration := `
		CREATE SCHEMA IF NOT EXISTS shortener;
		-- DROP SCHEMA shortener CASCADE ;
		-- CREATE SCHEMA shortener;
		SET SEARCH_PATH TO shortener;

		CREATE TABLE IF NOT EXISTS links(
  			id serial primary key,
  			link_id varchar,
  			original_url varchar,
  			uid varchar,
  			created_at TIMESTAMP
		);
		ALTER TABLE links ALTER COLUMN created_at SET DEFAULT now();
		CREATE UNIQUE INDEX IF NOT EXISTS original_url_idx ON links USING btree (original_url);
		`
	_, err := p.conn.Exec(context.Background(), migration)
	return err
}