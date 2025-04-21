# Server-Assisted Passphrase Encryption

MK Note 2024.06.04 -- I think this doc is out-of-date.

Users can choose to lock their local secret keys with a passphrase.
The server assists here so that the same passphrase is synced across
all devices. A change on one device will automatically reflect on the
others. Also, the user can force a freeze from the server-side if desired,
so in other words, the server retains a portion of the data used to 
unlock the passphrase and can withhold it under the right conditions.
But having it doesn't yield a decryption of the passphrase either.

## Basics

* The user keeps the same per-user salt across all passphrase versions.
* If the user has multiple accounts, each one gets its own passphrase.
This is of course necessary when the accounts are on different servers.
* To stretch, we are using argon2id with the parameters:
   * https://www.rfc-editor.org/rfc/rfc9106.html#name-parameter-choice
   * `If much less memory is available, a uniformly safe option is Argon2id with t=3 iterations, p=4 lanes, m=2^(16) (64 MiB of RAM), 128-bit salt, and 256-bit tag size. This is the SECOND RECOMMENDED option. `

## For passphrase version i:
* Compute stream_i = stretch(salt, pp_i)
* Compute e_i = stream_i[0:32]   --- EdDSA private key
* Compute s_i = stream_i[32:64]  --- symmetric encryption private key
* Compute r_i, 32-bytes of randomness
* Encrypt the local keys with secret_box and r_i as the key
* Collect the list L_i = (j, r_j) for all j. 
  * Client can query the old (j, r_j) from server if it knows the previous passphrase pp_{i-1}.
* Send the server L_i encrypted with s_i
* Send the encrypted (j, r_j), and E_i (the public key for e_i) up to server
* Send the server also e_i and r_i encrypted with PUSK_k, the latest per-user super key
  * The PUSK is shared amongst the user's backup devices.



