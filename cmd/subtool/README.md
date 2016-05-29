# subtool
A utility to control the *subd* daemon that runs on every dominated system.

The *subtool* utility may be used to debug and control a running *subd*. It can
be used to manipulate various operating parameters of a running *subd* and
perform RPC requests.

## Usage
*Subtool* supports several sub-commands. There are many command-line flags which
provide parameters for these sub-commands. The basic usage pattern is:

```
subtool [flags...] command [args...]
```

Built-in help is available with the command:

```
subd -h
```

Some of the sub-commands available are:

- **fetch**: tell *subd* to fetch the specified object from the objectserver
- **get-config**: get the current configuration from *subd*
- **get-file**: get a file from *subd*
- **poll**: get the checksumed file-system representation
- **set-config**: set the current configuration of *subd*

Note that sub-commands which change the configuration of *subd* may be reverted
by the *dominator*.

## Security
*Subd* restricts RPC access using TLS client authentication. *Subtool* expects
a valid certificate and key in the files `~/.ssl/cert.pem` and `~/.ssl/key.pem`,
respectively. *Subtool* will present this certificate to *subd*. If the
certificate is signed by a certificate authority that *subd* trusts, *subd* will
grant access.
