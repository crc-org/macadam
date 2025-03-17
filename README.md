macadam
-------

macadam is a cross-platform command-line tool to create and manage virtual
machines. It runs on Windows, macOS and linux, and uses each OS native
virtualization stacks, that is WSL2 on Windows, Apple’s Virtualization
Framework on macOS and QEMU on linux. macadam is reusing podman-machine code.

Its goal is to be able to run pre-installed disk images and to provide SSH or
serial console access to the guest. Configurability of the virtual machine will
be minimal to keep macadam as easy to use as possible, and to ease the
cross-platform work.

It can currently create a virtual machine (`macadam init`), start it (`macadam start`), stop it (`macadam stop`) and delete it (`macadam rm`).

Due to its podman-machine origin, it currently has multiple requirements on
what is running/installed in the guest. Work is being done to remove these.
We’re currently focused on running unmodified Fedora/EL cloud images, and will
provide links to images which work out of the box when they are available.
