
CREATE TABLE log_send (
    short_host_id SMALLINT NOT NULL,
    id BYTEA NOT NULL, /* a random 16-byte ID */
    uid BYTEA, /* the UID of the user sending the log, might be NULL if the user can't log in */
    ctime TIMESTAMP NOT NULL,
    PRIMARY KEY(short_host_id, id)
);
CREATE UNIQUE INDEX log_send_id_idx ON log_send(id);

CREATE TABLE log_send_files (
    short_host_id SMALLINT NOT NULL,
    ls_id BYTEA NOT NULL, /* a random 16-byte ID */
    file_id INTEGER NOT NULL, /* a random 16-byte ID */
    filename VARCHAR(255) NOT NULL, /* the name of the file */
    len INTEGER NOT NULL, /* the length of the file */
    nblocks INTEGER NOT NULL, /* the number of blocks in the file */
    hsh BYTEA NOT NULL, /* the SHA512/256 hash of the file */
    ctime TIMESTAMP NOT NULL,
    PRIMARY KEY(short_host_id, ls_id, file_id),
    FOREIGN KEY(short_host_id, ls_id) REFERENCES log_send(short_host_id, id)
);

CREATE TABLE log_send_blocks (
    short_host_id SMALLINT NOT NULL,
    ls_id BYTEA NOT NULL, /* a random 16-byte ID */
    file_id INTEGER NOT NULL, /* a random 16-byte ID */
    block_id INTEGER NOT NULL, /* the block ID within the file */
    block BYTEA NOT NULL, /* the block itself */
    PRIMARY KEY(short_host_id, ls_id, file_id, block_id),
    FOREIGN KEY(short_host_id, ls_id) REFERENCES log_send(short_host_id, id),
    FOREIGN KEY(short_host_id, ls_id, file_id) REFERENCES log_send_files(short_host_id, ls_id, file_id)
);