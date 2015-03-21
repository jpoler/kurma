# TODO

### Short Term

- [ ] Address using switch\_root to re-enable pivot\_root for containers.
- [ ] Re-enable user namespace functionality
- [ ] Implement ability to enter a container
- [X] Instrument uid/gid handling for the stage3 exec
- [X] Implement PID 1 system bootstrapping
- [X] Implement "exited" handling for when the stage3 process exits
- [ ] Implement hook calls
- [ ] Review Manager/Container lock handling
- [ ] Implement specifying the container name in the CLI
- [ ] Implement appc isolators for namespaces
- [ ] Implement appc isolators for capabilities
- [ ] Implement appc isolators for cgroups
- [ ] Implement remote image retrieval (with whitelist)
- [ ] Look at a futex for protecting concurrent pivot_root calls.
- [ ] Implement configuring disks
- [ ] Implement bootstrap containers

## Mid Term

- [ ] Module scoping for each environment
- [ ] Configurable configuration datasources

### Exploritory

- [ ] Change management of containers to be separated by process, so the daemon
  doesn't need a direct handle on the container.
- [ ] Investigate authentication with gRPC
