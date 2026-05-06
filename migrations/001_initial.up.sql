CREATE OR REPLACE FUNCTION set_update_time()
RETURNS TRIGGER AS $$
BEGIN
    NEW.update_time = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE links (
    id          BIGSERIAL   PRIMARY KEY,
    original_url    TEXT        NOT NULL,
    really_long_url    TEXT        NOT NULL UNIQUE,
    create_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    update_time TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_update_time_before_update
    BEFORE UPDATE ON links
    FOR EACH ROW
    EXECUTE FUNCTION set_update_time();
