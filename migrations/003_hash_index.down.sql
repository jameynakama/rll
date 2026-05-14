ALTER TABLE links DROP CONSTRAINT links_path_hash_unique;
ALTER TABLE links DROP COLUMN path_hash;
ALTER TABLE links ADD CONSTRAINT links_really_long_path_unique UNIQUE (really_long_path);
