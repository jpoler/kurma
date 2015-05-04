# Kurma

Kurma is a next generation execution environment for a containerized host. Kurma is built on the notion that everything is a container.

Kurma is more than just a container manager. Kurma is an operating system that allows containers to be managed and orchestrated by other processes. Kurma can be used to manage containers on a single host, and has an extensible API that can be used to create and manage containers across clusters of machines.

### Building Kurma

See the [KurmaOS Repo](https://github.com/apcera/kurmaos/blob/master/README.md#build-process) for instructions on how to build Kurma.

### Downloading Kurma

The latest release images can be found under [Releases](https://github.com/apcera/kurma/releases).

### Container Process

The process for managing containers comprises three stages that are responsible for setting up and launching a containerized host.

* Stage 1: This stage is responsible for configuring the filesystem that will
  comprise the container and gather all the dependencies. This stage should get
  all the dependencies in place so the container is ready to be launched.
* Stage 2: This stage takes care of launching the actual container.
  This stage takes care of creating the necessary kernel namespaces or joining existing ones,
  and also joins any cgroups that are necessary.
* Stage 3: After stage 2 is complete, stage 3 will `exec` the stage 3 binary which
  takes over inside the container and acts as an `init` process. It is used to
  execute the image's start command.

You can extend the containerization process beyond these 3 stages through
more explicit coordination between stage 1 and stage 3. For instance, additional
network configuration and management can be done if stage 3 inherits a file
descriptor that allows communication back to the stage 1 binary. This can
be where stage 3 is another chained process that configures networking before
`exec`ing to launch another binary which takes over the container.

## Code Segments

#### client

The `client` subdirectory represents the code for interacting with a Kurma
host. Currently, it contains the command line interface which talks to a local
Kurma daemon, and the remote API broker which allows external access from
`kurma-cli` using the `-H` flag.

#### stage1

The `stage1` subdirectory contains the code for managing the stage1 process,
which handles the set of containers, the RPC functionality, and the operations
for spinning up an individual container.

This code also contains the gRPC protobuf definition within the `stage1/client`
directory. This should be used by any client that wishes to interact with a
Kurma host.

#### stage2

The `stage2` subdirectory contains the code for handling container creation at
the kernel level. It is intended to be referenced into a binary, and when the
binary needs to setup a container, it will call itself with a specific intercept
environment variable set. This will trigger it to take over and to handle the
container setup. It is implemented in C, but still build with `go build` and be
included in a normal binary.

#### stage3

The `stage3` subdirectory contains the code for the process that acts as the
`init` within the container. This code exposes a simple text based RPC for the stage1
process to communicate with it to have it execute commands or check status.

## Related Repositories

- [KurmaOS](https://github.com/apcera/kurmaos)
- [Kurmaos-overlay](https://github.com/apcera/kurmaos-overlay)
- [logray](https://github.com/apcera/logray)
- [util](https://github.com/apcera/util)
- [termtables](https://github.com/apcera/termtables)
