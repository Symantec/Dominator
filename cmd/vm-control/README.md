# vm-control
A utility to manage Virtual Machines (VMs).

The *vm-control* creates and manages VMs by communicating with a *Hypervisor*.
It is typically run on a desktop, bastion or build machine.

## Usage
*Vm-Control* supports several sub-commands. There are many command-line flags
which provide parameters for these sub-commands. The most commonly used
parameters are `-clusterManagerHostname` or `hypervisorHostname` which specify
either the Cluster Resource Manager or a specific *Hypervisor* to communicate
with. The basic usage pattern is:

```
vm-control [flags...] command [args...]
```

Built-in help is available with the command:

```
vm-control -h
```

Some of the sub-commands available are:

- **add-address**: manually add a MAC address and IP address pair to a specific
                   *Hypervisor*. If the IP address is not specified an external
                   DHCP server is required to provides leases to VMs. This
                   command is only required if a *Cluster Resource Manager* is
                   not available
- **add-subnet**: manually add a subnet to a specific *Hypervisor*. This command
                  is only required if a *Cluster Resource Manager* is not
                  available
- **change-vm-tags**: change the tags for a VM
- **create-vm**: create a VM
- **destroy-vm**: destroy a VM (all ephemeral data and metadata are lost)
- **discard-vm-old-image**: discard the previous root image for a VM
- **discard-vm-old-user-data**: discard the previous user data for a VM
- **get-vm-info**: get and show the information for a VM
- **replace-vm-image**: replace the root image for a VM. The old root image is
                        saved. The VM must not be running
- **replace-vm-user-data**: replace the user data for a VM. The old user data is
                        saved
- **restore-vm-image**: restore the previously saved root image for a VM. The VM
                        must not be running
- **restore-vm-user-data**: restore the previously saved user data for a VM
- **start-vm**: start a stopped VM
- **stop-vm**: stop a running VM. All data and metadata are preserved

## Security
The *Hypervisor* restricts RPC access using TLS client authentication.
*Vm-Control* will load certificate and key files from the
`~/.ssl` directory. *Vm-Control* will present these certificates to
the *Hypervisor*. If one of the certificates is signed by a certificate
authority that the *Hypervisor* trusts, the *Hypervisor* will grant access.
