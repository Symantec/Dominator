# Getting Started
How to get started with the **Dominator**. For a more in-depth understanding of
the system, please read the
[design document](https://docs.google.com/document/d/1fiDFY9T0mc5zMcFqPvmQcD90T4WQr8wpiMTHVDjTkOE/pub).

## Overview
The **Dominator** is a system for pushing machine images (kernel, operating
system and application stack) to large numbers of machines and continuously
keeping them in compliance with their *required image*. The system is comprised
of several components:

- [dominator](../cmd/dominator/README.md): a daemon which constantly polls the
  file-system state of each machine in the fleet
- [filegen-server](../cmd/filegen-server/README.md): a daemon which computes
  file data on request from the [dominator](../cmd/dominator/README.md) (this is
  typically used to provide machine-specific files such as certificates and
  configuration files)
- [imageserver](../cmd/imageserver/README.md): a daemon which hosts the images
  registered with the system and responds to management and data requests
- [imagetool](../cmd/imagetool/README.md): a utility to manage images hosted on
  an [imageserver](../cmd/imageserver/README.md) (i.e. add image, delete image)
- [mdbd](../cmd/mdbd/README.md): a companion daemon for the
  [dominator](../cmd/dominator/README.md) which interfaces to different
  implementations of a Machine Data Base, providing the
  [dominator](../cmd/dominator/README.md) with a manifest of machines in the
  fleet and their corresponding *RequiredImage* data
- [subd](../cmd/subd/README.md): a daemon which runs on every machine in the
  fleet which constantly scans the local root file-system and responds to
  requests from the [dominator](../cmd/dominator/README.md) to poll the state of
  the file-system, fetch files from the
  [imageserver](../cmd/imageserver/README.md) and perform updates of the
  file-system

## Building from source code
The software was developed for and tested with Linux. Most of the code should
also compile with MacOS, except [subd](../cmd/subd/README.md), which depends on
the advanced namespace management that is unique to Linux.

First, grab a copy of the source code, using the following command:

```
git clone https://github.com/Symantec/Dominator.git
```

This will create a sub-directory called `Dominator` containing the source code.
You can update the directory to the latest version of the source code by using
the following command:

```
git pull
```

You will need [go1.7](https://golang.org/dl/) or newer to to compile the
**Dominator** software. Create the `$HOME/go/bin` directory and set the `GOPATH`
environment variable to `$HOME/go`. The following command will compile the
software:

```
make
```

The compiled binaries will be available in `$HOME/go/bin`.

## Making certificates
In order for the various components of the **Dominator** to communicate, each
component will need SSL certificate+key pairs and/or Certificate Authority (CA)
files, so that trust relationships can be established. This is necessary for
operational security, so that you can control and audit who creates images and
who can issue update requests to the *subs*.

### Creating a root CA
You will need to create a CA which will be used as the root of trust for all the
components. It is recommended that you do not use a commercial CA, since the CA
can issue certificates that effectively give root-level access to your machines.
Additionally, the trust relationships between the **Dominator** components are
only needed within your infrastructure, so there is no benefit to a public
(commercial) CA.

Below are sample commands which would produce a CA with 3 year expiration:

```
openssl genpkey -algorithm RSA -out root.key.pem -pkeyopt rsa_keygen_bits:4096
openssl req -new -key root.key.pem -days 1096 -extensions v3_ca -batch -out root.csr -utf8 -subj '/CN=Dominator'
openssl x509 -req -sha256 -days 1096 -in root.csr -signkey root.key.pem -set_serial 1 -out root.pem
chmod a+r root.pem
```

This root CA will be used to sign all the other certificate+key pairs. In
addition, the `root.pem` file that is created should be copied to
`/etc/ssl/CA.pem` on every machine which runs a daemon component of the
**Dominator** ([dominator](../cmd/dominator/README.md),
[filegen-server](../cmd/filegen-server/README.md),
[imageserver](../cmd/imageserver/README.md) and [subd](../cmd/subd/README.md)).
The simplest approach is to copy this file to all machines and/or including it
in the installation image that every machine is booted with.

### Creating a certificate+key for [subd](../cmd/subd/README.md)
Using the previously created root certificate+key, you can create and sign a
certificate and key pair for [subd](../cmd/subd/README.md) using the
[make-cert](../scripts/make-cert) utility provided in the source repository.
Use the following command to generate the certificate and key pair:

```
make-cert root subd AUTO subd 'ObjectServer.GetObjects'
```

This will create the `subd.pem` and `subd.key.pem` files. These should be copied
to the files `/etc/ssl/subd/cert.pem` and `/etc/ssl/subd/key.pem` respectively
on all machines. As with the CA file, this should also be included in the
installation image that every machine is booted with.

### Adding [subd](../cmd/subd/README.md) to all your machines and boot image
Before moving onto making other certificates, let's finish off the steps to get
[subd](../cmd/subd/README.md) onto all your machines and into your boot image,
so that it will run everywhere. You will need to copy `$HOME/go/bin/subd` and
`$HOME/go/bin/run-in-mntns` to your machines. The recommended location is
`/usr/local/sbin`. You will also need to copy the appropriate boot script from
the [init scripts](../init.d) directory, and run the OS-specific command to
install or activate the boot script.

### Creating a certificate+key for [dominator](../cmd/dominator/README.md)
Run the following command:

```
make-cert root Dominator AUTO dominator \
    'ObjectServer.AddObjects,Subd.*,ImageServer.GetImage,FileGenerator.Connect'
```

This will create the `Dominator.pem` and `Dominator.key.pem` files. These should
be copied to the files `/etc/ssl/dominator/cert.pem` and
`/etc/ssl/dominator/key.pem` on the machine where
[dominator](../cmd/dominator/README.md) will run. This is a high value key, as
it gives root level access to your fleet, so you should restrict access to it.
