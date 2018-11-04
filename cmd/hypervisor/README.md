# hypervisor
SmallStack Virtual Machine Hypervisor.

The *hypervisor* daemon manages virtual machines (VMs). Please read the
[SmallStack design document](https://bit.ly/SmallStack) to understand the
architecture.

## Status page
The *hypervisor* provides a web interface on port `6976` which shows a status
page, links to built-in dashboards and access to performance metrics and logs.
If *hypervisor* is running on host `myhost` then the URL of the main
status page is `http://myhost:6976/`. An RPC over HTTP interface is also
provided over the same port.


## Startup
*Hypervisor* is started at boot time, usually by one of the provided
[init scripts](../../init.d/). The *hypervisor* process is baby-sat by the init
script; if the process dies the init script will re-start it. It may be stopped
with the command:

```
service hypervisor stop
```

which also kills the baby-sitting init script. It may be started with the
comand:

```
service hypervisor start
```

There are many command-line flags which may change the behaviour of
*hypervisor* but many have defaults which should be adequate for most
deployments. Built-in help is available with the command:

```
hypervisor -h
```

## Security
RPC access is restricted using TLS client authentication. *Hypervisor* expects
a root certificate in the file `/etc/ssl/CA.pem` which it trusts to sign
certificates which grant access to methods. It trusts the root certificate in
the `/etc/ssl/IdentityCA.pem` file to sign identity-only certificates.

It also requires a certificate and key which grant it the ability to **get**
images and objects from an *[imageserver](../imageserver/README.md)*. These
should be in the files
`/etc/ssl/hypervisor/cert.pem` and `/etc/ssl/hypervisor/key.pem`, respectively.

## Control
The *[vm-control](../vm-control/README.md)* utility may be used to create,
modify and destroy VMs.

The *[hyper-control](../hyper-control/README.md)* utility is used to perform
administrator tasks on the *Hypervisor*.
