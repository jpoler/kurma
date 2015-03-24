// Copyright 2015 Apcera Inc. All rights reserved.

package client

import (
	"fmt"
	"os"
	"os/exec"

	_ "github.com/apcera/kurma/stage2"
)

// Launcher is used to encompass the logic needed to launch the stage2
// process. It allows the logic to be centralized here rather than separated
// everywhere that might use it.
type Launcher struct {
	Directory string
	User      string
	Group     string

	NewIPCNamespace     bool
	NewMountNamespace   bool
	NewNetworkNamespace bool
	NewPIDNamespace     bool
	NewUTSNamespace     bool
	NewUserNamespace    bool

	Chroot         bool
	Detach         bool
	HostPrivileged bool

	Environment []string
	Taskfiles   []string

	Stdin  *os.File
	Stdout *os.File
	Stderr *os.File

	postStart []func()
}

// generateArgs is used to generate the necessary command line arguments for the
// stage2 process based on the settings configured on the Launcher. It will
// return the arguments as well as the extra files that need to be passed, such
// as for its stdin, stdout, and stderr.
func (l *Launcher) generateArgs(cmdargs []string) ([]string, []*os.File) {
	// Generate the uid and gid maps for user namespaces
	// uidmap := fmt.Sprintf("0 %d %d\n", c.manager.namespaceUidOffset, c.manager.namespaceUidMaximum)
	// gidmap := fmt.Sprintf("0 %d %d\n", c.manager.namespaceGidOffset, c.manager.namespaceGidMaximum)

	// Initialize the options that will be passed to spawn the container.
	var args []string

	// Directory and filesystem settings
	if l.Directory != "" {
		args = append(args, "--directory", l.Directory)
	}
	if l.Chroot {
		args = append(args, "--chroot")
	}

	// Pass the user and group, if they're set
	if l.User != "" {
		args = append(args, "--user", l.User)
	}
	if l.Group != "" {
		args = append(args, "--group", l.Group)
	}

	// Add applicalble new namespace flags
	if l.NewIPCNamespace {
		args = append(args, "--new-ipc-namespace")
	}
	if l.NewMountNamespace {
		args = append(args, "--new-mount-namespace")
	}
	if l.NewNetworkNamespace {
		args = append(args, "--new-network-namespace")
	}
	if l.NewPIDNamespace {
		args = append(args, "--new-pid-namespace")
	}
	if l.NewUTSNamespace {
		args = append(args, "--new-uts-namespace")
	}
	if l.NewUserNamespace {
		args = append(args, "--new-user-namespace")
	}

	// If user namespaces are to be used, then add the parameter to populate it
	// and the uid and gid maps.
	// if !c.manager.unittestingSkipUserNamespace {
	// 	args = append(args, "--new-user-namespace")
	// }
	// args = append(args, "--uidmap", uidmap)
	// args = append(args, "--gidmap", gidmap)

	// Check for a privileged isolator
	if l.HostPrivileged {
		args = append(args, "--host-privileged")
	}

	// Loop and append all the cgroups taskfiles the container should be in.
	for _, f := range l.Taskfiles {
		args = append(args, "--taskfile", f)
	}

	// Handle any environment variables passed to the app
	for _, env := range l.Environment {
		args = append(args, "--env", env)
	}

	// Optionally detach
	if l.Detach {
		args = append(args, "--detach")
	}

	// Set the file descriptors it should use for stdin/out/err. Note this uses
	// the ExtraFiles on the os/exec below. The file descriptor numbers start from
	// after stderr (2). They are separate from the fd in this process.
	extraFiles := make([]*os.File, 0)

	// Always ensure stdin at a minimum is looped in
	if l.Stdin == nil {
		// Open /dev/null which is used for stdin.
		l.Stdin, _ = os.OpenFile("/dev/null", os.O_RDONLY, 0)
		l.postStart = append(l.postStart, func() { l.Stdin.Close() })
	}
	args = append(args, "--stdinfd", "3")
	extraFiles = append(extraFiles, l.Stdin)

	if l.Stdout != nil {
		args = append(args, "--stdinfd", fmt.Sprintf("%d", len(extraFiles)+3))
		extraFiles = append(extraFiles, l.Stdout)
	}
	if l.Stderr != nil {
		args = append(args, "--stderrfd", fmt.Sprintf("%d", len(extraFiles)+3))
		extraFiles = append(extraFiles, l.Stderr)
	}

	// Setup the command line to have it invoke the container's process.
	args = append(args, "--")
	args = append(args, cmdargs...)

	return args, extraFiles
}

// Run will launch the stage2 binary with the desired settings and execute the
// specified command. It will return once the stage2 has been started.
func (l *Launcher) Run(cmdargs []string) error {
	args, extraFiles := l.generateArgs(cmdargs)

	// Create and initialize the spawnwer.
	cmd := exec.Command(os.Args[0], args...)
	cmd.ExtraFiles = extraFiles
	if l.Stdout != nil {
		cmd.Stdout = l.Stdout
	}
	if l.Stderr != nil {
		cmd.Stderr = l.Stderr
	}

	// The spawner keys off this environment variable to know when it is supposed
	// to run and take over execution.
	cmd.Env = []string{
		"SPAWNER_INTERCEPT=1",
	}

	// Start the container.
	if err := cmd.Start(); err != nil {
		return err
	}

	// Run any postStart funcs, just to cleanup
	for _, f := range l.postStart {
		f()
	}

	// FIXME return process state?
	return nil
}
