ALTER TABLE links DROP CONSTRAINT links_really_long_path_unique;
ALTER TABLE links ADD COLUMN path_hash TEXT NOT NULL DEFAULT '';
UPDATE links SET path_hash = md5(really_long_path);
ALTER TABLE links ALTER COLUMN path_hash DROP DEFAULT;
ALTER TABLE links ADD CONSTRAINT links_path_hash_unique UNIQUE (path_hash);
