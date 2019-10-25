# filegen-server
The *filegen-server* daemon serves computed files for the **Dominator** system.

The *[dominator](../dominator/README.md)* queries zero or more *filegen-server*
instances when it needs to distribute *computed files*. This *filegen-server* is
a reference implementation and serves some simple computed files. For more
custom types of computed files, see the documentation for the
[lib/filegen](https://godoc.org/github.com/Cloud-Foundations/Dominator/lib/filegen)
package. This reference implementation may be used as a template for writing
your own file generator.

## Status page
The *filegen-server* provides a web interface on port `6972` which provides a
status page, links to built-in dashboards and access to performance metrics and
logs. If *filegen-server* is running on host `myhost` then the URL of the main
status page is `http://myhost:6972/`. An RPC over HTTP interface is also
provided over the same port.


## Startup
*Filegen-Server* is started at boot time, usually by one of the provided
[init scripts](../../init.d/). The *filegen-server* process is baby-sat by the
init script; if the process dies the init script will re-start it. It may be
stopped with the command:

```
service filegen-server stop
```

which also kills the baby-sitting init script. It may be started with the
comand:

```
service filegen-server start
```

There are many command-line flags which may change the behaviour of
*filegen-server* but many have defaults which should be adequate for most
deployments. Built-in help is available with the command:

```
filegen-server -h
```

### Key configuration parameters
The init script reads configuration parameters from the
`/etc/default/filegen-server` file. The following is the minimum likely set of
parameters that will need to be configured.

The `CONFIG_FILE` variable specifies the name of the file from which to read the
configuration.

The `USERNAME` variable specifies the username that *filegen-server* should run
as. Since *filegen-server* does not need root privileges, the init script runs
*filegen-server* as this user.

## Security
RPC access is restricted using TLS client authentication. *Filegen-Server*
expects a root certificate in the file `/etc/ssl/CA.pem` which it trusts to sign
certificates which grant access. It also requires a certificate and key which
clients will use to validate the server. These should be in the files
`/etc/ssl/filegen-server/cert.pem` and `/etc/ssl/filegen-server/key.pem`,
respectively.

## Configuration file
The configuration file contains zero or more lines of the form:
`keyword pathname [args...]`. The keyword specifies an algorithm to use to
generate data for the specified *pathname*. The following keywords are
supported:

- **DynamicTemplateFile** pathname *filename*: the contents of *filename* are
  used as a template to generate the file data. If the file contains sections of
  the form `{{.MyVar}}` then the value of the `MyVar` variable from the MDB for
  the host are used to replace the section. If *filename* changes (replaced with
  a different inode), then the data are regenerated and distributed to all
  clients

- **File** pathname *filename*: the contents of *filename* are used to provide
  the file data. If *filename* changes (replaced with a different inode), then
  the data are regenerated and distributed to all clients

- **MDB** pathname: the file data are the JSON encoding the MDB data for the
  host

- **StaticTemplateFile** pathname *filename*: the contents of *filename* are
  used as a template to generate the file data. If the file contains sections of
  the form `{{.MyVar}}` then the value of the `MyVar` variable from the MDB for
  the host are used to replace the section
