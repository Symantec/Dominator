# subd
The daemon that runs on every dominated system.

This daemon continuously checksum scans the root file-system and responds to
**poll**, **fetch files** and **update** RPC requests from the *dominator*.
In order to have a neglibible impact on system workload, it lowers its priority
(nice 15 by default), restricts itself to one CPU and automatically rate limits
its I/O to be 2% of the media speed.

## Status page
*Subd* provide a web interface on port `6969` which provides a status page,
access to performance metrics and logs. If *subd* is running on host `myhost`
then the URL of the main status page is `http://myhost:6969/`. An RPC over HTTP
interface is also provided over the same port.

## Startup
*Subd* is started at boot time, usually by one of the provided
[init scripts](../../init.d/). It may be stopped with the command:

```
service subd stop
```

and started with the comand:

```
service subd start
```

There are many command-line flags which may change the behaviour of *subd* but
the defaults should be adequate for most deployments. Built-in help is available
with the command:

```
subd -h
```

## Security
RPC access is restricted using TLS client authentication. *Subd* expects a root
certificate in the file `/etc/ssl/CA.pem` which it trusts to sign certificates
which grant access. It also requires a certificate and key which grant it the
ability to *fetch* files from the objectserver. These should be in the files
`/etc/ssl/subd/cert.pem` and `/etc/ssl/subd/key.pem`, respectively.

If any of these files are missing, *subd* will refuse to start. This prevents
accidental deployments without access control.

## Control and debugging
The [subtool](../subtool/) utility may be used to manipulate various operating
parameters of a running *subd* and perform RPC requests.
