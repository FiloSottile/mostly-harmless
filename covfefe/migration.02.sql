BEGIN;

UPDATE Messages SET account = json_array(account) WHERE NOT json_valid(account);

COMMIT;
