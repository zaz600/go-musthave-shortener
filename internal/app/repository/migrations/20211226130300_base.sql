-- +goose Up
CREATE SCHEMA IF NOT EXISTS shortener;
-- DROP SCHEMA shortener CASCADE ;
-- CREATE SCHEMA shortener;
SET SEARCH_PATH TO shortener;

CREATE TABLE IF NOT EXISTS links
(
    id           serial primary key,
    link_id      varchar,
    original_url varchar,
    uid          varchar,
    created_at   TIMESTAMP
);
ALTER TABLE links
    ALTER COLUMN created_at SET DEFAULT now();
CREATE UNIQUE INDEX original_url_idx ON links USING btree (original_url);

