# hyper-control
A utility to manage Hypervisors.

The *hyper-control* utility manages Hypervisors. It is typically run on a
desktop or bastion machine.

## Usage
*hyper-control* supports several sub-commands. There are many command-line flags
which provide parameters for these sub-commands. The most commonly used
parameters are `-fleetManagerHostname` or `-hypervisorHostname` which specify
either the Fleet Manager or a specific *Hypervisor* to communicate with. The
basic usage pattern is:

```
hyper-control [flags...] command [args...]
```

Built-in help is available with the command:

```
hyper-control -h
```

Some of the sub-commands available are:

- **add-address**: manually add a MAC address and IP address pair to a specific
                   *Hypervisor*. If the IP address is not specified an external
                   DHCP server is required to provides leases to VMs. This
                   command is only required if a *Fleet Manager* is not
                   available
- **add-subnet**: manually add a subnet to a specific *Hypervisor*. This command
                  is only required if a *Fleet Manager* is not available
- **get-updates**: get and show a continuous stream of updates from a
                   *Hypervisor* or *Fleet Manager*. This is primarily for
                   debugging
- **remove-excess-addresses**: remove free addresses for a specific *Hypervisor*
                               above the specified limit

## Security
The *Hypervisor* restricts RPC access using TLS client authentication.
*hyper-control* will load certificate and key files from the
`~/.ssl` directory. *hyper-control* will present these certificates to
the *Hypervisor*. If one of the certificates is signed by a certificate
authority that the *Hypervisor* trusts, the *Hypervisor* will grant access.
