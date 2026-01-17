-- name: StorePublication :exec
INSERT INTO publications (repo, rkey, record_json)
VALUES (?, ?, ?)
ON CONFLICT(repo, rkey) DO UPDATE SET record_json=excluded.record_json;

-- name: DeletePublication :exec
DELETE FROM publications
WHERE repo = ? AND rkey = ?;

-- name: StoreDocument :exec
INSERT INTO documents (repo, rkey, publication_repo, publication_rkey, document_json)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(repo, rkey) DO UPDATE SET document_json=excluded.document_json, publication_repo=excluded.publication_repo, publication_rkey=excluded.publication_rkey;

-- name: DeleteDocument :exec
DELETE FROM documents
WHERE repo = ? AND rkey = ?;

-- name: GetPublications :many
SELECT record_json, rkey
FROM publications
WHERE repo = ?
ORDER BY rowid DESC;

-- name: GetPublication :one
SELECT record_json
FROM publications
WHERE repo = ? AND rkey = ?;

-- name: GetDocumentsForPublication :many
SELECT document_json, rkey
FROM documents
WHERE publication_repo = ? AND publication_rkey = ?
AND repo = publication_repo -- don't let strangers inject documents into others' publications
ORDER BY rowid DESC;
