# Team Membership Chains

A user needs to keep track of all teams they are in. The user shouldn't
have to fully trust the server to do this, because the server might want to
prevent the user from rotating a given team. Of course if the server is truly
evil, always, then the user might be out of luck. But we want to enforce that
if the server is usually honest. the user can recreate the correct memberships
they have.

In general, this is a slightly complicated protocol since the admin of
the team signing the user into the team doesn't have the permissions to
write into the user's team membership chain. That has to happen after the
fact. Similarly, when a user is removed from a tea, they lose the privilege
to load that team. The server has to (politely) reject the user from 
further loads, while giving the user the hint that he's out of the team.
But the server ought not able to do this without the team admin's consent,
since an evil server could use this flow to DoS the user (as discussed above).

So the simplest thing just became tricky!

One more important thing to keep in mind: both users and team have team membership
chains because teams can be members of other teams. Most of what we are doing here
should generalize across both instantiations of what a "party" is. (A "party" is
a team or a user, i.e., something that can be a member of a team.)

## Adding Membership

Let's say team T has admin A, and wants to add user X to the team. The exact
join protocol is covered in other documents (see team_invites.md). In addition
to all the usual on-chain and off-chain activitiy (mainly the PTK boxes for the
user), the admin also generates a 32-byte random "deletion key" for X in T at
the given role (call it K). It boxes this key twice: once for the admins and
owners of T, using T's admin PTK (call it B1); and a second time for user X's
PUK (call it B2). Both boxes are written to the server and stored under the
primary key (T, X, R, Q), where R is the role and Q is the teamchain seqno of
the addition.

If X is a team rather than a user, the same protocol is followed, but using X's
PTK to encrypt in encryption (2) above.

When X is added to T, the server stores a notification for X. X polls these
notifications and when it sees a new one, will update its membership chain to
reflect the new mambership.  The notification comes down from the server with
(T,R,Q) as above, and also B2.  The user unboxes B2, and if successful, writes
(T,R,Q,H(K)) into its team membership chain. That is, the commitment of K is
stored in the chain, but not K itself, since K is hidden from the server.

## Removing Membership

When the admins of T (maybe A, maybe someone else) wants to remove X from the team,
they are free to do so at will. As a courtesy, they should "sign" this removal
by MAC'ing a statement with the deletion K key. The server can then send this
statement down to X, who can then reliably write to its team membership chain
that it has been removed from T. The server should send down B2 along with this
removal notification. When X unboxes K, it checks it against the commitment H(K)
that accompanied the addition link in X's team membership chain.

## Creating a Team

Creating a team is pretty similar to the above, with the important constraint that
A=X, and therefore, both chains can be written at once.