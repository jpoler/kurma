# Kurma

Kurma is a next generation execution environment for a containerized host. It is
implemented on the basis that everything is a container.

Kurma is more than just a container manager though, it is intended to be a
framework that allows containers to be managed and orchestrated in way beyond
itself. It is built as a way of managing containers on a single host, but with
an extensible API that could be built on top of to form a cluster of machines.

### Boot Process

TBD

### Container Process

The process for managing containers is broken into a couple different stages
which are responsible for setting up and launching a container.

* Stage 1: This stage is responsible for configuring the filesystem that will
  comprise the container and gather all of the dependencies. It should get them
  all in place so the container is ready to be launched.
* Stage 2: This stage takes care of launching the actual container. It takes
  care of creating the necessary kernel namespaces or joining existing ones, and
  also joins any cgroups that are necessary.
* Stage 3: After stage 2 is complete, it should `exec` the stage 3 binary which
  takes over inside the container and acts as an `init` process. It is used to
  execute the image's start command.

Note that the container process can be extended beyond the base 3 stages through
more explicit coordination between stage 1 and stage 3. For instance, additional
network configuration and management could be done if stage 3 inherited a file
descriptor that allowed communication back with the stage 1 binary. This could
be where stage 3 is another chained process that configures networking before
`exec`ing to launch another binary which takes over the container.

## Code Segments

#### client

The `client` subdirectory represents the code for interacting with a Kurma
host. Currently, it contains the command line interface which talks to a local
Kurma daemon.

#### stage1

The `stage1` subdirectory contains the code for managing the stage1 process,
which handles the set of containers, the RPC functionality, and the operations
for spinning up an individual container.

It also contains the gRPC protobuf definition within the `stage1/client`
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

The `stage3` subdirectory containers the code for the process that acts as the
`init` within the container. It exposes a simple text based RPC for the stage1
process to communicate with it to have it execute commands or check status.
