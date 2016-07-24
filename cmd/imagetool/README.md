# imagetool
A utility to add, delete, get and compare images on the
*[imageserver](../imageserver/README.md)*.

The *imagetool* is the most important utility in the **Dominator** system, as it
is used to manage images. *Imagetool* may be run on any machine. It is typically
run on a desktop, bastion or build machine, depending on the sophistication of
your build environment.

## Usage
*Imagetool* supports several sub-commands. There are many command-line flags
which provide parameters for these sub-commands. The most commonly used
parameter is `-imageServerHostname` which specifies which host the *imageserver*
to talk to is running on. The basic usage pattern is:

```
imagetool [flags...] command [args...]
```

Built-in help is available with the command:

```
imagetool -h
```

Some of the sub-commands available are:

- **add**: add an image using a compressed tarfile for image data
- **addi**: add an image using an existing image for image data
- **adds**: add an image using files from a running *subd* for image data (this
            allows "snapshotting" of a golden machine)
- **addrep**: add an image using an existing image and layer files from
              compressed tarfiles on top of existing files
- **check**: check if an image exists
- **chown**: change the owner group of an image directory
- **delete**: delete an image
- **diff**: compare two images
- **get**: get and unpack an image
- **list**: list all images
- **listdirs**: list all directories
- **mkdir**: make a directory
- **show**: show (list) an image

## Security
*[Imageserver](../imageserver/README.md)* restricts RPC access using TLS client
authentication. *Imagetool* will load certificate and key files from the
`~/.ssl` directory. *Imagetool* will present these certificates to
*imageserver*. If one of the certificates is signed by a certificate authority
that *imageserver* trusts, *imageserver* will grant access.
