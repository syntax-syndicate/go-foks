### Discovering New FOKS Nodes

Discovery happens in several places:

* On signup
* When inviting users from other servers into your team

In both cases, you can either use a 32-byte host ID or alternatively,
a DNS name that will be connected to via TLS using x509 certificates
and the root chain of trust. The former can be considered more secure, 
since it doesn't rely on trusting the root servers, but the latter is
obviously more convenient.

## Via HostID

Flow for initial lookup:

* User makes an RPC to directory.ne43.net:443 using the Beacon protocol,
asking for the DNS name, port, (and potentially root CA) for the hostID. 
* Gets back DNS name and root CA. Call this merkle.teledyne.ca, port 443, with a rootCA
to use
  * This is the merkle server / DNS name for this hostID.
  * Writes down this DNS name in sqlite hard state, should not have to query
    directory server again, unless the DNS resolution is removed, or the host
    at that DNS name goes down.
* Query merkle.teledyne.ca, port 443, over TLS over the Discovery protocol, get back:
  * Merkle Root, with header block that points to tail of hostchain
  * Hostchain from seqno=1
  * User writes down merkle root and also hostchain in sqlite hard state
* Play back chain:
  * ensure that the initial hostID matches what we started with
  * Get a HostTLSCA and a HostMetadaSigner keys
* Ask the Merkle server over the Discovery protocol for the latest endpoints, which should be signed
  with the HostMetadataSigner key.
* Now can make calls to reg.teledyne.ca etc to register.

Flow for subsequent lookups:

* Check sqlite hard state for hostid -> DNS name mapping.
* Request updated hostchain from merkle server:
  * Get current root, which has pointer to hostchain tail
  * Play back merkle roots from current root to this root
  * Get hostchain links from latest tail to what we had in storage
  * Play back hostchain links
* Yields reg.teledyne.ca, merkle.teledyne.ca, etc.

If the initial mapping is missing or fails to resolve, can restart the protocol
as one did just above.

## Via HTTPS

Works via TOFU. Use TLS to get the HostID and merkle server. Writes this mapping down for 
future refernce in sqlite hard state. Doesn't need to query the directory server as above.

* User inputs hostname berries.is, and an optional port. Maybe also a rootCA? That UI is ugly.
* Queries berries.is with the discovery protocol, asking for the Merkle Server and the hostID
* Runs the discovery protocol as above.


