# Getting Started
How to get started with the **Dominator**. For a more in-depth understanding of
the system, please read the
[design document](../design-docs/Dominator/README.md).

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
git clone https://github.com/Cloud-Foundations/Dominator.git
```

This will create a sub-directory called `Dominator` containing the source code.
You will also need some other repositories. You can use `go get` or run the
following commands:

```
git clone https://github.com/Cloud-Foundations/tricorder.git
git clone https://github.com/golang/exp.git
git clone https://github.com/aws/aws-sdk-go.git
git clone https://gopkg.in/fsnotify/fsnotify.v0
```

You can update the local copies of these repositories to the latest version of
the source code by using the following command in each:

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
files, so that trust relationships can be established. This is commonly known as
*TLS client authentication and authorisation*. Clients present a certificate to
servers (endpoints) which proves their identity (authentication) and lists the
RPC methods which they are permitted to call (authorisation). These certificates
give unrestricted access to the listed RPC methods.

This is necessary for operational security, so that you can control and audit
who creates images and who can issue update requests to the *subs*.

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

### Identity-only certificates
Some components (endpoints) permit access to selected RPC methods based on
identity only (these are "public" RPC methods). The same root CA may be used to
sign these identity-only certificates if needed, however this requires you to
build a certificate issuing system for users. An alternate *Identity CA* is
supported. The *[Keymaster](https://github.com/Symantec/keymaster)* is a good
choice for issuing these certificates. This root CA should be placed in
`/etc/ssl/IdentityCA.pem` and will be trusted for identity only (not for
granting unlimited access to RPC methods).

### Creating a certificate+key for [subd](../cmd/subd/README.md)
Using the previously created root certificate+key, you can create and sign a
certificate and key pair for [subd](../cmd/subd/README.md) using the
[make-cert](../scripts/make-cert) utility provided in the source repository.
Use the following command to generate the certificate and key pair:

```
make-cert root subd AUTO subd cmd/subd/required-methods
```

This will create the `subd.pem` and `subd.key.pem` files. These should be copied
to the files `/etc/ssl/subd/cert.pem` and `/etc/ssl/subd/key.pem` respectively
on all machines. As with the CA file, this should also be included in the
installation image that every machine is booted with.

Note how [subd](../cmd/subd/README.md) is given access to a single RPC method:
`ObjectServer.GetObjects`. This is required to allow it to fetch objects.

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
make-cert root Dominator AUTO dominator cmd/dominator/required-methods
```

This will create the `Dominator.pem` and `Dominator.key.pem` files. These should
be copied to the files `/etc/ssl/dominator/cert.pem` and
`/etc/ssl/dominator/key.pem` on the machine where
[dominator](../cmd/dominator/README.md) will run.

Note how (in addition to access to some other RPC methods) the
[dominator](../cmd/dominator/README.md) is given access to call all
[subd](../cmd/subd/README.md) RPC methods. Thus, this is a high value key, as it
gives root level access to your fleet, so you should restrict access to it.

### Creating a certificate+key for [imageserver](../cmd/imageserver/README.md)
Run the following command:

```
make-cert root imageserver AUTO imageserver cmd/imageserver/required-methods
```

This will create the `imageserver.pem` and `imageserver.key.pem` files. These
should be copied to the files `/etc/ssl/imageserver/cert.pem` and
`/etc/ssl/imageserver/key.pem` on the machine where
[imageserver](../cmd/imageserver/README.md) will run.

Note that the list of RPC methods given above allows
[imageserver](../cmd/imageserver/README.md) to replicate images from another
[imageserver](../cmd/imageserver/README.md). If you never plan to enable image
replication (that would be unwise), you could provide an empty list of methods.

### Creating a certificate+key for [imaginator](../cmd/imaginator/README.md)
Run the following command:

```
make-cert root imaginator AUTO imaginator cmd/imaginator/required-methods
```

This will create the `imaginator.pem` and `imaginator.key.pem` files. These
should be copied to the files `/etc/ssl/imaginator/cert.pem` and
`/etc/ssl/imaginator/key.pem` on the machine where
[imaginator](../cmd/imaginator/README.md) will run.

### Creating a certificate+key for [filegen-server](../cmd/filegen-server/README.md)
Run the following command:

```
make-cert root filegen-server AUTO filegen-server ''
```

This will create the `filegen-server.pem` and `filegen-server.key.pem` files.
These should be copied to the files `/etc/ssl/filegen-server/cert.pem` and
`/etc/ssl/filegen-server/key.pem` on the machine where
[filegen-server](../cmd/filegen-server/README.md) will run.

Note how an empty list of RPC methods is specified. This is because
[filegen-server](../cmd/filegen-server/README.md) does not initiate any RPC
connections: it only responds to RPC requests. Thus, it does not need permission
to access any methods. The certificate+key pair is a standard requirement for
every TLS server.

### Creating a certificate+key for [mdbd](../cmd/mdbd/README.md)
Run the following command:

```
make-cert root mdbd AUTO mdbd cmd/mdbd/required-methods
```
This will create the `mdbd.pem` and `mdbd.key.pem` files. These
should be copied to the files `/etc/ssl/mdbd/cert.pem` and
`/etc/ssl/mdbd/key.pem` on the machine where
[mdbd](../cmd/mdbd/README.md) will run.


### Creating a certificate+key pair for a user
Unlike daemons, which require access to a specific set of methods, users require
access to a variety of methods depending on their level of access and your
security policy, so this section will discuss creating these certificate+key
pairs in general terms. To create, run the following command:

```
make-cert root "$LOGNAME" AUTO "$LOGNAME" '$methods'
```

This will create the `$LOGNAME.pem` and `$LOGNAME.key.pem` files.
These should be copied to the `$HOME/.ssl` directory for the user, with matched
`$file.cert` and `$file.key` names. A common convention is to use the names
`$LOGNAME.cert` and `$LOGNAME.key`. The command-line tools such as
[domtool](../cmd/domtool/README.md) and [imagetool](../cmd/imagetool/README.md)
will read all certificate+key pairs from the `$HOME/.ssl` directory.

The forth parameter to `make-cert` is the username that the certificate+key pair
is issued to. This username will be recorded in logs for certain RPC methods and
will be recorded in image metadata when images are created. The entity creating
the certificate+key pairs must therefore be trusted.

The final parameter specifies the list of methods that the user may access. This
may either be a filename containing method names separated by newlines or a
comma separated list. The sections below discuss how to determine the list of
methods if there isn't a `required-methods` file to provide.

#### Discovering methods
The [list-methods](../scripts/list-methods) utility provided in the source
repository will connect to a running server and show the list of methods that
the server supports. To find the list of methods that a server supports, run the
following command:

```
list-methods host:port
```

These are the assigned port numbers:

- [dominator](../cmd/dominator/README.md): 6970
- [filegen-server](../cmd/filegen-server/README.md): 6972
- [imageserver](../cmd/imageserver/README.md): 6971
- [subd](../cmd/subd/README.md): 6969

By knowing the list of methods that servers (daemons) support, you can make an
informed choice about which methods to grant users access to.

#### Common method lists
In this section some common roles are listed, with the corresponding method
lists that are required to perform these roles. When creating certificate+key
pairs the list of methods most be comma separated. The method lists are split
out below for readability:

- Simple image creator (for use in an image build pipeline):
  ```
  ImageServer.AddImage
  ImageServer.CheckImage
  ObjectServer.AddObjects
  ```
- Image creator (can also create derivative images and snapshot machines):
  ```
  ImageServer.AddImage
  ImageServer.CheckImage
  ImageServer.GetImage
  ObjectServer.AddObjects
  ObjectServer.CheckObjects
  Subd.Poll
  ```
- Image administrator (i.e. can delete images, create directories and change
  directory access):
  ```
  ImageServer.*
  ```
- [dominator](../cmd/dominator/README.md) administrator:
  ```
  Dominator.*
  ```
- All-powerful user (full control over all **Dominator** components and
  root-level access to all *subs*):
  ```
  *.*
  ```
