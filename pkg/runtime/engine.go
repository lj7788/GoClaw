package runtime

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type Process struct {
	ID        string
	Cmd       *exec.Cmd
	StartedAt time.Time
	ExitCode  int
	Running   bool
	mu        sync.RWMutex
}

type Runtime struct {
	mu           sync.RWMutex
	processes    map[string]*Process
	workspaceDir string
	env          []string
}

func NewRuntime(workspaceDir string) *Runtime {
	return &Runtime{
		processes:    make(map[string]*Process),
		workspaceDir: workspaceDir,
		env:          os.Environ(),
	}
}

func (r *Runtime) Execute(ctx context.Context, cmd string, args ...string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	execCmd := exec.CommandContext(ctx, cmd, args...)
	execCmd.Dir = r.workspaceDir
	execCmd.Env = r.env

	output, err := execCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command failed: %w", err)
	}

	return string(output), nil
}

func (r *Runtime) StartProcess(ctx context.Context, name, cmd string, args ...string) (*Process, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	execCmd := exec.CommandContext(ctx, cmd, args...)
	execCmd.Dir = r.workspaceDir
	execCmd.Env = r.env
	// Note: SysProcAttr platform-specific settings removed for Windows compatibility

	if err := execCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	proc := &Process{
		ID:        name,
		Cmd:       execCmd,
		StartedAt: time.Now(),
		Running:   true,
	}

	r.processes[name] = proc

	go func() {
		execCmd.Wait()
		proc.mu.Lock()
		proc.Running = false
		if execCmd.ProcessState != nil {
			proc.ExitCode = execCmd.ProcessState.ExitCode()
		}
		proc.mu.Unlock()
	}()

	return proc, nil
}

func (r *Runtime) StopProcess(name string) error {
	r.mu.RLock()
	proc, exists := r.processes[name]
	r.mu.RUnlock()

	if !exists {
		return fmt.Errorf("process not found: %s", name)
	}

	if proc.Cmd.Process != nil {
		proc.Cmd.Process.Kill()
	}

	return nil
}

func (r *Runtime) GetProcess(name string) (*Process, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	proc, exists := r.processes[name]
	return proc, exists
}

func (r *Runtime) ListProcesses() []*Process {
	r.mu.RLock()
	defer r.mu.RUnlock()

	procs := make([]*Process, 0, len(r.processes))
	for _, proc := range r.processes {
		procs = append(procs, proc)
	}

	return procs
}

func (r *Runtime) RemoveProcess(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.processes[name]; !exists {
		return fmt.Errorf("process not found: %s", name)
	}

	delete(r.processes, name)
	return nil
}

func (r *Runtime) SetEnv(key, value string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, env := range r.env {
		if len(env) > len(key) && env[:len(key)] == key+"=" {
			r.env[i] = key + "=" + value
			return
		}
	}

	r.env = append(r.env, key+"="+value)
}

func (r *Runtime) GetEnv(key string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, env := range r.env {
		if len(env) > len(key) && env[:len(key)] == key+"=" {
			return env[len(key)+1:]
		}
	}

	return ""
}

func (p *Process) Wait() (int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.Running {
		return p.ExitCode, nil
	}

	return p.ExitCode, fmt.Errorf("process still running")
}

func (p *Process) Kill() error {
	if p.Cmd.Process != nil {
		return p.Cmd.Process.Kill()
	}
	return nil
}

type Workspace struct {
	mu      sync.RWMutex
	rootDir string
	subDirs map[string]string
}

func NewWorkspace(rootDir string) (*Workspace, error) {
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	ws := &Workspace{
		rootDir: rootDir,
		subDirs: make(map[string]string),
	}

	defaultDirs := []string{"tmp", "data", "logs", "cache"}
	for _, dir := range defaultDirs {
		dirPath := filepath.Join(rootDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create subdir %s: %w", dir, err)
		}
		ws.subDirs[dir] = dirPath
	}

	return ws, nil
}

func (ws *Workspace) Root() string {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.rootDir
}

func (ws *Workspace) SubDir(name string) string {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	if dir, exists := ws.subDirs[name]; exists {
		return dir
	}

	return filepath.Join(ws.rootDir, name)
}

func (ws *Workspace) Resolve(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(ws.rootDir, path)
}

func (ws *Workspace) ListSubDirs() []string {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	dirs := make([]string, 0, len(ws.subDirs))
	for dir := range ws.subDirs {
		dirs = append(dirs, dir)
	}

	return dirs
}
