ALTER TABLE links ADD COLUMN really_long_path TEXT;
ALTER TABLE links ADD COLUMN really_long_query TEXT;

UPDATE links SET
    really_long_path = CASE
        WHEN position('?' in really_long_url) > 0
        THEN substring(really_long_url from 1 for position('?' in really_long_url) - 1)
        ELSE really_long_url
    END,
    really_long_query = CASE
        WHEN position('?' in really_long_url) > 0
        THEN substring(really_long_url from position('?' in really_long_url))
        ELSE ''
    END;

ALTER TABLE links ALTER COLUMN really_long_path SET NOT NULL;
ALTER TABLE links ALTER COLUMN really_long_query SET NOT NULL;
ALTER TABLE links ADD CONSTRAINT links_really_long_path_unique UNIQUE (really_long_path);

ALTER TABLE links DROP COLUMN really_long_url;
