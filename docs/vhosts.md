# Primary and Virtual Hosts in FOKS Backends

A FOKS server backend consists of a suite of services (like user, reg, probe, merkle_batcher, etc) 
and also a persistent database they write to and read from. But multiple server suites are
allowed to talk to the same database (as currently happens in the FOKS test suite).

This doc discusses how we can have one primary, and 0 or more virtual hosts all running 
ontop of the same suite of service processes. We think we'll need this for allowing
FOKS users to host their own "host" without bothering to do much.

## Primary Hosts

### Keys, Chains, and Zones

A primary host in FOKS is determined by these 4 EdDSA keys:

    - The HostKey, the most imprtant key
    - The TLS CA key, used for signing TLS certs for this host
    - The Merkle Signer key, used for signing the host's merkle Tree
    - The Metadata Signer key, used to sign metadata about the host, like which 
      DNS names map to the host's FOKS services

Of these 4 keys, the first (the HostKey) is the most imporatant, since it can
sign others into existence. It can be rotated via the hostchain, but the hash
of the first HostKey becomes the "host ID", the unique identifier for this host.

When a host is initialzed, these 4 keys are generated, and a hostchain with one
link is generated. That chain introduces the HostKey, and also the 3 delegated
keys, signing and counter-signing them. The host can later introduce other links
to rotate or revoke any of these keys.

Once the hostchain and associated keys are generated, the host can generate
a Zone file. This file is signed with an active HostKey, and is stored on the
server. It is given out to clients via the "probe" process, which then allows
them to connect to the various services for this primary host. 

The zone file contains TCP Addresses for:

    - the Probe service itself -- used for probing hostchains and zones for this host
    - the Reg service -- used for registering, and opering logged-out
    - the User service -- used by logged-in users for this service (requiring mTLS)
    - the Merkle Query service -- used to query the merkle tree for this service;
        all data is publicly available.

### Constituent Services (front- and backend)

So what makes up a running FOKS services for a primary host?  And how is it different
for a virtual host? Let's deal with the primary host first.

As stated above, a primary host has a HostID generated from its HostKey. For this dicussion,
call it .2fAeBQ for this example (though a real hostID is longer). Also for example, let's
name the primary host here `ne43.pub`, meaning, the probe service for the primary
host will be available at the DNS name `ne43.pub`, port 443, and will have a TLS cert 
signed by the root CAs for `ne43.pub`.

There are two lookup paths for this host:

  - `ne43.pub:443` -> `.2fAeBQ` via probe service, over TLS.
  - `.2fAeBQ` -> `ne43.pub:443` via beacon service (run at `beacon.ne43.pub`) over TLS. This beacon
  server is hardcoded into the clients, or can be configured.

Either lookup yields a "probe" reply, that might have the following data:

  - CA for this service (`.2fAeBQ`)
  - `becaon.ne43.pub:443` -- beacon service, which uses the global CAs. This might be the only one
     globally for now.
  - `ne43.pub:443` -- probe server, which uses the global CAs
  - `user.ne43.pub:444` -- user service, which uses the service-specific CA
  - `reg.ne43.pub:445` -- reg service, which uses the service-specific CA
  - `merkle.ne43.pub:446` -- merkle service, which uses the service-specific CA

In this case, the various `*.ne43.pub` DNS names are all aliases for the same IP address (except 
for beacon, which has its own IP address).

There are various other backend hosts that need not have DNS names and aren't
accessible via the public internet. They are:

  - `merkle_batcher` -- batches merkle updates into deterministic batches
  - `merkle_builder` -- builds batches into the merkle tree
  - `merkle_signer` -- signs the merkle tree once it's been updated
  - `queue` -- a queue service (kind of like SQS) for driving Kex
  - `internal_ca` -- a CA that gives out client certificates to other backend processes so
    they can connect via mTLS


For all ~10 of these services, they get their own cert/key pair.  The first two use the 
root CAs, all the rest use the local CA for this service.

Also, internally, there is a `ShortHostID` for this host (like `2`), which shows up
in many columns of many tables. This is a short, unique identifier for this host,
local to this database.

# Virtual Hosts

Imagine a service like `ne43.pub` wants to make it easy for customers to have their own
FOKS host without having to host it themselves. They can have a "virtual host". 

First off, it's probably best to have a separate domain for these virtual hosts, to
eliminate any confusion or potential security problems. Call this domain `ne43.io`.
Now, if Nike Shoes comes along and wants a virtual host, it will go like this:

  - First, pull a new `ShortHostID` unique to the current database. Call it `15` for now.
  - Insert a mapping of `15` -> `2` in the `vhosts` table
  - Roll a new HostKey and HostID. Call the HostID `.2Ge4ffG`.
  - Establish probe and beacon mappings:
    - `nike.ne43.io:443` -> `.2Ge4ffG` (via probe service)
    - `.2Ge4ffG` -> `nike.ne43.io:443` (via beacon service)
  - When running the probe service `nike.ne43.io:443`, it will be signed via the global CAs 
  - Nike will use the same internal CA as the primary host, `.2fAeBQ`
  - The public Zone for `nike.ne43.io`, returned via its probe, will return
    the same non-probe servers as above (`user.ne43.pub:444`, etc).
  - The non-hostKeys for nike.ne43.io will point to those of the primary server, `ne43.pub`,
    via new linktype in the hostchain, Alias.

For those services that need a login, like the User service, the mTLS handshake will tell
the service which HostID to route to, since there might be several at this IP address/port.
For logged out services like Reg and MerkleQuery, we have two options: either make a throwawy
cert that specifics the intended hostID, and do mTLS anyways; or just add the host ID
in to the RPCs; actually a third option is SetVHost() call on in the protocol.


