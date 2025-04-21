
CREATE table hosts (
    host_id BYTEA NOT NULL PRIMARY KEY,
    tail BYTEA NOT NULL,
    seqno INTEGER NOT NULL,
    hostname VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL, 
    ctime TIMESTAMP NOT NULL,
    mtime TIMESTAMP NOT NULL
);
