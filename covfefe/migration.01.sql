PRAGMA foreign_keys=OFF;
BEGIN;

CREATE TABLE IF NOT EXISTS new_Messages (
    id INTEGER PRIMARY KEY,
    received DATETIME DEFAULT (DATETIME('now')),
    json TEXT NOT NULL,
    account TEXT NOT NULL
);
INSERT INTO new_Messages SELECT id, created, json, "FiloSottile" FROM Messages;
DROP TABLE Messages;
ALTER TABLE new_Messages RENAME TO Messages;
PRAGMA foreign_key_check;

CREATE TABLE IF NOT EXISTS new_Tweets (
    id INTEGER PRIMARY KEY,
    created DATETIME NOT NULL,
    user TEXT NOT NULL,
    message INTEGER NOT NULL REFERENCES Messages(id),
    deleted INTEGER REFERENCES Messages(id)
);
INSERT INTO Messages (received, json, account)
    SELECT deleted, json_object(
        'id', id, 'id_str', CAST(id AS TEXT), 'user_id', NULL, 'user_id_str', NULL, 'synthetic', 1
    ), "FiloSottile" FROM Tweets WHERE deleted IS NOT NULL;
INSERT INTO new_Tweets SELECT id, created, user, message, (
    SELECT id FROM Messages WHERE Tweets.deleted IS NOT NULL
    AND json_extract(json, '$.synthetic') AND json_extract(json, '$.id') = Tweets.id
) FROM Tweets;
DROP TABLE Tweets;
ALTER TABLE new_Tweets RENAME TO Tweets;
PRAGMA foreign_key_check;

COMMIT;
PRAGMA foreign_keys=ON;
