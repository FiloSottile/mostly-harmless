BEGIN;

ALTER TABLE Messages ADD COLUMN kind TEXT;

UPDATE Messages SET kind = "tweet"
WHERE json_extract(json, '$.retweet_count') IS NOT NULL;

UPDATE Messages SET kind = "event" WHERE kind IS NULL
AND json_extract(json, '$.event') IS NOT NULL;

UPDATE Messages SET kind = "del" WHERE kind IS NULL
AND json_extract(json, '$.synthetic') IS NOT NULL;
UPDATE Messages SET kind = "del" WHERE kind IS NULL
AND json_extract(json, '$.user_id_str') IS NOT NULL;

UPDATE Messages SET kind = "deletion" WHERE kind IS NULL
AND json_extract(json, '$.delete') IS NOT NULL;

SELECT kind, COUNT(*) FROM Messages GROUP BY kind;

COMMIT;
