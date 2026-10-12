CREATE TABLE blocked_domains (
    domain      TEXT        PRIMARY KEY,
    create_time TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Lowercased host of an absolute url, skipping any user@ prefix.
CREATE FUNCTION url_host(url TEXT) RETURNS TEXT
LANGUAGE sql IMMUTABLE AS $$
    SELECT lower(substring(url from '^[a-zA-Z][a-zA-Z0-9+.-]*://(?:[^/@?#]*@)?([^/:?#]+)'))
$$;

-- A blocked domain covers itself and its subdomains, not lookalikes:
-- bad-site.example matches www.bad-site.example but not notbad-site.example.
CREATE FUNCTION domain_matches(host TEXT, domain TEXT) RETURNS BOOLEAN
LANGUAGE sql IMMUTABLE AS $$
    SELECT host = domain OR right(host, length(domain) + 1) = '.' || domain
$$;
