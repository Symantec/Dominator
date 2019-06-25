Dominator Architectural Overview
================================
Richard Gooch
-------------
	
Overview
========

The **Dominator** is a robust, reliable and efficient system to push operating system images to large fleet of machines (physical or virtual). The design target is that a single system and administrator can manage the content of *at least* 10,000 systems with negligible performance impact on the managed systems, fast response to global changes and nearly atomic changes on those systems.

The **Dominator** takes a radically different approach from to fleet management based on three principles:

-   immutable infrastructure

-   golden “baked” images

-   fast, robust transitions.

Rather than package management, the **Dominator** uses an image management and deployment approach.

Please see [https://github.com/Symantec/Dominator](https://github.com/Symantec/Dominator) for the source code, [design document](README.md), [fact sheet](FactSheet.md) and [user guide](https://github.com/Symantec/Dominator/blob/master/user-guide/README.md).

Dominator Components
====================

The system is comprised of the following components:

-   an **Image Server** which stores images which include:

    -   file-system trees

    -   *trigger lists* which describes which services/daemons should be restarted if specified files change

    -   a *filter* which lists regular expression pathnames which should not be changed

    -   references to *computed files* which are dynamically generated

-   a **M**achine **D**ata**B**ase (**MDB**) which lists all the machines in the fleet and the name of the *required* image that should be on each machine (an enhancement is a secondary *planned* image for each machine)

-   a controller (master) system called the **Dominator** which;

    -   continuously *polls* each **sub** for it’s file-system state

    -   computes differences with the *required image*

    -   directs deviant **subs** to *fetch* files from an **Image Server** and *update* its file-system

-   zero or more **File Generator** servers which serve *computed file* data based on the **MDB** data for a particular machine

-   a slave agent on each machine in the fleet called the **sub**ject daemon (**subd**) which continuously checksum scans the root file-system with built-in rate limiting so as to avoid impacting system workload

The following diagram shows how these components are connected:
