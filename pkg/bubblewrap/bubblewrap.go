// Package bubblewrap provides a wrapper around the Linux bubblewrap (bwrap) utility.
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

	"pi/pkg/common"
	"pi/pkg/config"
)

// BindType represents the type of bind mount to perform.
type BindType = string

const (
	BIND         BindType = "--bind"
	BIND_TRY     BindType = "--bind-try"
	BIND_DEV     BindType = "--dev-bind"
	BIND_DEV_TRY BindType = "--dev-bind-try"
	BIND_RO      BindType = "--ro-bind"
	BIND_RO_TRY  BindType = "--ro-bind-try"
	BIND_FD      BindType = "--bind-fd"
	BIND_RO_FD   BindType = "--ro-bind-fd"
	BIND_DATA    BindType = "--bind-data"
	BIND_DATA_RO BindType = "--ro-bind-data"

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

// Bubblewrap represents a pending sandbox execution configuration.
type Bubblewrap struct {
	binds      map[string]bindPair
	envs       map[string]string
	unsets     []string
	flags      []string
	executable string
	cmdline    []string
}

// Create initializes a new Bubblewrap configuration with the current environment.
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
	args := []string{}
	args = append(args, b.flags...)

	for _, key := range sortedKeys(b.binds) {
		bind := b.binds[key]
		if bind.host_source == "" {
			args = append(args, bind.bindType, bind.cave_target)
		} else {
			args = append(args, bind.bindType, bind.host_source, bind.cave_target)
		}
	}

	for _, k := range sortedKeys(b.envs) {
		args = append(args, "--setenv", k, b.envs[k])
	}
	for _, k := range b.unsets {
		args = append(args, "--unsetenv", k)
	}

	if b.executable != "" {
		args = append(args, "--", b.executable)
		args = append(args, b.cmdline...)
	}

	return exec.Command(execPath, args...)
}

func (b *Bubblewrap) Spawn() error {
	cmd := b.Cmd()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func Exec(cmd *exec.Cmd) error {
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

func (b *Bubblewrap) BindSlice() []common.SandboxBind {
	var res []common.SandboxBind
	for _, key := range sortedKeys(b.binds) {
		bind := b.binds[key]
		res = append(res, common.SandboxBind{
			Source: bind.host_source,
			Target: bind.cave_target,
			Type:   bind.bindType,
		})
	}
	return res
}

func (b *Bubblewrap) EnvSlice() []string {
	var res []string
	for _, k := range sortedKeys(b.envs) {
		res = append(res, fmt.Sprintf("%s=%s", k, b.envs[k]))
	}
	return res
}

func CmdFromSandbox(s *common.SandboxConfig) *exec.Cmd {
	execPath := "/usr/bin/bwrap"
	args := []string{}

	if s != nil {
		args = append(args, s.Flags...)
		for _, bind := range s.Binds {
			if bind.Source == "" {
				args = append(args, bind.Type, bind.Target)
			} else {
				args = append(args, bind.Type, bind.Source, bind.Target)
			}
		}
		for _, k := range s.UnsetEnvs {
			args = append(args, "--unsetenv", k)
		}

		for _, env := range s.Env {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				args = append(args, "--setenv", parts[0], parts[1])
			}
		}

		if s.Exe != "" {
			args = append(args, "--", s.Exe)
			args = append(args, s.Args...)
		}
	}

	return exec.Command(execPath, args...)
}

// SandboxInfo provides all details needed to construct a sandbox.
type SandboxInfo struct {
	ID        string
	Workspace string
	HomePath  string
	CaveName  string
	Env       map[string]string // Settings Env
}

// ResolveLaunch prepares a command to be executed inside the bubblewrap sandbox.
func ResolveLaunch(ctx context.Context, cfg config.Config, info SandboxInfo, prep *common.PreparationResult, command []string) (*common.SandboxConfig, error) {
	b := Create()
	internalHome := cfg.GetHostHome()

	b.AddFlag("--unshare-pid")
	b.AddFlag("--die-with-parent")

	b.AddBind(BIND_RO, "/usr")
	b.AddBind(BIND_RO, "/lib")
	if _, err := os.Stat("/lib64"); err == nil {
		b.AddBind(BIND_RO, "/lib64")
	}
	b.AddBind(BIND_RO, "/bin")
	b.AddBind(BIND_RO, "/sbin")
	b.AddBind(BIND_RO, "/opt")
	b.AddBind(BIND_RO, "/etc")

	b.AddVirtual(PROC, "/proc")
	b.AddVirtual(DEV, "/dev")
	b.AddVirtual(TMPFS, "/tmp")
	b.AddVirtual(TMPFS, "/run")

	if prep.CacheDir != "" {
		caveCache := filepath.Join(info.HomePath, ".cache", "pi")
		if err := os.MkdirAll(caveCache, 0755); err != nil {
			return nil, fmt.Errorf("failed to create cave cache dir: %w", err)
		}
		b.AddMapBind(BIND_RO, prep.CacheDir, filepath.Join(internalHome, ".cache", "pi"))
	}

	b.AddBind(BIND, info.Workspace)
	if err := os.MkdirAll(info.HomePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create home directory: %w", err)
	}
	localBin := filepath.Join(info.HomePath, ".local", "bin")
	if err := os.MkdirAll(localBin, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .local/bin: %w", err)
	}
	b.AddMapBind(BIND, info.HomePath, internalHome)

	xdgRuntime := os.Getenv("XDG_RUNTIME_DIR")
	if xdgRuntime != "" {
		b.envs["XDG_RUNTIME_DIR"] = xdgRuntime
		b.AddBind(BIND, xdgRuntime)
		if wd := os.Getenv("WAYLAND_DISPLAY"); wd != "" {
			b.envs["WAYLAND_DISPLAY"] = wd
		}
		if da := os.Getenv("DBUS_SESSION_BUS_ADDRESS"); da != "" {
			b.envs["DBUS_SESSION_BUS_ADDRESS"] = da
		}
	}

	if sshAuth := os.Getenv("SSH_AUTH_SOCK"); sshAuth != "" {
		b.envs["SSH_AUTH_SOCK"] = sshAuth
		b.AddBind(BIND_RO_TRY, sshAuth)
	}

	b.AddBind(BIND_RO, "/sys")
	b.AddBind(BIND_DEV, "/dev/dri")
	b.AddBind(BIND_DEV_TRY, "/dev/bus/usb")

	b.envs["HOME"] = internalHome
	b.envs["USER"] = cfg.GetUser()
	b.envs["PI_WORKSPACE"] = info.Workspace
	b.envs["PI_CAVENAME"] = info.CaveName

	b.AddEnvFirst("PATH", "/usr/bin:/bin")
	b.AddEnvFirst("PATH", filepath.Join(internalHome, ".local", "bin"))

	b.UnsetEnv("GTK_USE_PORTAL")
	b.UnsetEnv("QT_USE_PORTAL")

	for k, v := range prep.Env {
		b.envs[k] = v
	}
	for k, v := range info.Env {
		b.envs[k] = v
	}

	if len(command) > 0 {
		b.SetCommand(command[0], command[1:]...)
	} else {
		b.SetCommand("/bin/bash")
	}

	return &common.SandboxConfig{
		Exe:       b.executable,
		Args:      b.cmdline,
		Env:       b.EnvSlice(),
		Binds:     b.BindSlice(),
		Flags:     b.flags,
		UnsetEnvs: b.unsets,
	}, nil
}
