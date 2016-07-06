# subtool
A utility to control the *[subd](../subd/README.md)* daemon that runs on every
dominated system.

The *subtool* utility may be used to debug and control a running *subd*.
*Subtool* may be run on any machine and can be used to manipulate various
operating parameters of a running *subd* and perform RPC requests. It is
typically run on a desktop or bastion machine.

## Usage
*Subtool* supports several sub-commands. There are many command-line flags which
provide parameters for these sub-commands. The most commonly used parameter is
`-subHostname` which specifies which host the *subd* to control is running on.
The basic usage pattern is:

```
subtool [flags...] command [args...]
```

Built-in help is available with the command:

```
subtool -h
```

Some of the sub-commands available are:

- **fetch**: tell *subd* to fetch the specified object from the objectserver
- **get-config**: get the current configuration from *subd*
- **get-file**: get a file from *subd*
- **poll**: get the checksumed file-system representation
- **push-image**: push an image directly to the *[subd](../subd/README.md)*,
                  bypassing the *[dominator](../dominator/README.md)*
- **set-config**: set the current configuration of *[subd](../subd/README.md)*
                  (such as rate limits for scanning the file-system and
                  **fetching** objects)

Note that sub-commands which change the configuration of
*[subd](../subd/README.md)* may be reverted by the
*[dominator](../dominator/README.md)*. Thus, it may be more appropriate to use
the *[dominator](../dominator/README.md)* to change the configuration of all the
*[subd](../subd/README.md)* instances in the fleet.

## Security
*[Subd](../subd/README.md)* restricts RPC access using TLS client
authentication. *Subtool* expects a valid certificate and key in the files
`~/.ssl/cert.pem` and `~/.ssl/key.pem`, respectively. *Subtool* will present
this certificate to *subd*. If the certificate is signed by a certificate
authority that *subd* trusts, *subd* will grant access.
