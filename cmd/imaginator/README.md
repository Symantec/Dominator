# imaginator
The *imaginator* daemon builds OS and OS+application images which are specified
by *[image manifests](../../user-guide/image-manifest.md)*.

## Status page
The *imaginator* provides a web interface on port `6975` which shows a status
page, links to built-in dashboards and access to performance metrics and logs.
If *imaginator* is running on host `myhost` then the URL of the main status page
is `http://myhost:6975/`. An RPC over HTTP interface is also provided over the
same port.


## Startup
*Imaginator* is started at boot time, usually by one of the provided
[init scripts](../../init.d/). The *imaginator* process is baby-sat by the init
script; if the process dies the init script will re-start it. It may be stopped
with the command:

```
service imaginator stop
```

which also kills the baby-sitting init script. It may be started with the
comand:

```
service imaginator start
```

There are many command-line flags which may change the behaviour of
*imaginator*, generally with defaults which should be adequate for most
deployments. Built-in help is available with the command:

```
imaginator -h
```

### Key configuration parameters
The init script reads configuration parameters from the
`/etc/default/imaginator` file. The following is the minimum likely set of
parameters that will need to be configured.

The `CONFIGURATION_URL` variable specifies the main configuration URL, which may
be a local file or a HTTP URL. It is recommended to specify a HTTP URL which
references a file in a Git repository.

The `IMAGE_SERVER_HOSTNAME` variable specifies the hostname where the
*[imageserver](../imageserver/README.md)* is running. This hostname must be
resolvable by the *imaginator*.

The `VARIABLES_FILE` variable specifies an optional filename from which to read
special variables which may be used for variable expansion. These are typically
used to store secrets for accessing Git repositories which require
authentication. Each line should contain a single `NAME=Value` entry.

## Security
RPC access is restricted using TLS client authentication. *Imaginator* expects
a root certificate in the file `/etc/ssl/CA.pem` which it trusts to sign
certificates which grant access. It also requires a certificate and key which
grant it the ability to add and get images and objects from the *imageserver*.
These should be in the files `/etc/ssl/imaginator/cert.pem` and
`/etc/ssl/imaginator/key.pem`, respectively.

## Control
The *[builder-tool](../builder-tool/README.md)* utility may be used to request
the *imaginator* to build an image.

## Main Configuration URL
The main configuration URL points to a JSON encoded file that describes all the
*image streams* and how to build them. The top-level JSON object should contain
the following fields:
- `BootstrapStreams`: a table of *bootstrap image* stream names and their
  		      respective configurations
- `ImageStreamsToAutoRebuild`: an array of *image stream* names that should be
  			       rebuilt periodically, in addition to *bootstrap
			       streams* that are always rebuilt automatically
- `ImageStreamsUrl`: the URL of a configuration file containing a list of all
  		     the user-defined *image streams*
- `PackagerTypes`: a table of *packager type* names (i.e. `deb` and `rpm`) and
  		   their respective configurations

A [sample configuration file](conf.json) is provided which may be modified to
suit your environment. This is a fully working configuration and only requires
modification of the location of the package repositories and the
`ImageStreamsUrl` for your custom *image streams*.

### Bootstrap Streams configuration
Each *bootstrap stream* is configured by a JSON object with the following
fields:
- `BootstrapCommand`: an array of strings containing the bootstrap script to run
  		      to generate the image contents (typically `debootstrap`
		      and `yumbootstrap`). The `$dir` variable expands to the
		      root directory of the image to build
- `FilterLines`: an array of regular expressions matching files which should not
  		 be included in the image
- `PackagerType`: the name of the packager type to use

### Image Streams URL
This is a JSON encoded configuration file listing all the user-defined *image
streams*. It contains a top-level `Streams` field which in turn contains a table
of *image stream* names and their respective configurations. The configuration
for an *image stream* is a JSON object with the following fields:
- `ManifestUrl`: the URL of a Git repository containing the
  		 *[image manifest](../../user-guide/image-manifest.md)* for the
		 image. The special URL scheme `dir` points to a local directory
		 tree. Variables specified in the `VARIABLES_FILE` will be
		 expanded here
- `ManifestDirectory`: the directory within the Git repository containing the
  		 *[image manifest](../../user-guide/image-manifest.md)* for the
		 image. If unspecified, the top-level directory in the
		 repository is used. The `$IMAGE_STREAM` variable expands to the
		 name of the *image stream*

An [example configuration file](streams.json) is provided. Note the use of
variables in different places.

### Packager Types
Each *packager type* is configured by a JSON object with the following fields:
- `CleanCommand`: an array of strings containing the command to run when
  		  cleaning up packager debris
- `InstallCommand`: an array of strings containing the command to run when
  		    installing packages. The name of the package to be installed
		    is appended to the command-line
- `ListCommand`: a JSON object defining how to list packages which are
  		 installed. This JSON object contains the following fields:
  - `ArgList`: an array of strings containing the command to run when listing
    	       installed packages
  - `SizeMultiplier`: an optional multiplier to apply to the output of the
    		      listing command to convert the size result to Bytes
- `UpdateCommand`: an array of strings containing the command to run when
  		   updating the package database
- `UpgradeCommand`: an array of strings containing the command to run when
  		    upgrading the already installed packages
- `Verbatim`: an array of strings containing commands to run before any of the
  	      above defined commands

These parameters are used to generate a `/bin/generic-packager` script which is
used as an interface to the native OS packaging tools.
