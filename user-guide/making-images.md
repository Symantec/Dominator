# Making Images
A collection of recipes to make images.

## I just want to push a few files to my machines
You can do this with a *sparse* image. Unlike regular images where files on the
machines may be removed if they are not in the image, with *sparse* images only
the files present in the image are pushed to the machines. All other files are
left alone.

### I don't want to restart any services. Just push the files
In this case, you don't provide a *triggers* file. Normally, *triggers* are used
to tell the **Dominator** that the specified services should be restarted when
certain files are changed.

First, you will need to add your image to the imageserver. Run this command:

```
imagetool -imageServerHostname=imageserver.my.domain add sparse.0 path "" ""
```

This will package up the files in the directory tree `path` and will upload them
to the imageserver `imageserver.my.domain`, creating the image `sparse.0`.
If `path` is a tarfile (extension `.tar`) or a compressed tarfile (extension
`.tar.gz`), then the contents of the tarfile are uploaded.

### Show me what I just created
You can see the new image in the list of images using the following command:

```
imagetool -imageServerHostname=imageserver.my.domain list
```

You can show the contents of the image (the SHA-512 hashes of regular files are
shown) using the following command:

```
imagetool -imageServerHostname=imageserver.my.domain show sparse.0
```

### I changed my mind. I want to restart a service
Imagine that your *sparse* image has the SSH daemon configuration file:
`/etc/ssh/sshd.conf` and you want to restart sshd if the file changes. You will
need to create a file with the following content:

```
[
    {
        "MatchLines": [
            "/etc/ssh/sshd[.]conf",
            "/usr/sbin/sshd"
        ],
        "Service": "sshd"
    }
]
```

This says that whenever the `/etc/ssh/sshd.conf` or `/usr/sbin/sshd` files are
changed (note the regular expression syntax), the `sshd` service should be
restarted (normally the command `service sshd stop; service sshd start` commands
are executed). If this triggers configuration is stored in the file
`/tmp/triggers.sshd` then you could use the following command to create the
image, this time with the triggers configuration:

```
imagetool -imageServerHostname=imageserver.my.domain add sparse.0 path "" /tmp/triggers.sshd
```

## I've realised I want total *Domination*
You've discovered how powerful it is to push a few files to your machines and
have them constantly kept in compliance without having to do further work, and
now you want to control the whole root file-systems of your machines. For this
you will need to create normal images rather than *sparse* images. With normal
images you will need to specify a *filter*. This tells the **Dominator** that
certain files should not be updated. For example, you probably don't want to be
updating the `/etc/fstab` or `/etc/hostname` files, since they will be different
on each machine.

A filter file is a simple text file with a list of filter lines, each being a
regular expression pathname that you want to exclude from changes. An example
filter file may contain:

```
/etc/fstab
/etc/hostname
/tmp/.*
/var/log/.*
/var/mail/.*
/var/spool/.*
/var/tmp/.*
```

If this is contained in the file `/tmp/filter` then you would use the following
command to create an image:

```
imagetool -imageServerHostname=imageserver.my.domain add image.0 path /tmp/filter /tmp/triggers
```

The main difference between this command and the one shown earlier is that
`/tmp/filter` is passed for the name of the filter file rather than an empty
string.

This will package up the files in the directory tree `path` and will upload them
to the imageserver `imageserver.my.domain`, creating the image `image.0`.
If `path` is a tarfile (extension `.tar`) or a compressed tarfile (extension
`.tar.gz`), then the contents of the tarfile are uploaded.
