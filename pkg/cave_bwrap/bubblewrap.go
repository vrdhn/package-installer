package cave_bwrap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"pi/pkg/cave"
	"pi/pkg/cave/config"
	"pi/pkg/pkgs"
)

// Arguments to bubblewrap.

type BindType = string

const (
	// SRC DEST Bind mount the host path SRC on DEST
	BIND BindType = "--bind"
	// SRC DEST Equal to --bind but ignores non-existent SRC
	BIND_TRY BindType = "--bind-try"
	// SRC DEST Bind mount the host path SRC on DEST, allowing device access
	BIND_DEV BindType = "--dev-bind"
	// SRC DEST Equal to --dev-bind but ignores non-existent SRC
	BIND_DEV_TRY BindType = "--dev-bind-try"
	// SRC DEST Bind mount the host path SRC readonly on DEST
	BIND_RO BindType = "--ro-bind"
	// SRC DEST Equal to --ro-bind but ignores non-existent SRC
	BIND_RO_TRY BindType = "--ro-bind-try"
	// FD DEST Bind open directory or path fd on DEST
	BIND_FD BindType = "--bind-fd"
	// FD DEST Bind open directory or path fd read-only on DEST
	BIND_RO_FD BindType = "--ro-bind-fd"
	// FD DEST Copy from FD to file which is bind-mounted on DEST
	BIND_DATA BindType = "--bind-data" // Note: space removed to match usage
	// FD DEST Copy from FD to file which is readonly bind-mounted on DEST
	BIND_DATA_RO BindType = "--ro-bind-data" // Note: space removed
)

type bindPair struct {
	cave_target string
	host_source string
	bindType    BindType
}

type Bubblewrap struct {
	// keep this sorted by cave_target
	binds map[string]bindPair

	// environment variables to set
	envs map[string]string

	// the full path to executable, like /usr/bin/bash
	executable string

	// Note that first argument is NOT argv0
	cmdline []string
}

func Create() *Bubblewrap {

	envs := make(map[string]string)
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) == 2 {
			envs[pair[0]] = pair[1]
		}
	}

	return &Bubblewrap{
		executable: "",
		cmdline:    nil,
		binds:      make(map[string]bindPair),
		envs:       envs,
	}

}

func (b *Bubblewrap) AddBind(typ BindType, path string) {
	b.binds[path] = bindPair{path, path, typ}
}

func (b *Bubblewrap) AddBinds(typ BindType, paths ...string) {
	for _, path := range paths {
		b.binds[path] = bindPair{path, path, typ}
	}
}

func (b *Bubblewrap) AddMapBind(typ BindType, hostpath string, cavepath string) {
	b.binds[cavepath] = bindPair{cavepath, hostpath, typ}
}

func (b *Bubblewrap) AddEnvFirst(name string, entry string) {
	val := b.envs[name]

	parts := strings.Split(val, ":")
	newParts := []string{entry}
	for _, p := range parts {
		if p != "" && p != entry {
			newParts = append(newParts, p)
		}
	}
	b.envs[name] = strings.Join(newParts, ":")
}

func (b *Bubblewrap) SetCommand(executable string, cmdline ...string) {
	b.executable = executable
	b.cmdline = cmdline
}

// Cmd returns an *exec.Cmd to run the bubblewrap process.
func (b *Bubblewrap) Cmd() *exec.Cmd {
	execPath := "/usr/bin/bwrap"

	// Construct argument list
	args := []string{}

	// Ensure executable is set in argv0 if not empty
	if b.executable != "" {
		args = append(args, "--argv0", b.executable)
	}

	for _, key := range sortedKeys(b.binds) {
		bind := b.binds[key]
		args = append(args, bind.bindType, bind.host_source, bind.cave_target)
	}

	// Finally add the command and arguments.
	if b.executable != "" {
		args = append(args, "--", b.executable)
		args = append(args, b.cmdline...)
	}

	cmd := exec.Command(execPath, args...)

	envList := make([]string, 0, len(b.envs))
	for _, k := range sortedKeys(b.envs) {
		envList = append(envList, fmt.Sprintf("%s=%s", k, b.envs[k]))
	}
	cmd.Env = envList

	return cmd
}

// Spawn runs the command and waits for it to finish.
func (b *Bubblewrap) Spawn() error {
	cmd := b.Cmd()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// Exec replaces the current process with the given command.
// It uses syscall.Exec which does not return on success.
func Exec(cmd *exec.Cmd) error {
	// syscall.Exec requires the binary path to be the first argument if we want it to be argv[0],
	// but actually the first argument to syscall.Exec is the path, and the second is the slice of arguments
	// where the first element is usually the program name.
	// exec.Command sets cmd.Args[0] to the command name/path.

	// Ensure we have an absolute path if possible, though exec.Command might have resolved it?
	// exec.Command only looks up path if it contains no separators.
	// However, Cmd() here sets path to "/usr/bin/bwrap" which is absolute.

	// Check if Path is set (it should be from exec.Command)
	if cmd.Path == "" {
		return fmt.Errorf("command path is empty")
	}

	return syscall.Exec(cmd.Path, cmd.Args, cmd.Env)
}

func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ResolveLaunch implements cave.Backend

func (b *Bubblewrap) ResolveLaunch(ctx context.Context, c *cave.Cave, settings *config.CaveSettings, symlinks []pkgs.Symlink, command []string) (*exec.Cmd, error) {

	// 1. Setup basic binds

	b.AddBind(BIND_RO, "/usr")

	b.AddBind(BIND_RO, "/bin")

	b.AddBind(BIND_RO, "/lib")

	if _, err := os.Stat("/lib64"); err == nil {

		b.AddBind(BIND_RO, "/lib64")

	}

	b.AddBind(BIND_RO, "/etc/alternatives") // For java etc

	b.AddBind(BIND_RO, "/etc/resolv.conf")

	// 2. Setup Workspace

	b.AddBind(BIND, c.Workspace)

	// 3. Setup Home

	if err := os.MkdirAll(c.HomePath, 0755); err != nil {

		return nil, fmt.Errorf("failed to create home directory: %w", err)

	}

	b.AddMapBind(BIND, c.HomePath, os.Getenv("HOME"))

	// 3.1 Create symlinks for packages
	if err := pkgs.CreateSymlinks(c.HomePath, symlinks); err != nil {
		return nil, fmt.Errorf("failed to create symlinks: %w", err)
	}

	// 3.2 Ensure .local/bin is in PATH
	b.AddEnvFirst("PATH", filepath.Join(os.Getenv("HOME"), ".local", "bin"))

	// 4. Set environment variables from settings

	for k, v := range settings.Env {

		b.envs[k] = v

	}

	// 5. Set command

	if len(command) > 0 {

		b.SetCommand(command[0], command[1:]...)

	} else {

		shell := os.Getenv("SHELL")

		if shell == "" {

			shell = "/bin/bash"

		}

		b.SetCommand(shell)

	}

	// 6. Return command

	return b.Cmd(), nil

}
