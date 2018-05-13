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

- **boost-cpu-limit**: raise the CPU limit until the next scan cycle (this does
                       not change the priority (nice) level)
- **cleanup**: empty the object cache
- **delete**: delete specified pathnames
- **fetch**: tell *subd* to fetch the specified object from the objectserver
- **get-config**: get the current configuration from *subd*
- **get-file**: get a file from *subd*
- **list-missing-objects**: list objects in the specified image that are missing
                            on the sub
- **poll**: get the checksumed file-system representation
- **push-file**: push a single file
- **push-image**: push an image directly to the *[subd](../subd/README.md)*,
                  bypassing the *[dominator](../dominator/README.md)*
- **push-missing-objects**: push objects in the specified image that are missing
                            to the sub
- **restart-service**: restart the specified service
- **set-config**: set the current configuration of *[subd](../subd/README.md)*
                  (such as rate limits for scanning the file-system and
                  **fetching** objects)
- **show-update-request**: compute and show the update request for the
                           specified image
- **wait-for-image**: wait for the sub to be updated to the specified image
                      (another entity is responsible for triggering the update)

Note that sub-commands which change the configuration of
*[subd](../subd/README.md)* may be reverted by the
*[dominator](../dominator/README.md)*. Thus, it may be more appropriate to use
the *[dominator](../dominator/README.md)* to change the configuration of all the
*[subd](../subd/README.md)* instances in the fleet.

## Security
*[Subd](../subd/README.md)* restricts RPC access using TLS client
authentication. *Subtool* will load certificate and key files from the
`~/.ssl` directory. *Subtool* will present these certificates to *subd*. If one
of the certificates is signed by a certificate authority that *subd* trusts,
*subd* will grant access.
