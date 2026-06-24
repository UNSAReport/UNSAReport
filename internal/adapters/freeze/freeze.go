package freeze

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/UNSAReport/UNSAReport/internal/dependencies"
	"github.com/UNSAReport/UNSAReport/internal/ports"
	"github.com/aymanbagabas/go-pty"
	"github.com/taigrr/bubbleterm/emulator"
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

	width := cfg.Columns
	height := cfg.Rows
	if height <= 0 {
		height = 500
	}

	output, err := runInPTY(ctx, commands, cfg, width, height)
	if err != nil && output == "" {
		return "", fmt.Errorf("run in pty: %w", err)
	}

	tempInput, err := os.CreateTemp("", "unsarep-freeze-input-*.txt")
	if err != nil {
		return output, fmt.Errorf("create temp input file: %w", err)
	}
	defer os.Remove(tempInput.Name()) //nolint:errcheck // best-effort cleanup

	if _, err := tempInput.WriteString(output); err != nil {
		return output, fmt.Errorf("write temp input file: %w", err)
	}
	tempInput.Close() //nolint:errcheck // file will be read-only after this

	if filepath.Ext(resultPath) == "" {
		resultPath += ".png"
	}

	svgPath := resultPath + ".svg"
	os.Remove(svgPath)       //nolint:errcheck // best-effort cleanup of old file
	defer os.Remove(svgPath) //nolint:errcheck // best-effort cleanup

	freezeArgs := []string{
		tempInput.Name(),
		"--language", "ansi",
		"--wrap", strconv.Itoa(width),
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

func getAnsi(colors map[string]string, name string) string {
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

func typeColored(ptmx io.Writer, vtWrite io.Writer, cmdStr string, cfg ports.CaptureConfig) {
	cCol := getAnsi(cfg.Colors, "command")
	aCol := getAnsi(cfg.Colors, "args")
	rCol := getAnsi(cfg.Colors, "reset")

	parts := strings.SplitN(cmdStr, " ", 2)
	firstWord := parts[0]
	rest := ""
	if len(parts) > 1 {
		rest = " " + parts[1]
	}

	vtWrite.Write([]byte(cCol))   //nolint:errcheck // PTY write, best-effort
	ptmx.Write([]byte(firstWord)) //nolint:errcheck // PTY write, best-effort
	if rest != "" {
		time.Sleep(20 * time.Millisecond)
		vtWrite.Write([]byte(aCol)) //nolint:errcheck // PTY write, best-effort
		ptmx.Write([]byte(rest))    //nolint:errcheck // PTY write, best-effort
	}
	time.Sleep(20 * time.Millisecond)
	vtWrite.Write([]byte(rCol)) //nolint:errcheck // PTY write, best-effort
}

func runInPTY(ctx context.Context, commands []ports.CaptureCommand, cfg ports.CaptureConfig, width, height int) (string, error) {
	shell, args := getDefaultShell()

	ptmx, err := pty.New()
	if err != nil {
		return "", err
	}
	defer ptmx.Close() //nolint:errcheck // PTY cleanup

	c := ptmx.Command(shell, args...)

	if cmd, ok := any(c).(*exec.Cmd); ok {
		cmd.Env = []string{
			"HOME=" + os.Getenv("HOME"),
			"USER=" + os.Getenv("USER"),
			"TERM=xterm-256color",
			"FORCE_COLOR=1",
			"PATH=" + os.Getenv("PATH"),
			"LANG=" + os.Getenv("LANG"),
		}
	}

	if err := c.Start(); err != nil {
		return "", err
	}

	if err := ptmx.Resize(width, height); err != nil {
		return "", err
	}

	vtRead, vtWrite := io.Pipe()
	emu, err := emulator.NewFromPipes(width, height, vtRead, ptmx)
	if err != nil {
		return "", err
	}
	defer emu.Close() //nolint:errcheck // emulator cleanup

	copyDone := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(vtWrite, ptmx)
		copyDone <- copyErr
	}()

	prompt := cfg.Prompt
	if prompt == "" {
		prompt = "❯ "
	}

	pCol := getAnsi(cfg.Colors, "prompt")
	rCol := getAnsi(cfg.Colors, "reset")

	if runtime.GOOS == "windows" {
		styledPrompt := pCol + prompt + rCol
		io.WriteString(ptmx, fmt.Sprintf("function prompt { \"%s\" }\r", styledPrompt)) //nolint:errcheck // PTY write
		io.WriteString(ptmx, "Clear-Host\r")                                            //nolint:errcheck // PTY write
	} else {
		io.WriteString(ptmx, fmt.Sprintf("export PS1='\\[\\e[%sm\\]%s\\[\\e[0m\\]'\r", cfg.Colors["prompt"], prompt)) //nolint:errcheck // PTY write
		io.WriteString(ptmx, "clear\r")                                                                               //nolint:errcheck // PTY write
	}

	time.Sleep(500 * time.Millisecond)

	io.WriteString(ptmx, "echo ---START---\r") //nolint:errcheck // PTY write

	anchorFound := false
	for range 20 {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		frame := emu.GetScreen()
		for _, row := range frame.Rows {
			if strings.Contains(row, "---START---") {
				anchorFound = true
				break
			}
		}
		if anchorFound {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	for _, cmd := range commands {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		switch cmd.Type {
		case "Command":
			typeColored(ptmx, vtWrite, cmd.Args, cfg)
			io.WriteString(ptmx, "\r") //nolint:errcheck // PTY write
			time.Sleep(500 * time.Millisecond)
		case "Type":
			typeColored(ptmx, vtWrite, cmd.Args, cfg)
			time.Sleep(100 * time.Millisecond)
		case "Enter":
			io.WriteString(ptmx, "\r") //nolint:errcheck // PTY write
			time.Sleep(500 * time.Millisecond)
		case "Raw":
			io.WriteString(ptmx, cmd.Args) //nolint:errcheck // PTY write
			time.Sleep(100 * time.Millisecond)
		case "Ctrl":
			if len(cmd.Args) > 0 {
				char := strings.ToLower(cmd.Args)[0]
				if char >= 'a' && char <= 'z' {
					ctrlChar := char - 'a' + 1
					ptmx.Write([]byte{ctrlChar}) //nolint:errcheck // PTY write, best-effort
				}
			}
			time.Sleep(500 * time.Millisecond)
		case "Key":
			switch strings.ToLower(cmd.Args) {
			case "enter":
				io.WriteString(ptmx, "\r") //nolint:errcheck // PTY write
			case "tab":
				io.WriteString(ptmx, "\t") //nolint:errcheck // PTY write
			case "backspace":
				io.WriteString(ptmx, "\x7f") //nolint:errcheck // PTY write
			case "escape", "esc":
				io.WriteString(ptmx, "\x1b") //nolint:errcheck // PTY write
			}
			time.Sleep(500 * time.Millisecond)
		case "Sleep":
			time.Sleep(cmd.Delay)
		}
	}

	time.Sleep(1 * time.Second)
	frame := emu.GetScreen()
	ptmx.Close() //nolint:errcheck // PTY cleanup before waiting for process

	// Wait for the copy goroutine to finish
	<-copyDone

	done := make(chan struct{})
	go func() {
		_ = c.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}

	startRow := 0
	for i, row := range frame.Rows {
		if strings.Contains(row, "---START---") {
			startRow = i + 1
		}
	}

	lastIdx := -1
	for i := len(frame.Rows) - 1; i >= startRow; i-- {
		clean := strings.ReplaceAll(frame.Rows[i], "\033[0m", "")
		if strings.TrimSpace(clean) != "" {
			lastIdx = i
			break
		}
	}

	if lastIdx == -1 || lastIdx < startRow {
		return "", nil
	}

	return strings.Join(frame.Rows[startRow:lastIdx+1], "\n") + "\n", nil
}
