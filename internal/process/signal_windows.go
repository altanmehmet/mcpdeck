package process

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func configureProcess(c *exec.Cmd) {
	ext := strings.ToLower(filepath.Ext(c.Path))
	if c.Err != nil || (ext != ".cmd" && ext != ".bat") {
		return
	}
	if ext == ".cmd" && resolveNodeShim(c) {
		return
	}
	// cmd.exe has different escaping from CreateProcess. Reject shell expansion
	// characters instead of evaluating untrusted MCP arguments as commands.
	quoted := make([]string, len(c.Args))
	for i, arg := range c.Args {
		if i == 0 {
			arg = c.Path
		}
		if strings.ContainsAny(arg, "&|<>^%!\r\n\"") {
			c.Err = fmt.Errorf("Windows batch arguments contain shell expansion characters; use an explicit executable runtime")
			return
		}
		quoted[i] = `"` + arg + `"`
	}
	directory, err := windows.GetSystemDirectory()
	if err != nil {
		c.Err = err
		return
	}
	c.Path = filepath.Join(directory, "cmd.exe")
	c.SysProcAttr = &syscall.SysProcAttr{CmdLine: `cmd.exe /d /s /c "` + strings.Join(quoted, " ") + `"`}
}

// A suspended start ensures no descendant can escape before job assignment.
// Job handles are private to this process; closing one kills its whole tree.
var processJobs = struct {
	sync.Mutex
	handles map[*exec.Cmd]windows.Handle
}{handles: make(map[*exec.Cmd]windows.Handle)}

func startProcess(c *exec.Cmd) error {
	// Bound inherited-pipe shutdown on Windows without changing Unix behavior.
	if c.WaitDelay == 0 {
		c.WaitDelay = 2 * time.Second
	}
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return err
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		windows.CloseHandle(job)
		return err
	}
	processJobs.Lock()
	processJobs.handles[c] = job
	// Keep cancellation from closing (and potentially reusing) the handle
	// while the suspended process is being assigned and resumed.
	if c.SysProcAttr == nil {
		c.SysProcAttr = &syscall.SysProcAttr{}
	}
	c.SysProcAttr.CreationFlags |= windows.CREATE_SUSPENDED
	if err = c.Start(); err != nil {
		processJobs.Unlock()
		releaseProcess(c)
		return err
	}
	process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(c.Process.Pid))
	if err == nil {
		err = windows.AssignProcessToJobObject(job, process)
		windows.CloseHandle(process)
	}
	if err == nil {
		err = resumeProcess(uint32(c.Process.Pid))
	}
	processJobs.Unlock()
	if err != nil {
		_ = forceProcess(c)
		_ = c.Process.Kill()
		_ = c.Wait()
		return fmt.Errorf("prepare Windows process tree: %w", err)
	}
	return nil
}

func resumeProcess(pid uint32) error {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(snapshot)
	entry := windows.ThreadEntry32{Size: uint32(unsafe.Sizeof(windows.ThreadEntry32{}))}
	for err = windows.Thread32First(snapshot, &entry); err == nil; err = windows.Thread32Next(snapshot, &entry) {
		if entry.OwnerProcessID != pid {
			continue
		}
		thread, err := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, entry.ThreadID)
		if err != nil {
			return err
		}
		_, err = windows.ResumeThread(thread)
		windows.CloseHandle(thread)
		return err
	}
	return fmt.Errorf("suspended process thread not found")
}

func terminateProcess(c *exec.Cmd) error { return forceProcess(c) }
func releaseProcess(c *exec.Cmd)         { _ = forceProcess(c) }
func forceProcess(c *exec.Cmd) error {
	processJobs.Lock()
	job, ok := processJobs.handles[c]
	delete(processJobs.handles, c)
	processJobs.Unlock()
	if ok {
		return windows.CloseHandle(job)
	}
	if c.Process != nil {
		return c.Process.Kill()
	}
	return nil
}
