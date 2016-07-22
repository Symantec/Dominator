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
