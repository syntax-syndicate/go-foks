
CREATE TABLE merkle_roots (
    short_host_id SMALLINT NOT NULL,
    epno INT NOT NULL,
    ctime TIMESTAMP NOT NULL,
    hash BYTEA NOT NULL,
    body BYTEA NOT NULL,
    root_node BYTEA NOT NULL,
    sig BYTEA,
    PRIMARY KEY(short_host_id, epno)
);
CREATE INDEX merkle_roots_hash_idx ON merkle_roots (hash);
CREATE INDEX merkle_roots_to_sign_idx ON merkle_roots (short_host_id, epno) WHERE sig IS NULL;

CREATE TABLE merkle_nodes (
    hash BYTEA NOT NULL PRIMARY KEY,
    bit_start SMALLINT NOT NULL,
    bit_count SMALLINT NOT NULL,
    key_segment BYTEA NOT NULL,
    l BYTEA,
    r BYTEA
);

/* 
 * Rather than index on whether a root is signed or not, we just write the last sig here,
 * will be way more efficient.
 */
CREATE TABLE merkle_last_sig (
    short_host_id SMALLINT NOT NULL,
    epno INT NOT NULL,
    mtime TIMESTAMP NOT NULL,
    PRIMARY KEY(short_host_id)
);

/* Sigh, I guess it should be possible to sign 1-100 with key A, then 101-200 with key B,
 * then 201+ with key A again. Maybe due to a botched upgrade? Anyways, allow this in the 
 * schema plan.
 */
CREATE TABLE merkle_signing_keys (
    short_host_id SMALLINT NOT NULL,
    key_id BYTEA NOT NULL,
    epno INT NOT NULL,
    ctime TIMESTAMP NOT NULL,
    PRIMARY KEY(short_host_id, key_id, epno)
);

CREATE UNIQUE INDEX merkle_signing_keys_epno_idx ON merkle_signing_keys (short_host_id, epno);

CREATE TABLE merkle_leaves (
   hash BYTEA NOT NULL PRIMARY KEY,
   k BYTEA NOT NULL,
   v BYTEA NOT NULL,
   epno INT NOT NULL /* when it was inserted into the tree */
);

CREATE TABLE merkle_bookkeeping (
    short_host_id SMALLINT NOT NULL PRIMARY KEY,
    build_next_batchno INT NOT NULL, /* written by the builder, next batchno to build on */
    batch_next_batchno INT NOT NULL, /* written by the batcher, next batchno to batch */
    pos INT NOT NULL, /* only used by the builder, next position in the batch to build at */
    mtime TIMESTAMP NOT NULL
);

CREATE INDEX merkle_bookkeeping_host_idx ON merkle_bookkeeping(short_host_id) WHERE (build_next_batchno < batch_next_batchno);

CREATE TABLE merkle_hostchain_tails (
    short_host_id SMALLINT NOT NULL,
    seqno INT NOT NULL,
    hash BYTEA NOT NULL,
    PRIMARY KEY(short_host_id, seqno)
);

/*
 * per-tree metadata, one row per tree. Might add more fields in the future.
 */
CREATE TABLE merkle_tree_metadata (
    short_host_id SMALLINT NOT NULL PRIMARY KEY,
    first_node BOOLEAN NOT NULL
);

CREATE UNIQUE INDEX merkle_leaves_key_idx ON merkle_leaves(k);