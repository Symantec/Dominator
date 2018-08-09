# vm-control
A utility to manage Virtual Machines (VMs).

The *vm-control* utility creates and manages VMs by communicating with a
*Hypervisor*. It is typically run on a desktop, bastion or build machine.

## Usage
*vm-control* supports several sub-commands. There are many command-line flags
which provide parameters for these sub-commands. The most commonly used
parameters are `-fleetManagerHostname` or `-hypervisorHostname` which specify
either the Fleet Manager or a specific *Hypervisor* to communicate with. The
basic usage pattern is:

```
vm-control [flags...] command [args...]
```

Built-in help is available with the command:

```
vm-control -h
```

Some of the sub-commands available are:

- **become-primary-vm-owner**: become the primary owner of a VM
- **change-vm-owner-users**: change the extra owners for a VM
- **change-vm-tags**: change the tags for a VM
- **create-vm**: create a VM
- **destroy-vm**: destroy a VM (all ephemeral data and metadata are lost)
- **discard-vm-old-image**: discard the previous root image for a VM
- **discard-vm-old-user-data**: discard the previous user data for a VM
- **discard-vm-snapshot**: discard the previous snapshot for a VM
- **get-vm-info**: get and show the information for a VM
- **import-local-vm**: import a local raw VM. This is primarily for debugging
- **import-virsh-vm**: import a local virsh VM. The specified domain name must
                       be a FQDN, which is used to obtain the IP address of the
                       imported VM. The virsh VM must first be shut down. The
                       imported VM is started
- **list-hypervisors**: list healthy Hypervisors in the specified location
- **list-locations**: list locations within the specified top location
- **list-vms**: list the IP addresses for all VMs
- **probe-vm-port**: probe (from its *Hypervisor*) a TCP port for a VM
- **replace-vm-image**: replace the root image for a VM. The old root image is
                        saved. The VM must not be running
- **replace-vm-user-data**: replace the user data for a VM. The old user data is
                        saved
- **restore-vm-from-snapshot**: restore VM volumes from the previous snapshot,
                                discarding current volumes
- **restore-vm-image**: restore the previously saved root image for a VM. The VM
                        must not be running
- **restore-vm-user-data**: restore the previously saved user data for a VM
- **snapshot-vm**: create a snapshot of the VM volumes, discarding previous one
- **start-vm**: start a stopped VM
- **stop-vm**: stop a running VM. All data and metadata are preserved
- **trace-vm-metadata**: trace the requests a VM makes to the metadata service

## Security
The *Hypervisor* restricts RPC access using TLS client authentication.
*vm-control* will load certificate and key files from the
`~/.ssl` directory. *vm-control* will present these certificates to
the *Hypervisor*. If one of the certificates is signed by a certificate
authority that the *Hypervisor* trusts, the *Hypervisor* will grant access.
Most operations only require a certificate that proves *identity*. The
*[Keymaster](https://github.com/Symantec/keymaster)* is a good choice for
issuing these certificates.

## Importing virsh (libvirt) VMs
A libvirt VM may be imported into the *Hypervisor*. This is only supported for
simple VMs with a single network interface and IP address. Once the VM is
*committed* it is removed from the libvirt database and is fully "owned" by the
*Hypervisor*. Importing a VM requires root access on the *Hypervisor* (the
*vm-control* tool will use the `sudo` command if needed).

There are a few simple steps that should be followed to import a VM. In the
example below, the MAC address of the VM to be imported is `52:54:de:ad:be:ef`
and the hostname (DNS entry) is `jump.prod.company.com`. The IP address of the
VM may also be used. In either case, the hostname or IP address provided must
match the libvirt *domain name*.
- log into the VM and determine its MAC address
- run `vm-control import-virsh-vm 52:54:de:ad:be:ef jump.prod.company.com`
- enter `shutdown` at the prompt
- wait for the VM to start and log in and check that it is functioning properly
- respond to the `commit/defer/abandon` prompt:
  - `commit`: the VM is removed from the libvirt database
  - `defer`: the VM is left running on the *Hypervisor*. It may later be
             committed (and the `virsh undefine` command should be used to
             remove it from the libvirt database) or destroyed. This is not
             recommended
  - `abandon`: the VM is deleted from the *Hypervisor* and will need to be
               manually started with the `virsh` command
