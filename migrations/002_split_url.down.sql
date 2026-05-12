ALTER TABLE links ADD COLUMN really_long_url TEXT;

UPDATE links SET really_long_url = really_long_path || really_long_query;

ALTER TABLE links DROP COLUMN really_long_path;
ALTER TABLE links DROP COLUMN really_long_query;

ALTER TABLE links ALTER COLUMN really_long_url SET NOT NULL;
ALTER TABLE links ADD CONSTRAINT links_really_long_url_unique UNIQUE (really_long_url);
