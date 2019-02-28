BEGIN;

CREATE TABLE new_Messages (
    id INTEGER PRIMARY KEY,
    received DATETIME DEFAULT (DATETIME('now')),
    json TEXT NOT NULL,
    source TEXT NOT NULL -- JSON array of source IDs
);
INSERT INTO new_Messages SELECT id, received, json, 
    replace(replace(replace(account, '[', '["tl:'), ',', '","tl:'), ']', '"]')
FROM Messages;
DROP TABLE Messages;
ALTER TABLE new_Messages RENAME TO Messages;

COMMIT;
VACUUM;
