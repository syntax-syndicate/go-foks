# Revocation Device Locking in FOKS

## Overview

The goal here is that revokes of a key and usages of that
key won't "cross in the mail", so that when k is revoked, it
contains a hash containing all links it ever signed in its
lifetime. This way verifiers can verify that the signatures
happened before the revoke. 

If we didn't do this, we might see a link signed by device k on
a sidechain, which shows up in the merkle tree at epoch 1020, but
k being revoked, and epoch 1000 being signed into the link. There is
now no way to make sure the revoke happened after the signature
on the side chain.

## Example

Here's an example of what we'd like to avoid. Imagine two racing
threads, one using key K, and the other revoking key K. 

User U is signing a link with key K. The link points to Merkle Epno=10,
and it eventually gets signed into the tree at Merkle Epno=20:

* User U
  * Sign link with K
  * Header -> Merkle=10
  * <- Merkle=20

User R is revoking K:

* User R
  * Revoke key K with key J
  * Header -> Merkle=12
  * <- Merkle=24

Note that user U's request hits the database before user R's, so it will
pass simple checks that user K isn't revoked before it's use.  The
problem is that the Header pointing to Merkle=12 is a tree that does not
contain user U's link, since that doesn't happen until later (Merkle=20).

Of course R's transtion should fail since it should see that there are some
links in the Merkle work queue that have not yet been signed into the tree.
That's the simple idea.

## Specifics

Device locks involve 2 types of players (revokers (R) and users (U))
and two DB tables (revoke_key_locks and merkle_wurk_queue).

The general idea is to fake a "shared lock" using SQL.  Many Us can
operate in parallel, but only one R can operate at a time, and it must
be to the exclusion of all Us.

For revokers, the pattern is roughly:

    BEGIN (α) 
        DELETE FROM device_keys WHERE verify_key=K       (α0)
        INSERT INTO revoke_key_locks VALUES(K)           (α1)
        SELECT FROM merkle_work_queue WHERE signer=K     (α2)
    END

See InsertLink in server/shared/chain.go for most of the details on α.

For users, the pattern is roughly, on insert:

    BEGIN (β)
        SELECT FROM device_keys WHERE verify_key=K       (β0)   (note: NOT select for UPDATE)
        INSERT INTO revoke_key_locks VALUES(K)           (β1)
        INSERT INTO merkle_work_queue (signer) VALUES(K) (β2)
        DELETE FROM revoke_key_locks WHERE verify_key=K  (β3)
    END

For users, on commit of link into Merkle Tree:

    BEGIN (γ)
        UPDATE merkle_work_queue WHERE verify_key=K      (γ1) (via checkTreeLeafForUpdates)
    END

We can think about why this (hopefully) works by playing with the ordering of α and β
(with γ being roughly orthogonal):

### α1 < β1

Imagine this interleaving:

    β0 < α0 < α1 < β1

It matters most that α1 < β1, since the logic is the same for β0 < α0  or α0 <
β0. In either case, the revoke_key_locks table will written first by R.
Statement α1 will be written, and it will cause β1 to block, waiting for the
lock. The first subcase is the select in α2 returning nothing in flight. Then
transaction α commits, and the lock row is left in the table. Then β1 unblocks,
but with a failure, since it will fail the primary key constraint (duplicate
insert). This fails the whole transaction, and using the device doesn't work.
This is the correct behavior, since as far as U is concerned, the device is
revoked, and can't be used to make any further signatures.

###  β1 < α1

Imagine interleavings like:

    α0 < β0 < β1 < β2 < β3 < α1 < α2

Or: 

    β0 < α0 < β1 < β2 < β3 < α1 < α2

As before, the cases where α0 < β0 < β1 and β0 < α0 < β1 are the same.  The
important thing is that β1 < α1 and therefore β2 < α2. The select in α2 returns
still outstanding links. Let's look further at the SELECT in α2, which is this:

```sql
		SELECT state, COALESCE(epno, -1) FROM merkle_work_queue
		WHERE short_host_id=$1 AND id=$2 AND signer=$3
```

The transaction in R loops through all work_queue items, looking for any that
are not marked committed, or that were commited after the epoch in the revoke
link. The existence of any such link causes the revoke to fail, because the
revoke link is too old, and might not cover new signatures.  In this subcase,
the α transaction fails, and the row in the locks table is not committed.

Then looking at transaction β, the insert in statement β1 suceeds, since the
statement in α1 never commited. U is free to go through with its update,
as are subsequent updates, since the lock is removed at statement β3.

The transaction in R will retry and will continue to fail until
the thread in merkle_batcher runs γ, which will allow the select
in step α2 to succeed.

## Device Reuse

If you see the test TestRevokeAndReuse, it should be possible to reuse a verify
key, as does happy with YubiKeys, since we can't regenerate those keys.  But
also note that in the revoke workflow above (α), we never delete the row from
the revoke_key_locks. This would break TestRevokeAndReuse, which it did. But of 
course we solve the problem by further indexing revoke_keys_lock on seqno, which
is the sequence number in the user sigchain that this key is introduced. It will
be different for the second reuse of the same key, so these two locks take up
different rows. We had to plumb Seqno throughout the whole callchain, but it
does seem to work.

Note that for teams, this seqno is always 0, since we didn't bother to plumb it,
but also, for teams, there is no reuse of PTK Verify keys, so we should be able
to get away with this.

## Self-Revokes

Self revokes are slightly different since the same thread is both a user and revoker.
We have to convince ourselves that one thread (which is both a user a revoker)
can't race with another thread that is simply a key user. We need to make sure
that the self-revoke will fail if there are still signatures by the key inflight,
but that it can succeed if all signatures are reflected in the Merkle Tree. 
The code in InsertLink has slight ordering modifications here, but not that change
the logic of the above arugments.