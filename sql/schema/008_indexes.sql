-- +goose Up

CREATE INDEX posts_feed_id_published_at_idx ON posts (feed_id, published_at DESC);
CREATE INDEX feeds_last_fetched_at_idx ON feeds (last_fetched_at);

-- +goose Down

DROP INDEX feeds_last_fetched_at_idx;
DROP INDEX posts_feed_id_published_at_idx;
