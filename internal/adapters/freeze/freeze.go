package freeze

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/aymanbagabas/go-pty"
	"github.com/christianmz565/lab-report/internal/dependencies"
	"github.com/christianmz565/lab-report/internal/ports"
)

type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Render(ctx context.Context, resultPath string, commands []ports.CaptureCommand, flags []string, cfg ports.CaptureConfig) (string, error) {
	if err := dependencies.Check(dependencies.Freeze); err != nil {
		return "", err
	}
	if err := dependencies.Check(dependencies.ImageMagick); err != nil {
		return "", err
	}

	output, err := runInPTY(ctx, commands, cfg)
	if err != nil && output == "" {
		return "", fmt.Errorf("run in pty: %w", err)
	}

	tempInput, err := os.CreateTemp("", "lab-report-freeze-input-*.txt")
	if err != nil {
		return output, fmt.Errorf("create temp input file: %w", err)
	}
	defer os.Remove(tempInput.Name())

	if _, err := tempInput.WriteString(output); err != nil {
		return output, fmt.Errorf("write temp input file: %w", err)
	}
	tempInput.Close()

	svgPath := resultPath + ".svg"
	os.Remove(svgPath)
	defer os.Remove(svgPath)

	freezeArgs := []string{
		tempInput.Name(),
		"--language", "ansi",
		"--output", svgPath,
	}
	freezeArgs = append(freezeArgs, flags...)

	freezeCmd := exec.CommandContext(ctx, "freeze", freezeArgs...)
	if out, err := freezeCmd.CombinedOutput(); err != nil {
		return output, fmt.Errorf("freeze failed: %w (output: %s)", err, string(out))
	}

	magickCmd := exec.CommandContext(ctx, "magick", svgPath, resultPath)
	if out, err := magickCmd.CombinedOutput(); err != nil {
		return output, fmt.Errorf("magick failed: %w (output: %s)", err, string(out))
	}

	return output, nil
}

func getDefaultShell() (string, []string) {
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("pwsh"); err == nil {
			return "pwsh", []string{"-NoProfile"}
		}
		if _, err := exec.LookPath("powershell"); err == nil {
			return "powershell", []string{"-NoProfile"}
		}
		return "cmd", []string{}
	}

	shell := "bash"
	if _, err := exec.LookPath("bash"); err != nil {
		shell = "sh"
	}
	return shell, []string{"--norc", "--noprofile"}
}

func runInPTY(_ context.Context, commands []ports.CaptureCommand, cfg ports.CaptureConfig) (string, error) {
	shell, args := getDefaultShell()

	ptmx, err := pty.New()
	if err != nil {
		return "", err
	}

	c := ptmx.Command(shell, args...)

	if cmd, ok := any(c).(*exec.Cmd); ok {
		cmd.Env = append(os.Environ(), "TERM=xterm-256color", "FORCE_COLOR=1")
	}

	if err := c.Start(); err != nil {
		return "", err
	}

	if err := ptmx.Resize(100, 30); err != nil {
		return "", err
	}

	io.WriteString(ptmx, "echo ---READY---\r")
	if runtime.GOOS == "windows" {
		io.WriteString(ptmx, "function prompt { \"\" }\r")
	} else {
		io.WriteString(ptmx, "export PS1=\"\"\r")
		io.WriteString(ptmx, "stty -echo\r")
	}

	time.Sleep(500 * time.Millisecond)

	readyFound := false
	drainBuf := make([]byte, 1024)
	for !readyFound {
		n, err := ptmx.Read(drainBuf)
		if err != nil {
			break
		}
		if strings.Contains(string(drainBuf[:n]), "---READY---") {
			readyFound = true
		}
	}

	time.Sleep(200 * time.Millisecond)

	var buf strings.Builder
	var mu sync.Mutex
	done := make(chan struct{})
	go func() {
		defer close(done)
		tmp := make([]byte, 1024)
		for {
			n, err := ptmx.Read(tmp)
			if err != nil {
				return
			}
			mu.Lock()
			buf.Write(tmp[:n])
			mu.Unlock()
		}
	}()

	for _, cmd := range commands {
		switch cmd.Type {
		case "Command":
			mu.Lock()
			buf.WriteString(formatPrompt(cmd.Args, cfg))
			mu.Unlock()
			io.WriteString(ptmx, cmd.Args+"\r")
			time.Sleep(500 * time.Millisecond)
		case "Type":
			io.WriteString(ptmx, cmd.Args)
		case "Sleep":
			time.Sleep(cmd.Delay)
		}
	}

	time.Sleep(500 * time.Millisecond)

	io.WriteString(ptmx, "exit\r")

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		ptmx.Close()
	}

	waitErr := c.Wait()

	return buf.String(), waitErr
}

func formatPrompt(cmdStr string, cfg ports.CaptureConfig) string {
	prompt := cfg.Prompt
	if prompt == "" {
		prompt = "❯ "
	}
	colors := cfg.Colors

	getAnsi := func(name string) string {
		if colors == nil {
			if name == "reset" {
				return "\x1b[0m"
			}
			return ""
		}
		code, ok := colors[name]
		if !ok {
			if name == "reset" {
				return "\x1b[0m"
			}
			return ""
		}
		if code == "0" {
			return "\x1b[0m"
		}
		return "\x1b[" + code + "m"
	}

	pCol := getAnsi("prompt")
	cCol := getAnsi("command")
	aCol := getAnsi("args")
	rCol := getAnsi("reset")

	parts := strings.SplitN(cmdStr, " ", 2)
	firstWord := parts[0]
	rest := ""
	if len(parts) > 1 {
		rest = " " + parts[1]
	}

	var sb strings.Builder
	sb.WriteString(pCol)
	sb.WriteString(prompt)
	sb.WriteString(rCol)
	sb.WriteString(cCol)
	sb.WriteString(firstWord)
	sb.WriteString(rCol)
	if rest != "" {
		sb.WriteString(aCol)
		sb.WriteString(rest)
		sb.WriteString(rCol)
	}
	sb.WriteString("\n")
	return sb.String()
}
