BEGIN;

CREATE TABLE Users (
    id INTEGER NOT NULL,
    handle TEXT NOT NULL,
    name TEXT NOT NULL,
    bio TEXT NOT NULL,
    first_seen INTEGER NOT NULL REFERENCES Messages(id),
    UNIQUE (id, handle, name, bio) ON CONFLICT IGNORE
);
CREATE TABLE Follows (
    follower INTEGER NOT NULL,
    target INTEGER NOT NULL,
    first_seen INTEGER NOT NULL REFERENCES Messages(id),
    UNIQUE (target, follower) ON CONFLICT IGNORE
);

DELETE FROM Tweets;

CREATE TABLE new_Messages (
    id INTEGER PRIMARY KEY,
    received DATETIME DEFAULT (DATETIME('now')),
    json TEXT NOT NULL,
    account TEXT NOT NULL -- JSON array of IDs
);
INSERT INTO new_Messages SELECT id, received, json, 
    replace(replace(replace(replace(account, '"FiloSottile"', 51049452), '"Benjojo12"', 40015387),
    '"kgibilterra"', 27384801), '"AnnaOpss"', 503152671) FROM Messages;
DROP TABLE Messages;
ALTER TABLE new_Messages RENAME TO Messages;

COMMIT;
