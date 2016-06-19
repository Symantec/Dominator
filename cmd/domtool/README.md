# domtool
A utility to control the *[dominator](../dominator/README.md)*.

The *domtool* utility may be used to control a running *dominator*.
*Domtool* may be run on any machine and can be used to manipulate various
operating parameters of a running *dominator* and perform RPC requests. It is
typically run on a desktop or bastion machine.

## Usage
*Domtool* supports several sub-commands. There are many command-line flags which
provide parameters for these sub-commands. The most commonly used parameter is
`-domHostname` which specifies which host the *dominator* to control is running
on.
The basic usage pattern is:

```
domtool [flags...] command [args...]
```

Built-in help is available with the command:

```
domtool -h
```

Some of the sub-commands available are:

- **disable-updates**: tell *dominator* to not perform automatic updates of *subs*
- **enable-updates**: tell *dominator* to perform automatic updates of *subs*
- **configure-subs**: set the current configuration of all *subs* (such as rate limits
                  for scanning the file-system and **fetching** objects)

## Security
*[Dominator](../dominator/README.md)* restricts RPC access using TLS client
authentication. *Domtool* expects a valid certificate and key in the files
`~/.ssl/cert.pem` and `~/.ssl/key.pem`, respectively. *Domtool* will present
this certificate to *dominator*. If the certificate is signed by a certificate
authority that *dominator* trusts, *dominator* will grant access.
