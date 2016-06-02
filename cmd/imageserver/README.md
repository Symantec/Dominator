# imageserver
The *imageserver* daemon serves images and objects for the **Dominator** system.

## Status page
The *imageserver* provides a web interface on port `6971` which provides a
status page, links to built-in dashboards and access to performance metrics and
logs. If *imageserver* is running on host `myhost` then the URL of the main
status page is `http://myhost:6971/`. An RPC over HTTP interface is also
provided over the same port.


## Startup
*Imageserver* is started at boot time, usually by one of the provided
[init scripts](../../init.d/). The *imageserver* process is baby-sat by the init
script; if the process dies the init script will re-start it. It may be stopped
with the command:

```
service imageserver stop
```

which also kills the baby-sitting init script. It may be started with the
comand:

```
service imageserver start
```

There are many command-line flags which may change the behaviour of
*imageserver* but many have defaults which should be adequate for most
deployments. Built-in help is available with the command:

```
imageserver -h
```

### Key configuration parameters
The init script reads configuration parameters from the
`/etc/default/imageserver` file. The following is the minimum likely set of
parameters that will need to be configured.

If `ARCHIVE_MODE` is set to `true` then the *imageserver* will ignore all
**delete** operations, effectively turning it into an archiver (backup).

The `IMAGE_DIR` variable specifies the directory where images are stored. It is
recommended to specify a directory on a file-system with plenty of free space.

The `IMAGE_SERVER_HOSTNAME` variable specifies another *imageserver* which will
serve as the source of image updates. This may be used to configure simple, fast
and secure image replication between *imageservers*. If this variable is unset
then the *imageserver* is a master/standalone server

The `OBJECT_DIR` variable specifies the directory where objects are stored. It
is recommended to specify a directory on a file-system with plenty of free
space.

The `USERNAME` variable specifies the username that *imageserver* should run as.
Since *imageserver* does not need root privileges, the init script runs
*imageserver* as this user.

## Security
RPC access is restricted using TLS client authentication. *Imageserver* expects
a root certificate in the file `/etc/ssl/CA.pem` which it trusts to sign
certificates which grant access. It also requires a certificate and key which
grant it the ability to **get** images and objects from another imageserver.
These should be in the files `/etc/ssl/imageserver/cert.pem` and
`/etc/ssl/imageserver/key.pem`, respectively.

## Control
The *[imagetool](../imagetool/README.md)* utility may be used to add, delete,
get and compare images. It is the most important utility in the **Dominator**
system.
