# TODO

### Short Term

- [ ] Re-enable user namespace functionality
- [ ] Implement ability to enter a container
- [ ] Implement hook calls
- [ ] Review Manager/Container lock handling
- [ ] Implement specifying the container name in the CLI
- [ ] Implement appc isolators for capabilities
- [ ] Implement appc isolators for cgroups
- [ ] Look at a futex for protecting concurrent pivot_root calls.
- [ ] Implement configuring disks
- [ ] Add resource allocation
- [X] Address using switch\_root to re-enable pivot\_root for containers.
- [X] Instrument uid/gid handling for the stage3 exec
- [X] Implement PID 1 system bootstrapping
- [X] Implement "exited" handling for when the stage3 process exits
- [X] Implement appc isolators for namespaces
- [X] Implement remote image retrieval
- [X] Implement bootstrap containers

## Mid Term

- [ ] Kernel module scoping for each environment
- [ ] Configurable configuration datasources
- [ ] Add support for image retrieval through an http proxy
- [ ] Add whitelist support for where to retrieve an image from
- [ ] Add baseline enforcement of certain kernel namespaces, like mount, ipc,
  and pid.

### Exploritory

- [ ] Change management of containers to be separated by process, so the daemon
  doesn't need a direct handle on the container.
- [ ] Investigate authentication with gRPC
