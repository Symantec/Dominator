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
