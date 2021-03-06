package console

import (
	"fmt"
	"io"
	"os"
	"os/signal"

	syscall "github.com/docker/docker/pkg/signal"
	"github.com/docker/docker/pkg/term"
	"github.com/jcelliott/lumber"
	"github.com/nanobox-io/golang-docker-client"

	"github.com/nanobox-io/nanobox/commands/registry"
	"github.com/nanobox-io/nanobox/models"
)

type ConsoleConfig struct {
	Command string
	Cwd     string
	Shell   string
	DevIP   string
}

func Run(id string, consoleConfig ConsoleConfig) error {
	// set the default shell
	if consoleConfig.Shell == "" {
		consoleConfig.Shell = "bash"
	}

	// this is the default command to run in the container
	cmd := []string{"/bin/bash", "-cl"}

	cmdPart := "exec "
	if consoleConfig.Cwd != "" {
		cmdPart = fmt.Sprintf("cd %s; %s", consoleConfig.Cwd, cmdPart)
	}

	if consoleConfig.Command != "" {
		cmdPart = cmdPart + consoleConfig.Command
	} else {
		cmdPart = cmdPart + consoleConfig.Shell
	}
	cmd = append(cmd, cmdPart)

	// establish file descriptors for std streams
	stdin, stdout, _ := term.StdStreams()
	stdInFD, isTerminal := term.GetFdInfo(stdin)
	stdOutFD, _ := term.GetFdInfo(stdout)

	user := "gonano"
	if registry.GetString("console_user") != "" {
		user = registry.GetString("console_user")
	}
	// initiate a docker exec
	execConfig := docker.ExecConfig{
		ID:     id,
		User:   user,
		Cmd:    cmd,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
		Tty:    isTerminal,
	}

	exec, resp, err := docker.ExecStart(execConfig)
	if err != nil {
		lumber.Error("dockerexecerror: %s", err)
		return err
	}
	defer resp.Conn.Close()

	console := models.Console{exec.ID, id}
	console.Save()
	defer console.Delete()

	// if we are using a term, lets upgrade it to RawMode
	if isTerminal {
		go monitor(stdOutFD, exec.ID)

		oldInState, err := term.SetRawTerminal(stdInFD)
		if err == nil {
			defer term.RestoreTerminal(stdInFD, oldInState)
		}

		oldOutState, err := term.SetRawTerminalOutput(stdOutFD)
		if err == nil {
			defer term.RestoreTerminal(stdOutFD, oldOutState)
		}
	}

	go io.Copy(resp.Conn, os.Stdin)
	io.Copy(os.Stdout, resp.Reader)

	// after the console closes lets get the exit code
	exInspect, _ := docker.ExecInspect(exec.ID)
	if exInspect.ExitCode != 0 {
		registry.Set("exit_code", exInspect.ExitCode)
	}

	return nil
}

// monitor ...
func monitor(stdOutFD uintptr, execID string) {
	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGWINCH)
	defer signal.Stop(sigs)

	// inform the server what the starting size is
	resize(stdOutFD, execID)

	// resize the tty for any signals received
	for range sigs {
		resize(stdOutFD, execID)
	}
}

func resize(fd uintptr, execID string) {
	ws, err := term.GetWinsize(fd)
	if err != nil {
		lumber.Error("env:console:resize():docker.ContainerExecResize(%d): %s", fd, err)
		return
	}

	// extract height and width
	w := int(ws.Width)
	h := int(ws.Height)

	err = docker.ContainerExecResize(execID, h, w)
	if err != nil {
		lumber.Error("env:console:resize():docker.ContainerExecResize(%s, %d, %d): %s", execID, h, w, err)
		return
	}
}
