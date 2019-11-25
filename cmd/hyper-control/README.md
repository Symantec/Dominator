# hyper-control
A utility to manage SmallStack Hypervisors.

The *hyper-control* utility manages [Hypervisors](../hypervisor/README.md). It
is typically run on a desktop or bastion machine. Please read the
[SmallStack design document](https://bit.ly/SmallStack) to understand the
architecture.

## Usage
*hyper-control* supports several sub-commands. There are many command-line flags
which provide parameters for these sub-commands. The most commonly used
parameters are `-fleetManagerHostname` or `-hypervisorHostname` which specify
either the Fleet Manager or a specific *Hypervisor* to communicate with. At
startup, *hyper-control* will read parameters from the
`~/.config/hyper-control/flags.default` and
`~/.config/hyper-control/flags.extra` files. These are simple `name=value`
pairs. The basic usage pattern is:

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
                   is only required if a *Fleet Manager* is not available
- **add-subnet**: manually add a subnet to a specific *Hypervisor*. This is only
                  required if a *Fleet Manager* is not available
- **change-tags**: change the tags for a specific *Hypervisor*
- **get-machine-info**: get information for a specific *Hypervisor*
- **get-updates**: get and show a continuous stream of updates from a
                   *Hypervisor* or *Fleet Manager*. This is primarily for
                   debugging
- **installer-shell**: start a remote shell (via SRPC) to the installer running
                       on a machine
- **make-installer-iso**: make a bootable installation ISO (CD-ROM) image for a
                          machine
- **move-ip-address**: move a (free) IP address to a specific *Hypervisor*
- **netboot-host**: temporarily enable PXE-based network booting and installing
                    for a machine
- **netboot-machine**: temporarily enable PXE-based network booting for a
                       machine
- **netboot-vm**: create a temporary VM and install with PXE booting. This is
                  for debugging physical machine installation
- **reinstall**: reinstall the local machine. This erases all data
- **remove-excess-addresses**: remove free addresses for a specific *Hypervisor*
                               above the specified limit
- **remove-ip-address**: remove a (free) IP address from a specific *Hypervisor*
- **remove-mac-address**: remove a (free) MAC address from a specific
                          *Hypervisor*
- **rollout-image**: safely roll out specified image to all *Hypervisors* in a
                     location
- **write-netboot-files**: write the configuration files for installing a
                           machine. This is primarily for debugging

## Security
The *Hypervisor* restricts RPC access using TLS client authentication.
*hyper-control* will load certificate and key files from the
`~/.ssl` directory. *hyper-control* will present these certificates to
the *Hypervisor* (or *Fleet Manager*). If one of the certificates is signed by a
certificate authority that the *Hypervisor* trusts, the *Hypervisor* will grant
access.

## Installing Hypervisors
The *hyper-control* tool may be used to install *Hypervisors* (OS+Hypervisor) on
physical machines. This requires that information about machines and subnets is
recorded in the topology (usually in Git), which is obtained from the
[Fleet Manager](../fleet-manager/README.md). *Hypervisors* may be installed by
PXE booting an installer, booting from a custom ISO image or installing over a
running system. Please read the
[Machine Birthing design document](../../design-docs/MachineBirthing/README.md)
which describes the principles of installing physical machines.

### Network (PXE) Installation
This is the most common method of installing. If there is at least one working
Hypervisor on the same subnet, you can PXE (network) boot. The *hyper-control*
tool is used to automatically select a *Hypervisor* to configure as a PXE server
(it creates a temporary DHCP lease and will serve configuration files via TFTP).

The following options must be provided:
- `fleetManagerHostname`
- `imageServerHostname`
- `installerImageStream`

You may want to increase the `-netbootTimeout` option if the machine takes a
long time to boot. Run the following command:

```
hyper-control netboot-host $target_host
```

Then, initiate a PXE boot for the target machine. It should boot the installer
image, configure and install the machine and then reboot into the new OS. In
principle, multiple machines can be installed in parallel, one machine for each
time the above command is run.

### Installing from an ISO (CD-ROM) image
If there is no working *Hypervisor* on the subnet and if there is no DHCP relay
configured to forward DHCP requests to a *Hypervisor* on another subnet, then
another option is to install the machine using a custom ISO (CD-ROM) image. The
ISO image can be written to a CD-ROM, a USB memory drive or served by a NFS/SMB
server (this requires that the machine BIOS supports booting from remote media).

As above, the same required options must be provided. The following command will
generate a custom ISO image for the target machine. Note that the networking
configuration for the machine is baked into the ISO, so the ISO is only good for
one machine.

```
hyper-control make-installer-iso $target_host $destdir
```

This will create a custom ISO for this machine in the specified directory, with
both hostname and IP address file entries.

### Self installing a Hypervisor
If there is a Linux OS already installed and running on a machine, you can use
the *hyper-control* utility to install a new OS+Hypervisor. This uses the
`kexec` utility to boot directly into the new kernel, skipping the BIOS. This is
a good way of wiping and getting a fresh install.

As above, the same required options must be provided. Run this command:

```
hyper-control reinstall
```

This method is much faster, since it skips rebooting via the BIOS the first.
Remember: it wipes all data on the machine!

## Upgrading Hypervisors
Upgrading *Hypervisors* is done using the *hyper-control* tool. This sets the
`RequiredImage` tag on *Hypervisors* in the specified location. The rollout is
controlled, starting slow and gaining speed as *Hypervisors* complete upgrades
and remain healthy. The upgrade may include a new kernel, in which case the
machine will be rebooted as part of the upgrade, thus taking several minutes for
the per-machine upgrade+health check cycle to complete (compared to less than
one minute for most upgrades). The actual upgrades are performed by the
[dominator](../dominator/README.md) which reads the tags to determine the image
to push to the machine. Once the rollout is complete, the new tags are saved by
committing to the Git repository containing the topology.

The following options must be provided:
- `fleetManagerHostname`
- `imageServerHostname`
- `location`
- `topologyDir`

To rollout a new *Hypervisor* image to all the *Hypervisors* in the specified
location, run a command like this:

```
hyper-control rollout-image $image_name
```
