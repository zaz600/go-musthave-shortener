package repository

import (
	"context"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

type PgLinksRepository struct {
	conn           *pgx.Conn
	insertLinkStmt *pgconn.StatementDescription
}

func NewPgLinksRepository(ctx context.Context, databaseDSN string) (*PgLinksRepository, error) {
	conn, err := pgx.Connect(ctx, databaseDSN)
	if err != nil {
		return nil, err
	}
	query := `insert into shortener.links(link_id, original_url, uid) values($1, $2, $3)`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	repo := PgLinksRepository{
		conn: conn,
	}
	err = repo.migrate(ctx)
	if err != nil {
		return nil, err
	}
	stmt, err := conn.Prepare(ctx, "insert link", query)
	if err != nil {
		return nil, err
	}
	repo.insertLinkStmt = stmt
	return &repo, nil
}

// Get достает по linkID из БД информацию по сокращенной ссылке LinkEntity
func (p *PgLinksRepository) Get(ctx context.Context, linkID string) (LinkEntity, error) {
	query := `select uid, original_url, link_id  from shortener.links where link_id = $1`
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	var entity LinkEntity
	result := p.conn.QueryRow(ctx, query, linkID)
	err := result.Scan(&entity.UID, &entity.OriginalURL, &entity.ID)
	if err != nil {
		return LinkEntity{}, err
	}
	return entity, nil
}

// PutIfAbsent сохраняет в БД длинную ссылку, если такой там еще нет.
// Если длинная ссылка есть в БД, выбрасывает исключение LinkExistsError с идентификатором ее короткой ссылки.
func (p *PgLinksRepository) PutIfAbsent(ctx context.Context, linkEntity LinkEntity) (LinkEntity, error) {
	query := `
WITH new_link AS (
    INSERT INTO shortener.links(link_id, original_url, uid) VALUES ($1, $2, $3)
    ON CONFLICT(original_url) DO NOTHING
    RETURNING link_id
) SELECT COALESCE(
    (SELECT link_id FROM new_link),
    (SELECT link_id FROM shortener.links WHERE original_url = $2)
);`
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	var linkID string
	err := p.conn.QueryRow(ctx, query, linkEntity.ID, linkEntity.OriginalURL, linkEntity.UID).Scan(&linkID)
	if err != nil {
		return LinkEntity{}, err
	}
	if linkEntity.ID != linkID {
		// хотели положить в бд ссылку с одним коротким айди,
		// а вернулся айди ранее сохкращеной ссылки
		return LinkEntity{}, NewLinkExistsError(linkID)
	}
	return linkEntity, nil
}

// PutBatch сохраняет в БД список сокращенных ссылок. Все ссылки записываются в одной транзакции.
func (p *PgLinksRepository) PutBatch(ctx context.Context, linkEntities []LinkEntity) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	tx, err := p.conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	for _, entity := range linkEntities {
		if _, err = tx.Exec(ctx, p.insertLinkStmt.Name, entity.ID, entity.OriginalURL, entity.UID); err != nil {
			return err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

// Count возвращает количество записей в репозитории.
func (p *PgLinksRepository) Count(ctx context.Context) (int, error) {
	query := `select count(*) from shortener.links`
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	var count int
	err := p.conn.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// FindLinksByUID возвращает ссылки по идентификатору пользователя
func (p *PgLinksRepository) FindLinksByUID(ctx context.Context, uid string) ([]LinkEntity, error) {
	query := `select uid, original_url, link_id  from shortener.links where uid=$1 and removed = false`
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	var result []LinkEntity
	rows, err := p.conn.Query(ctx, query, uid)
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

// DeleteLinksByUID удаляет ссылки пользователя
func (p *PgLinksRepository) DeleteLinksByUID(ctx context.Context, uid string, ids []string) error {
	// TODO надо бить ids на чанки по 1024- штуки
	query := `delete from shortener.links where uid=$1 and link_id = any($2)`
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := p.conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, query, uid, ids)
	if err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

// Status статус подключения к хранилищу
func (p *PgLinksRepository) Status(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	return p.conn.Ping(ctx)
}

// Close закрывает, все, что надо закрыть
func (p *PgLinksRepository) Close(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	_ = p.conn.Deallocate(ctx, p.insertLinkStmt.Name)
	return p.conn.Close(ctx)
}

func (p *PgLinksRepository) migrate(ctx context.Context) error {
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
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := p.conn.Exec(ctx, migration)
	return err
}
