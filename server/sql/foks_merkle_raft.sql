
CREATE TABLE raft_kv_store (
    short_host_id SMALLINT NOT NULL,
    k VARCHAR(255) NOT NULL,
    v BYTEA,
    PRIMARY KEY(short_host_id, k)
);