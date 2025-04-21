# Team Invites

Team invites are a tricky dance since the parties involves don't want to allow
open access to the relevant sigchains, especially across multiple hosts.
Here is the plan to address this:

## Remote Joins

Setup: alice@x.xx is is an admin of team T. She invites bob@y.yy to join as a reader.
(Note that bob can't join as an admin or owner since he's on a different host).

1. Alice generates a TeamCert for T. This cert contains:
   - The teamID
   - The latest PTK
   - A Timestamp (since it eventually should expire).
It is signed by:
   - The first PTK, that corresponds to the teamID
   - The latest PTK

2. Alice posts this team cert to x.xx, via the RPC TeamAdmin.StoreTeamCert. Alice can do
this as an authenticated user of x.xx, and as an authorized admin of T. This RPC
as a result stores the cert to the SQL table foks_users.team_certs. 

3. Alice now can make a TeamInvite, which has two components: (1) the hash of the
cert she generated in step 1; and (2) the hostID of x.xx. This is a smallish
data object, that is encoded to about 101 bytes of ASCII via B62Encoding. For
example: YcarI5JTMAVWou8ubp26lQsnot0YPa8DlHL05Ip0RvdHFtGK4KZH9gClxMCZQoTyov0mWMU0GgdAUVMvtQRIK8OBbSinNfezVQnpr
Alice can post this into a signal or whatsapp group, saying "please join my team
for further work on this topic." Local or remote users can then copy/paste this
into their `foks team invite-accept` command.  For now, let's only cover the
remote case.

4. Bob sees the invite (YcarI5JTM...) in his WhatsApp group, and now he wants to 
join the team T. What he needs to do is to grant the admins of T the ability to
load his user, even remotely. This is done via "PermissionTokens", which are
bearer tokens that allow the holder to load Bob's user.  The interesting thing 
here is that Bob doesn't trust x.xx as a whole to handle this token, only
Alice and the other admins of T. Thus, Bob needs to encrypt this token
so that Alice can decrypt it. Bob has team T's public Dh key, or can get ahold
of it via the TeamInvite he received. This first step is getting the whole
cert (which is about 300 bytes) via the hash he has. He splits the invite into
two parts. He does a beacon discovery of the host ID (which maps to x.xx), then
issues a TeamGuest.LookupTeamCertByHash RPC to the reg server of that host.
(It has to be reg since Bob doesn't have a login on host x.xx). The RPC
returns the full cert, and then Bob can hash it to ensure it matches the
hash he has. On the server side, this RPC is served via SELECT into the
foks_user.team_certs token. Anyone who knows the hash (even logged-out users)
can get the cert.

5. Bob sends a User.grantRemoteViewPermission RPC to y.yy, his home server.
The grant is scoped to the team in the invite, so it can be later revoked if
necessary. The RPC inserts the grant into foks_users.remote_view_permissions,
and returns a 16-byte random view token.

6. Now Bob can formulate a TeamRemoteJoinReq. The plaintext inputs 
contain the view token, and are then boxed for the DH public key that Bob
got in step 4. Bob uses his PUK as the other key in the DH computation.

7. Bob posts the TeamRemoteJoinReq to x.xx, indexed by the TeamCert 
generated in Step 1. This is via the TeamGuest.AcceptInvite RPC,
running on x.xx's Reg server. Note that Bob doesn't need to be logged in
to do this, he only needs to know a TeamCert. As such, he can DoS
x.xx, but the server can disable team T to mitigate this attack. This
operation yields a TeamRemoteJoinReqToken, which Bob can paste back into
his WhatsApp group, or x.xx can communicate to Alice via the "team inbox"
mechanism (which is TBD). The resulting row shows up in 
foks_users.remote_joinreqs, for Alice to access in the next step....

8. Alice gets the TeamRemoteJoinReqToken, and uses it to load the
TeamRemoteJoinReq that Bob put on the database in Step 7. This is
via the RPC TeadAdmin.LoadTeamRemoteJoinReq, but maybe in the future
there might be an inbox function.

9. Alice can now add Bob to her team, and she can attach the TeamRemoteJoinReq
into the editTeam RPC. She stores Bob's permission token in 
`foks_users.team_member_view_tokens`, boxed for the admin's PTK.

## Local Joins

As above, Alice is the admin, and Bob wants to join, but it's a local team, so 
they are both on the same (virtual) host.

1-3. As above, Alice can generate a 101-character invitation code. This is good
for both a remote or a local join. Even for a local join, it's good to have the
full hostID as part of the invitation code, to dispell any ambiguities around
the fact that users can act as multiple personaes, or teams, across any number
of virtual or real hosts.

4-8. Bob gets the invite, and grants local view permission to the team given by
the invitation. The outcome is that a row is inserted into
local_view_permissions, yielding a permission token as above (in the remote
case). Bob can insert a row directly into local_joinreqs to request memobership.

9. Alice can now add Bob to the team. There's no real need to attach
a view token, since local viewership is based on database ACLs.

# Removals

Let's say Alice invites Bob into a team and later removes Bob. Bob then loses
the ability to load to the team, so he can't verify the fact that Alice was the
one who actually removed him. It could just be the server DoS'ing him!

To address this issue, when Alice adds Bob into a team, she also generates
a random 32-byte removal token, via team.NewBoxedTeamRemovalKey. This key
is encrypted twice --- once for the team admins, and once for the member
being added. Alice sends these two boxes up to the server, along with a commitmment 
to this key, which is just the SHA512/256 hash of the key (we call this
the "key commitment"). Of course, it's a keyed hash, using the typical SNOWP-driven
keyed-hashing system.

On removal, the admin unboxes the key (note this might have been an admin different
from Alice), and then uses this key to HMAC a proto.TeamRemovalMACPayload. This HMAC
is stored on the server, which then can send it to Bob later. The server could not
have forged this, and Bob can grudgingly admit that Alice (or her cohort) wanted
her out.  Without this proof, Bob might be suspicious that the server is
DoS'ing him.

# Open Hosts

The admin of a vhost (whether canned or free-form) can set the host to be open.
This means, any Alice and Bob can see each other on the host.