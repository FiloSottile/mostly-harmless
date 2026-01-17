CREATE TABLE IF NOT EXISTS publications (
    repo TEXT NOT NULL, -- did
    rkey TEXT NOT NULL,
    record_json BLOB NOT NULL,
    PRIMARY KEY (repo, rkey)
) STRICT;

CREATE TABLE IF NOT EXISTS documents (
    repo TEXT NOT NULL, -- did
    rkey TEXT NOT NULL,
    publication_repo TEXT NOT NULL,
    publication_rkey TEXT NOT NULL,
    document_json BLOB NOT NULL,
    PRIMARY KEY (repo, rkey)
    -- No foreign key because we might observe documents before their publication
) STRICT;

CREATE INDEX IF NOT EXISTS idx_documents_publication
    ON documents (publication_repo, publication_rkey);
