# dominator
The *dominator* daemon is the heart of the **Dominator** system. It continuously
**polls** all the known *subs* and directs them to make corrections if needed.

The list of known *subs* is read from a local file (`/var/lib/Dominator/mdb` by
default). This file is updated by the *[mdbd](../mdbd/README.md)* companion
daemon.

## Status page
The *dominator* provides a web interface on port `6970` which provides a status
page, links to built-in dashboards and access to performance metrics and logs.
If *dominator* is running on host `myhost` then the URL of the main status page
is `http://myhost:6970/`.

## Startup
*Dominator* is started at boot time, usually by one of the provided
[init scripts](../../init.d/). The *dominator* process is baby-sat by the init
script; if the process dies the init script will re-start it. It may be stopped
with the command:

```
service dominator stop
```

which also kills the baby-sitting init script. It may be started with the
comand:

```
service dominator start
```

There are many command-line flags which may change the behaviour of *dominator*
but many have defaults which should be adequate for most deployments. Built-in
help is available with the command:

```
dominator -h
```

### Key configuration parameters
The init script reads configuration parameters from the `/etc/default/dominator`
file. The following is the minimum likely set of parameters that will need to be
configured.

The `IMAGE_SERVER_HOSTNAME` variable specifies the hostname where the
*[imageserver](../imageserver/README.md)* is running. This hostname must be
resolvable by the *dominator* and all the *subs*. In a multi-zone deployment,
it is recommended to use a geoDNS name, as it makes *dominator* configuration
uniform across zones.

The `USERNAME` variable specifies the username that *dominator* should run as.
Since *dominator* does not need root privileges, the init script runs
*dominator* as this user.

## Security
RPC access is restricted using TLS client authentication. *Dominator* expects a
root certificate in the file `/etc/ssl/CA.pem` which it trusts to sign
certificates which grant access.

*Dominator* will require signed SSL certificates in order to communicate with
*[subd](../subd/README.md)* and the *[imageserver](../imageserver/README.md)*.
The certificate and key should be in the files
`/etc/ssl/dominator/cert.pem` and `/etc/ssl/dominator/key.pem`, respectively.

If any of these files are missing, *dominator* will refuse to start. This
prevents accidental deployments without access control.

## Control
The *[domtool](../domtool/README.md)* utility may be used to manipulate various
operating parameters of a running *dominator* and perform RPC requests.
