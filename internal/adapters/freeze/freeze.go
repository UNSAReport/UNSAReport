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
	height := 500

	output, err := runInPTY(ctx, commands, cfg, width, height)
	if err != nil && output == "" {
		return "", fmt.Errorf("run in pty: %w", err)
	}

	tempInput, err := os.CreateTemp("", "unsarep-freeze-input-*.txt")
	if err != nil {
		return output, fmt.Errorf("create temp input file: %w", err)
	}
	defer os.Remove(tempInput.Name())

	if _, err := tempInput.WriteString(output); err != nil {
		return output, fmt.Errorf("write temp input file: %w", err)
	}
	tempInput.Close()

	if filepath.Ext(resultPath) == "" {
		resultPath += ".png"
	}

	svgPath := resultPath + ".svg"
	os.Remove(svgPath)
	defer os.Remove(svgPath)

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

	vtWrite.Write([]byte(cCol))
	ptmx.Write([]byte(firstWord))
	if rest != "" {
		time.Sleep(20 * time.Millisecond)
		vtWrite.Write([]byte(aCol))
		ptmx.Write([]byte(rest))
	}
	time.Sleep(20 * time.Millisecond)
	vtWrite.Write([]byte(rCol))
}

func runInPTY(_ context.Context, commands []ports.CaptureCommand, cfg ports.CaptureConfig, width, height int) (string, error) {
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

	if err := ptmx.Resize(width, height); err != nil {
		return "", err
	}

	vtRead, vtWrite := io.Pipe()
	emu, err := emulator.NewFromPipes(width, height, vtRead, ptmx)
	if err != nil {
		return "", err
	}
	defer emu.Close()

	go io.Copy(vtWrite, ptmx)

	prompt := cfg.Prompt
	if prompt == "" {
		prompt = "❯ "
	}

	pCol := getAnsi(cfg.Colors, "prompt")
	rCol := getAnsi(cfg.Colors, "reset")

	if runtime.GOOS == "windows" {
		styledPrompt := pCol + prompt + rCol
		io.WriteString(ptmx, fmt.Sprintf("function prompt { \"%s\" }\r", styledPrompt))
		io.WriteString(ptmx, "Clear-Host\r")
	} else {
		io.WriteString(ptmx, fmt.Sprintf("export PS1='\\[\\e[%sm\\]%s\\[\\e[0m\\]'\r", cfg.Colors["prompt"], prompt))
		io.WriteString(ptmx, "clear\r")
	}

	time.Sleep(500 * time.Millisecond)

	io.WriteString(ptmx, "echo ---START---\r")

	anchorFound := false
	for range 20 {
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
		switch cmd.Type {
		case "Command":
			typeColored(ptmx, vtWrite, cmd.Args, cfg)
			io.WriteString(ptmx, "\r")
			time.Sleep(500 * time.Millisecond)
		case "Type":
			typeColored(ptmx, vtWrite, cmd.Args, cfg)
			time.Sleep(100 * time.Millisecond)
		case "Enter":
			io.WriteString(ptmx, "\r")
			time.Sleep(500 * time.Millisecond)
		case "Raw":
			io.WriteString(ptmx, cmd.Args)
			time.Sleep(100 * time.Millisecond)
		case "Ctrl":
			if len(cmd.Args) > 0 {
				char := strings.ToLower(cmd.Args)[0]
				if char >= 'a' && char <= 'z' {
					ctrlChar := char - 'a' + 1
					ptmx.Write([]byte{ctrlChar})
				}
			}
			time.Sleep(500 * time.Millisecond)
		case "Key":
			switch strings.ToLower(cmd.Args) {
			case "enter":
				io.WriteString(ptmx, "\r")
			case "tab":
				io.WriteString(ptmx, "\t")
			case "backspace":
				io.WriteString(ptmx, "\x7f")
			case "escape", "esc":
				io.WriteString(ptmx, "\x1b")
			}
			time.Sleep(500 * time.Millisecond)
		case "Sleep":
			time.Sleep(cmd.Delay)
		}
	}

	time.Sleep(1 * time.Second)
	frame := emu.GetScreen()
	ptmx.Close()

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
