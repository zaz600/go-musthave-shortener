package repository

import (
	"context"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/zaz600/go-musthave-shortener/internal/entity"
)

type PgLinksRepository struct {
	conn           *pgx.Conn
	insertLinkStmt *pgconn.StatementDescription
	removeLinkStmt *pgconn.StatementDescription
}

func NewPgLinksRepository(ctx context.Context, databaseDSN string) (*PgLinksRepository, error) {
	conn, err := pgx.Connect(ctx, databaseDSN)
	if err != nil {
		return nil, err
	}
	repo := PgLinksRepository{
		conn: conn,
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = repo.migrate(ctx)
	if err != nil {
		return nil, err
	}

	queryInsert := `insert into shortener.links(link_id, original_url, uid) values($1, $2, $3)`
	stmtInsert, err := conn.Prepare(ctx, "insert link", queryInsert)
	if err != nil {
		return nil, err
	}
	repo.insertLinkStmt = stmtInsert

	queryRemove := `update shortener.links set removed=true where uid=$1 and link_id = any($2)`
	stmtRemove, err := conn.Prepare(ctx, "remove links", queryRemove)
	if err != nil {
		return nil, err
	}
	repo.removeLinkStmt = stmtRemove

	return &repo, nil
}

// Get достает по linkID из БД информацию по сокращенной ссылке entity.LinkEntity
func (p *PgLinksRepository) Get(ctx context.Context, linkID string) (*entity.LinkEntity, error) {
	query := `select uid, original_url, link_id, removed  from shortener.links where link_id = $1`
	var e entity.LinkEntity
	result := p.conn.QueryRow(ctx, query, linkID)
	err := result.Scan(&e.UID, &e.OriginalURL, &e.ID, &e.Removed)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// PutIfAbsent сохраняет в БД длинную ссылку, если такой там еще нет.
// Если длинная ссылка есть в БД, выбрасывает исключение LinkExistsError с идентификатором ее короткой ссылки.
func (p *PgLinksRepository) PutIfAbsent(ctx context.Context, linkEntity entity.LinkEntity) (entity.LinkEntity, error) {
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
	err := p.conn.QueryRow(ctx, query, linkEntity.ID, linkEntity.OriginalURL, linkEntity.UID).Scan(&linkID)
	if err != nil {
		return entity.LinkEntity{}, err
	}
	if linkEntity.ID != linkID {
		// хотели положить в бд ссылку с одним коротким айди,
		// а вернулся айди ранее сохкращеной ссылки
		return entity.LinkEntity{}, NewLinkExistsError(linkID)
	}
	return linkEntity, nil
}

// PutBatch сохраняет в БД список сокращенных ссылок. Все ссылки записываются в одной транзакции.
func (p *PgLinksRepository) PutBatch(ctx context.Context, linkEntities []entity.LinkEntity) error {
	tx, err := p.conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	for _, e := range linkEntities {
		if _, err = tx.Exec(ctx, p.insertLinkStmt.Name, e.ID, e.OriginalURL, e.UID); err != nil {
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
	var count int
	err := p.conn.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// FindLinksByUID возвращает ссылки по идентификатору пользователя
func (p *PgLinksRepository) FindLinksByUID(ctx context.Context, uid string) ([]entity.LinkEntity, error) {
	query := `select uid, original_url, link_id  from shortener.links where uid=$1 and removed = false`

	var result []entity.LinkEntity
	rows, err := p.conn.Query(ctx, query, uid)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var e entity.LinkEntity
		err = rows.Scan(&e.UID, &e.OriginalURL, &e.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteLinksByUID удаляет ссылки пользователя
func (p *PgLinksRepository) DeleteLinksByUID(ctx context.Context, uid string, linkIDs ...string) error {
	// TODO надо бить ids на чанки по 1024- штуки
	tx, err := p.conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, p.removeLinkStmt.Name, uid, linkIDs)
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
	return p.conn.Ping(ctx)
}

// Close закрывает, все, что надо закрыть
func (p *PgLinksRepository) Close(ctx context.Context) error {
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
  			created_at TIMESTAMP,
			removed boolean
		);
		ALTER TABLE links ALTER COLUMN created_at SET DEFAULT now();
		ALTER TABLE links ALTER COLUMN removed SET DEFAULT false;
		CREATE UNIQUE INDEX IF NOT EXISTS original_url_idx ON links USING btree (original_url);
		`
	_, err := p.conn.Exec(ctx, migration)
	return err
}
