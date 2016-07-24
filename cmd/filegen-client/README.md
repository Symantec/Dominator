# filegen-client
A utility to test and benchmark a
*[filegen-server](../filegen-server/README.md)*.

The *filegen-client* utility may be used to send queries to a running
*filegen-server* in order to debug or benchmark the server.

## Usage
*Filegen-client* will get per-machine file data for a specified pathname from
the specified *filegen-server*. The data for each machine are written to stdout.
The basic usage pattern is:

```
filegen-client [flags...] pathname source
```

Built-in help is available with the command:

```
filegen-client -h
```

The `-mdbFile` option specifies the file to read MDB data from. If this file
changes then it is re-read and if the MDB data for any machine changes, new file
contents will be generated and displayed.

## Security
*[Filegen-server](../filegen-server/README.md)* restricts RPC access using TLS
client authentication. *Filegen-client* will load certificate and key files from
the `~/.ssl` directory. *Filegen-client* will present these certificates to
*filegen-server*. If one of the certificates is signed by a certificate
authority that *filegen-server* trusts, *filegen-server* will grant access.
