# Keybase v3

Keybase v3 is the third build of Keybase, the first being the original one and
the second that was built in-house at Zoom. It's meant primarily as a dev tool
for developer-types, and not for mainstream consumer adoption. But as such, it
will have features that didn't seem worthwhile for the first two versions.

## Why?

I think this will be a fun project to build, and I think it find at least 1,000
true fans, and probably a lot more, since it's critical infrastructure for which
there isn't currently a good answer.  If Keybase git went offline tomorrow, I
for one would sorely miss it and would wish for a viable alternative.

## Intended Audience

Keybase v3 is aimed at developers, from software engineers to devops engineers, to 
security researchers. The intended audience is roughly "those people who understand,
at a basic level, how to use git."

I think for several reasons, developers didn't feel comfortable taking to Keybase v1.
The most important of these is that Keybase v1 was hardcoded with a centralized server,
run by Keybase. This configuration meant Keybase couldn't be run from the datacenter, since
most production configurations don't allow external network dependencies, or at least
avoid them for robustness. Also developers are naturally skeptical of "vendor lock-in"
and likely won't trust a long-term storage mechanism if they can't control it or at 
least have the option to swap it out. And similarly, Keybase v3 should funtion as "basic 
infrastructure" much like TLS, SMTP or DNS. And most developers would be skeptical
of basic infrastructure that's transparently controled by a for-profit, VC-backed
company.

Also, Keybase didn't support some key features that developers often wanted, like SSO (required for
enterprise use) and YubiKeys, which are in many ways a superior backup solution to paper keys.

## How It's Different from Keybase v1

Keybase v3 is a lot like Keybase v1, but it has some key feature addition and subtractions.

### Feature Additions

* Federated Key directory; there will be a key directory run by the company, but
  also anyone can run their own key directory, and interoperate with the others in use. These
  key directories will be independently adminstrated but have to run the agreed-upon protocol.
  This idea being similar of course to the Web/HTTP and Email/SMTP.
* Mutable names: a "name" for something will be global in an administrative domain, and will
  point to a user or a group. The name is mutable, so it can be changed.
* Groups of groups of groups, etc: The group system will be recursive so there can be groups
  of groops, etc. Also, individual devices or yuibkeys can be members of groups, etc. The
  structure is generic, so "users" in the Keybase sense and "groups" in the Keybase same
  are treated roughly equivalent.
* Privacy-preserving Transparency Tree: The key directory will preserve the privacy of keys,
  users and groups as much as is possible. Users will grant other users the abiilty to explicitly
  access their public key data, and it will not be visible by default.
* Yubikey support: better than paper keys
* SSO support: required for enteprirse adoption
* Better support for "bot" users: in Keybase, they were hacked in as users, but this time, we'll
  give them better treatment so it's safer to give key access to compute nodes in the datacenter.
* Independent key directory, git server, chat server, etc: the key directory, git server and chat
  server can be mix-and-match, so you can use the same provider or different providers, depending on
  your tastes and needs.

### Feature Subtractions

* Social proofs
* Sharing before signup
* Adding users to groups without that user's permission
* Nice GUI, at first
* Chat, at first
* Any blockchain dependency, except, maybe, a global directory of domains
* PGP

## Roadmap

## How Will Anyone Learn about it?

## What Applications Will be built?

## How Will it Make Money?

# Blog Schedule

The release of FOKS will present to the world a lot of new stuff in a short amount of time.
With Keybase, we had a slow march of features being released, and could count on each big 
release to revitalize the buzz. Here, we don't want to shoot our shot all at once. So
therefore, I think the following sequence of blog posts will be good:

  - Keybase v3: The Overall Vision (see above)
  - Keybase v3: The Overall Architecture
  - Keybase v3: Use of TLS and x.509 and questions about Let's Encrypt
  - Keybase v3: Privcay-preserving Transparency Tree Without Fancy Crypto
  - Keybase v3: Snowpack RPC
  - Keybase v3: Post-Quantum Support, and YubiKey
  - Keybase v3: The KV-Store
  - Keybase v3: Fast Git



