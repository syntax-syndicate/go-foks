# Intro: FOKS KV-Store

Foks has a build-in Key-Value storage mechanism. Team or users (which recall are called "parties")
can store key/values namespaced by partyID. The keys occupy a file-system like hierarchy, like
`/a/boo/foo/yodo.txt`. There are no real limitations on how many entries can be put into
a directory, so the infrastructure must be able to scale appropriately. Keys and values
are encrypted with PTKs or PUKs. Therefore, they should rotate whenever the PTKs or PUKs rotate,
but there is an unvavoidable limitation here. The server can always choose to remember old
encyrptions and therefore, is always in a position to reconstruct files with recovered
secret device keys. Thus, we assume servers behave semi-honestly, meaning they overwrite and throw
away data appropriately. If this assumption is false, *and* the server gains possession of a
revoked device key, it can decrypt server-side data. But we don't really see a way around this.

In terms of API support, we wanted to allow efficient renames: either within a directory, or 
across a directoy. And directories themselves ought to support cheap renames. To achieve
this goal, but the server and the client need to be aware of the namespace hierarchy. This leaks
a little information to the server as opposed to a KBFS-style architecture, but most of that
information could have been inferred in the KBFS case by timing and blocksize observations.
That being said, the goal is to hide file names and file contents from the server, and in
both cases, to guard file integrity.

## What We Cover in this Article

- API Supported
- Permissions? (Read level = floor, write level might be higher, server enforces write level, crypto enforces read level)
- Storage Strategy (server)
- Dirents
- Files
- Merkle Sub-trees
- Rotations (cheap and expensive)

# API Supported

Here is a rough description of the protocol-level API that is supported:

- Mkfs(PartyID) -> fsid
- Mkdir(fsid, ReadRole, WriteRole) -> dirID
- SetRoot(fsid, dirID)
- GetRoot(fsid) -> dirID
- PutFile(fsid, data, ReadRole) -> fileID
- GetFile(fsid, fileID) -> data
- PutSoftLink(fsid, path, target, ReadRole) -> linkID
- Put(fsid, path, (dirID | fileID | linkID ), Role)
- Get(fsid, path) -> (dirID | fileID | linkID)
- Ls(fsid, path, start, limit) -> List(dirent)
- ChmodWrite(fsid, path, Role)
- Rm(fsid, path)
- RmFile(fsid, fileID) 

These primitive APIs can be combined to achieve a higher-level API with commands like:

- Put(fsid, path, data, ReadRole)
- Get(fsid, path, ReadRole)
- Rm(fsid, path)
- Mkdir(fsid, path, ReadRole, WriteRole)
- Rmdir(fsid, path) (unix semantics, directory needs to be empty)
- Mv(fsid, src, dst)
- Hardlink(fsid, src, dst) (we don't allow hardlinks to directories)
- SoftLink(fsid, src, dst)
- Ls(fsid, path) -> List(path)

# Storage Stretegy (Server-side)

The server-side storage strategy combines use of SQL databases (BTrees) and flat
KV-stores (like S3). The SQL database are:

### fs table (one row for each file system)
- fsID PRIMARY KEY
- partyID
- root dirID

### dir table (1-2 rows for each directory)

#### Schema
- fsID
- dirID
- version
- ptkGen 
- readRole (detertmines the role of the key used in the below box)
- seedBox (32-byte secret seed boxed for PTK at <readRole, ptkGen>)
- writeRole (who can add entries to or remove entries from this dir)
- PRIMARY KEY(fsID, dirID, version)

In general, the largest ptkGen is the active ptk for this directory.
However, for big directories in the midst of a key rotation operation,
there could be 2 valid seeds in operation for this directory. In this case,
Get and Put operations have to send up 2 possible encryptions for every name
they try to access.

One the rotation process completes, the older rows are deleted.

Note we are not required to rotate the dir key on every ptkGen. So there could
be gaps in the ptkGen sequence, though there are none in the version sequence.

### dirents table (one row for each dirent across the whole FS)

#### Schema

- fsID
- dirID
- dirVersion (= dir.version)
- nameBox (convergently-encrypted with key in dirKeySeedBox)
- nameNonce (MACed with the key in dirKeySeedBox)
- value (a dirID, fileID or linkID; 17-byte identifier, where the 0ths byte is the type)
- writeRole (who can overwrite this entry)
- version (starts at 0 and increase monotonically with each change)
- mac (hmac of <fsID, dirID, dirVersion, nameNonce, value, writeRole, version> mac'ed with MAC key derived from dirKeySeedBox).
- PRIMARY KEY(fsID, dirID, dirVersion, nameNonce)

As above, there could be 2 rows for each underlying dirent if the directory is
in the middle of a key rotation.

### large files table 

#### Schema

- fsID
- fileID
- version (bumps whenever rotated)
- size
- readRole 
- ptkGen
- keyBox
- refcount
- rcTime
- PRIMARY KEY(fsID, fileID)
- INDEX(rcTime) where refcount=0 ??

File data is stored in a flat KV-store, and the fileID is the key. Once files are written,
their contents are immutable, though their encryption can be updated.

### small files table (<128 bytes)

#### Schema

- fsID
- fileID (== random nonce, see below).
- ptkGen
- readRole
- size
- box 
- refcount
- rcTime
- PRIMARY KEY(fsID, fileID)
- INDEX(rcTime) where refcount=0 ??

# Permissions

Read permissions are deterined at the directory level by dir.readRole, and at
the file level by then PTK role of the boxer. In both cases, the readers are
those who can access the private keys associated with that team (or device)
role. Write permissions are determined at the directly level, and the dirent
level, which use server-enforced ACLs. The directory writeRole controls
who can delete or create dirents in that directory. The writeRole on the 
dirent level controls who can edit where that name points to.

Files don't have associated write permissions since their data is immutable after
initial write.

# Crypto

## Directories

### Initialization

When a new directory is created, the process is:
- Pick a random 16-byte directory ID (written to the DB as a 17-byte BYTEA with a 0x40 prefix). Call it i
- Pick a random 32-byte crypto seed s
  - Derive r_box = HMAC(s, 0x5)
  - Derive r_mac = HMAC(s, 0x4)
  - Derive r_comm = HMAC(s, 0x6)
- Compute SecretBox(key=PTK, plaintext=s, nonce=i), for PTK at the requested readRole, at the latest
  generation.
- Write i, the secret box, readRole, writeRole, ptkGen to the directory

### Dirents

#### Computation

For a name n in directory i, with seed s, and r as above:

- Compute nonce=HMAC(r_mac, i||n)
- Compute SecretBox(
    key=r_box,
    plaintext=n,
    nonce=nonce,
)
- Write the secret box and the nonce into the dirents table along with the rest of the dirent

#### Comments

The encryption here is "convergent encryption", meaning different parties who attempt this 
encryption will get the same value. Note that the nonce doesn't allow a "guess and chaeck"
attack, since it depends on the secret key. 

Note also that the DB row for a dirent is HMAC'ed by the corresponding directory keys.
This prevents the server from swapping in a file from somewhere else that wasn't intended.
This does *not* prevent the server from rolling back to a previous version. Commitment
to a merkle tree will be needed for that.

#### Rotation

In rotations, we generate a new seed, and encrypt it for the team's most recent PTK.
Everything encrypted or MAC'ed with keys derived from the old seed must be re-encrypted
and re-MACed.

For small directories, it seems fine to lock the directory and re-encrypt
everything in one fell swoop. But for large directories, it probably makes sense
to support 2 co-existing seeds. During this time, there isn't a good way to
preserve the convergent encryption property. Hence this protocol:

1. Pick directory d
2. Roll a new seed for the latest PTK
3. Insert a row into the dir table, now there are two active rows for this dirID
4. Iterate over all dirents in the directory:
   - insert row for new enryption of name with the new directory key material. This
     encryption won't match the encryption under the previous key.
   - increment the version of the row.
5. Delete the row of the old key from the dir table
6. Iterate over all dirents in the directory:
   - delete the old row

## Files

### Large Files

#### Computation

- Given a large file f (>128 bytes), a PTK at role r and generation g:
  - Pick a random 16-byte file ID, i
  - Pick a random 32-byte seed, s
  - As above, derive r_box, r_mac, r_comm (via HMAC)
  - Compute KeyBox = SecretBox(
        key=PTK,
        plaintext=s,
        nonce=i
      )
  - Compute DataBox = SecretBox(
        key=r_box,
        plaintext=f,
        nonce=i
      )
  - Output is i || KeyBox || DataBox. Can store all of this in S3, or maybe just DataBox, depending.

#### Comments

Encryption is randomized, so 2 different attempts to encrypt the same file will yield different
encryptions. The downside is that this might result in extra stroage if the same big file appears
in mulitple places across the FS. The client can locally cache these files though and optionally
get the same effect.

Note that Copilot recommends that nonce=HMAC(r_mac, i) is used. I don't think
this is wrong, but I can't see a good reason to do that over the simple nonce=i.
Even nonce=i might not be necessary, since r_box only has one usage, and is
bound to i in the KeyBox.

For a security analysis, first assume that the server does not know the PTKs in
question. Then, the server cannot generate a malicious KeyBox, as the encryption
is authenticated. A malicious server might try to maliciously mix valid KeyBox
and DataBoxes, but the nonce i is generated once per file, and prevents the
mix-and-match.

Without decrypting the KeyBox, a malicious server can't decrypt the value of s, and
therefore cannot derive r_box. Therefore, it cannot generate new valid new encryptions
of f, or decrypt it, since the decryptor will only attempt to decrypt with r_box
(derived from the s it got above).

For reasoning around rotations, see below.

#### Rotations

The general idea of rotation here is to rotate the KeyBox, which is small, and to
leave the DataBox as is, since updating it might involve significant bandwidth
consumption (and time).

The rotation process is to retain i and r, but to rebox the KeyBox for a new
PTK. Why bother one might ask? A malicious server could use the old KeyBox, along with 
the compromised old PTK, to recover r and to decrypt the file in the databox. This is
true. In fact, no amount of reencryption will save a user in the worst case, since we
can simply assume that a malicious server retained all boxes, and has the chain of 
keys necessary to perform any decryption rooted in the compromised key.

A more interesting question is to ask what happens if the server is honest at
the time of the rotation, but becomes compromised later. In this line of
reasoning, the DataBox would be compromised if not reencrypted, as it's
encrypted with the old PTK.  It suffices to keep the old i and old r, but just
reencrypt the databox with the new PTK, assuming the serer can be trusted to
throw away the old KeyBox.

### Small Files

#### Computation

- Given a file f, and a PTK at role r, generation g:
  - pick random 16-byte nonce 
  - output p || r || g || nonce || secretBox(PTK, nonce, f) 
- Overhead is 16-bytes for nonce, 16 bytes for tag + 6 or so assorted bytes
- The nonce can be reused as the file ID, to save space.

#### Comments

The encryption scheme is straight-forward. Again, encryption is randomized since
we have random nonces. We have no intermediate key as we do in large-file
encryption, since there aren't appreciable bandwidth savings to be gained on
rotation.

# Merkle Sub-trees

## Motivation

There are several things that can be rolled back in this system in a server compromise:

 - Entries in the dir/ table; clients are not obligated to re-roll dirkeys when they rotate
PTKs, so the rotations can happen at some arbitrary point in the future. This brings up the problem
that the server, if temporarily compromised, might roll back these dirkeys so that new dirents
written down use an old (potentially compromised) version.
 - Entries in the dirent/ table, which can be updated to point to files other than those intended.
 - Entries in the large files tables, since keyBoxes can be rotated.
 - Entries in the small files table, since small files are directly reencrypted with updated PTKs.

The idea is to put a representation of all 4 types of rows (dirs, dirents, large files, small files)
into the Merkle Tree. An immediate issue we'll see is that the number of writes here are O(N) times
of that of the main tree, where N is the number of files the team is storing. To see this, just
observe that each PTK rotation forces a rotation, eventually, of all associated dirs, dirents,
and files.

## Scaling

To mitigate this explosion of Merkle writes, we introduce Merkle Sub-trees. A
Merkle Sub-Tree is a Merkle Tree that functions largely as the main merkle tree,
except it's scoped for a particular party fsID. Its root is periodically written
back to the main subtree, allowing many batched updates for each main Merkle
tree update. In terms of deployment, subtrees can be easily parallelized on
standalone databases and applicatino servers.  Because the ordering properties
aren't as strict as they are with the main tree, we can get away with batching
updates.

One simplification we have with subtrees is that tree locations in the subtree 
don't need to be randomized, since no one outside of the group can see the tree.
However, we should continue to randomize the inclusion of the subtree root
in the main tree.

## Specifics

### Dir

- Key:
  - dirID
  - table: "dir"
  - version
- Value:
  - ptkGen
  - readRole
  - writeRole

### Dirent:

- Key:
  - dirID
  - dirVersion
  - nameNonce
- Value:
  - version
  - value
  - writeRole

Note that dirents are still written even if a file is "tombstoned",
i.e., deleted. The value for a deleted file is 

### Small Files:

There is no need to store small files in the tree, since they are immutable once
written. So rollbacks, or confusions around versions, aren't a consideration.

### Large Files:

- Key:
   - fileID
   - version
- Value:
   - ptkGen

## Inclusion in Main Tree

The subtree is included back into the main tree with an incrementing counter and a
location randomizer.

- Key:
  - fsID
  - epno
  - location randomizer i (0 for epno=0)
- Value:
  - subtreeRoot
  - location randomizer i+1

# Garbage collection

Garbage colletion on small and large files is done with simple refcounting. We increment
refcounts when we link a file into a directory, and decrement when we unlink. There is a
grace period of about 30 days before deleting something for real.

# Credits and Quota

- Anyone can create an FS
- Need to allocate it credits
  - Single credit for a dirent or a small file
  - 1 additional credit for every 4k of a large file
  - 10 for making a new FS

# Audit

TBD, but important. Roughly, the server will write down access to dirents as they happen.
In an audit, it will ship them back to the client, who can decrypt them, to see who
accessed what. The one thing I'm quite confused about is what to do about the case
in which Bob is a member of a team X, which is a member of a team Y, and Y's KV-data
is acceessed by X. It would be nice to show that at the end of the chain, it was Bob doing
the access, and not X, but that will take some machinery that we currently don't have.
