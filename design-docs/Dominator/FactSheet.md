Dominator Fact Sheet
====================
Richard Gooch
-------------

Operational Principles
======================

**Dominator** pushes OS images to machines (physical or virtual). The contents of each machine is continuously checksum-scanned and compared with the *required image* for that machine. Any time deviations are detected they are automatically corrected. Deviations are not only files which are different on the machine compared to the image, but also files which are missing on the machine and files which are on the machine but missing from the image. All such deviations are automatically corrected. **Dominator** continually drives machines into compliance.

By manipulating the *required image* attribute in a Machine DataBase, rollouts and deployments can be effected. **Dominator** is agnostic about the MDB technology used and the rollout technology. It’s sole purpose is to drive machines into the desired state, which it fetches from the MDB.

An agent runs on every machine (subd) which performs continuous scanning of the root file-system, and the dominator (master) continuously *polls* each sub so that it can compare the root file-system contents with the *required image*. The subd does not generate network traffic unless directed to by the dominator.

Since **Dominator** has visibility into the contents of the root file-system of every machine, it may be leveraged to detect some classes of intrusion.

Issues Addressed
================

This section covers some common questions and concerns.

Performance Impact of subd
--------------------------

The subd rate limits the scanning of the root file-system. It detects if the scanning requires reading from the storage media and if so, limits itself to 2% of the media bandwidth. In case of emergency (system overload), the media scanning can be reduced from 2% to 0.1% in order to shed nonessential services.

In some cases the root file-system will fit into the page cache, and no I/O is required. In this case scanning is limited by the time taken to compute the file checksums, and the scanning is CPU-bound. By default, subd runs the scanning code at nice level 15. If the machine is otherwise idle, the scanning will consume a full CPU. If the machine is busy with other work, then subd will get up to 8% of a single CPU.

At any time, if subd scanning leads to media I/O, it will immediately switch back to self rate limiting (default 2% of the media speed).

In addition to rate limits for scanning, subd rate limits *fetching* files from the object server. Again, this is to prevent saturating the network and interfering with the normal machine workload. By default, it is limited to 10% of the network bandwidth. Note that *fetching* is only performed if the dominator has detected deviations which must be corrected. In normal operation, less than 0.1% of the time is spent *fetching* files.

Updates
=======

Machines are updated in four phases:

-   *Fetching* of files to an object store private to subd. This does not affect any running services

-   Stopping any services which have changed configuration files, libraries or binaries

-   Fast atomic moves of the fetched files into the running file-system. This typically takes 10s to 100s of milliseconds

-   Starting any services which have changed configuration files. This stop/start approach ensures that new services are started, existing services are restarted and old (removed) services are stopped

Overall, the period where a machine is in a transition state is a small fraction of a second, which reduces the chances of problems during the change. This is several orders of magnitude faster than other update systems such as Puppet.

Access Control
==============

All components of the **Dominator** system have access control for the RPC methods. The implementation is RPC over HTTP, and is secured with client authentication using TLS v1.2 or later. In other words, in order to make any RPC call, a client must present a valid certificate to the server, which checks that the certificate was signed by a trusted CA. Certificates include the username to whom the certificate was issued (so that sensitive operations can be logged and attributed) and a optional list of RPC methods that are permitted. This provides effective separation of powers: a user may have upload permissions to the imageserver, but only the dominator can issue requests to change machines.

Links
=====

**Dominator** is an Open Source project hosted on the [Cloud-Foundations/Dominator](https://github.com/Cloud-Foundations/Dominator) page at [GitHub](https://www.github.com/). There is a [design document](README.md), [architectural overview]ArchitecturalOverview.md() and [user guide](https://github.com/Cloud-Foundations/Dominator/blob/master/user-guide/README.md) available.
