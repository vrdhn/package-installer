package bubblewrap

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
	"pi/pkg/config"
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

	// Virtual filesystems
	PROC  BindType = "--proc"
	DEV   BindType = "--dev"
	TMPFS BindType = "--tmpfs"
	DIR   BindType = "--dir"
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

	// environment variables to unset
	unsets []string

	// flags like --unshare-pid
	flags []string

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

func (b *Bubblewrap) AddFlag(flag string) {
	b.flags = append(b.flags, flag)
}

func (b *Bubblewrap) UnsetEnv(name string) {
	b.unsets = append(b.unsets, name)
}

func (b *Bubblewrap) AddVirtual(typ BindType, path string) {
	// We use the same map but since hostpath is not needed for these,
	// we'll handle them specially in Cmd() or just store them separately.
	// For simplicity, let's just store them in binds with empty host_source.
	b.binds[path] = bindPair{path, "", typ}
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

	// Flags
	args = append(args, b.flags...)

	// Binds and Virtual FS
	for _, key := range sortedKeys(b.binds) {
		bind := b.binds[key]
		if bind.host_source == "" {
			args = append(args, bind.bindType, bind.cave_target)
		} else {
			args = append(args, bind.bindType, bind.host_source, bind.cave_target)
		}
	}

	// Environment
	for _, k := range sortedKeys(b.envs) {
		args = append(args, "--setenv", k, b.envs[k])
	}
	for _, k := range b.unsets {
		args = append(args, "--unsetenv", k)
	}

	// Finally add the command and arguments.
	if b.executable != "" {
		args = append(args, "--", b.executable)
		args = append(args, b.cmdline...)
	}

	cmd := exec.Command(execPath, args...)
	// Note: We don't set cmd.Env here because we use --setenv inside bwrap
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
func (b *Bubblewrap) ResolveLaunch(ctx context.Context, cfg config.ReadOnly, c *cave.Cave, settings *cave.CaveSettings, prep *pkgs.Result, command []string) (*exec.Cmd, error) {
	// Internal home path inside the sandbox
	internalHome := cfg.GetHostHome()

	// 1. Isolation & Flags
	b.AddFlag("--unshare-pid")
	b.AddFlag("--new-session")
	b.AddFlag("--die-with-parent")

	// 2. Base system bindings (read-only)
	b.AddBind(BIND_RO, "/usr")
	b.AddBind(BIND_RO, "/lib")
	if _, err := os.Stat("/lib64"); err == nil {
		b.AddBind(BIND_RO, "/lib64")
	}
	b.AddBind(BIND_RO, "/bin")
	b.AddBind(BIND_RO, "/sbin")
	b.AddBind(BIND_RO, "/opt")
	b.AddBind(BIND_RO, "/etc")

	// 3. Virtual filesystems
	b.AddVirtual(PROC, "/proc")
	b.AddVirtual(DEV, "/dev")
	b.AddVirtual(TMPFS, "/tmp")
	b.AddVirtual(TMPFS, "/run")

	// 4. Bind global cache directory (readonly)
	if prep.CacheDir != "" {
		caveCache := filepath.Join(c.HomePath, ".cache", "pi")
		if err := os.MkdirAll(caveCache, 0755); err != nil {
			return nil, fmt.Errorf("failed to create cave cache dir: %w", err)
		}
		b.AddMapBind(BIND_RO, prep.CacheDir, filepath.Join(internalHome, ".cache", "pi"))
	}

	// 5. Setup Workspace & Home
	b.AddBind(BIND, c.Workspace)
	if err := os.MkdirAll(c.HomePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create home directory: %w", err)
	}
	localBin := filepath.Join(c.HomePath, ".local", "bin")
	if err := os.MkdirAll(localBin, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .local/bin: %w", err)
	}
	b.AddMapBind(BIND, c.HomePath, internalHome)

	// 6. Audio / Display / Agents (RO Try)
	xdgRuntime := os.Getenv("XDG_RUNTIME_DIR")
	if xdgRuntime != "" {
		b.envs["XDG_RUNTIME_DIR"] = xdgRuntime
		b.AddBind(BIND, xdgRuntime) // Shared for bus, display etc

		// Wayland
		waylandDisplay := os.Getenv("WAYLAND_DISPLAY")
		if waylandDisplay != "" {
			b.envs["WAYLAND_DISPLAY"] = waylandDisplay
		}

		// DBus
		dbusAddr := os.Getenv("DBUS_SESSION_BUS_ADDRESS")
		if dbusAddr != "" {
			b.envs["DBUS_SESSION_BUS_ADDRESS"] = dbusAddr
		}
	}

	// SSH Agent
	sshAuth := os.Getenv("SSH_AUTH_SOCK")
	if sshAuth != "" {
		b.envs["SSH_AUTH_SOCK"] = sshAuth
		b.AddBind(BIND_RO_TRY, sshAuth)
	}

	// 7. Graphics / USB / Hardware
	b.AddBind(BIND_RO, "/sys")
	b.AddBind(BIND_DEV, "/dev/dri")
	b.AddBind(BIND_DEV_TRY, "/dev/bus/usb")

	// 8. Create symlinks for packages in host Cave Home
	if err := pkgs.CreateSymlinks(c.HomePath, prep.Symlinks); err != nil {
		return nil, fmt.Errorf("failed to create symlinks: %w", err)
	}

	// 9. Set Environment
	b.envs["HOME"] = internalHome
	b.envs["USER"] = cfg.GetUser()
	b.envs["PI_WORKSPACE"] = c.Workspace
	caveName := c.Config.Name
	if c.Variant != "" {
		caveName = fmt.Sprintf("%s:%s", caveName, c.Variant)
	}
	b.envs["PI_CAVENAME"] = caveName

	b.AddEnvFirst("PATH", "/usr/bin:/bin")
	b.AddEnvFirst("PATH", filepath.Join(internalHome, ".local", "bin"))

	// Portals
	b.UnsetEnv("GTK_USE_PORTAL")
	b.UnsetEnv("QT_USE_PORTAL")

	// Apply recipe environment variables
	for k, v := range prep.Env {
		b.envs[k] = v
	}

	// Apply settings environment variables
	for k, v := range settings.Env {
		b.envs[k] = v
	}

	// 10. Set command
	if len(command) > 0 {
		b.SetCommand(command[0], command[1:]...)
	} else {
		b.SetCommand("/bin/bash")
	}

	return b.Cmd(), nil
}
