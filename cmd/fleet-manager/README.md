# fleet-manager
SmallStack Fleet Manager.

The *fleet-manager* daemon manages a fleet of
*[Hypervisors](../hypervisor/README.md)* and the Virtual Machines running on
them. Please read the
[SmallStack design document](https://bit.ly/SmallStack) to understand the
architecture.

## Status page
The *fleet-manager* provides a web interface on port `6977` which shows a status
page, links to built-in dashboards and access to performance metrics and logs.
If *fleet-manager* is running on host `myhost` then the URL of the main
status page is `http://myhost:6977/`. An RPC over HTTP interface is also
provided over the same port.


## Startup
*fleet-manager* is started at boot time, usually by one of the provided
[init scripts](../../init.d/). The *fleet-manager* process is baby-sat by the init
script; if the process dies the init script will re-start it. It may be stopped
with the command:

```
service fleet-manager stop
```

which also kills the baby-sitting init script. It may be started with the
comand:

```
service fleet-manager start
```

There are many command-line flags which may change the behaviour of
*fleet-manager* but many have defaults which should be adequate for most
deployments. Built-in help is available with the command:

```
fleet-manager -h
```

## Security
RPC access is restricted using TLS client authentication. *fleet-manager*
expects a root certificate in the file `/etc/ssl/CA.pem` which it trusts to sign
certificates which grant access to methods. It trusts the root certificate in
the `/etc/ssl/IdentityCA.pem` file to sign identity-only certificates.

It also requires a certificate and key which grant it the ability to manage
*[Hypervisors](../hypervisor/README.md)*. These should be in the files
`/etc/ssl/fleet-manager/cert.pem` and `/etc/ssl/fleet-manager/key.pem`,
respectively.

## Control
The *[vm-control](../vm-control/README.md)* utility may be used to create,
modify and destroy VMs.

The *[hyper-control](../hyper-control/README.md)* utility is used to perform
administrator tasks on *Hypervisors*.
