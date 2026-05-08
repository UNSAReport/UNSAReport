package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/christianmz565/lab-report/internal/capture"
	"github.com/christianmz565/lab-report/internal/config"
	"github.com/christianmz565/lab-report/internal/fsutil"
	"github.com/spf13/cobra"
)

type captureOptions struct {
	cwd             string
	freezeFlags     string
	saveFreezeFlags bool
}

func newCaptureCmd() *cobra.Command {
	opt := captureOptions{}

	cmd := &cobra.Command{
		Use:   "capture [flags] <output-file> <command> [ms:input]...",
		Short: "Capture terminal output and render it to a PNG via freeze",
		Long: `Capture terminal output and render it to a PNG.

Arguments:
  <output-file>  The base name of the generated file (e.g., 'img/demo'). Output will be '<output-file>.png'.
  <command>      The shell command to run and capture.
  [ms:input]     Optional. Simulate user input typed into the terminal.
                 Format is delay_in_ms:text. Example: 500:hello 1000:\n

Example:
  lab-report capture img/demo "python script.py" 500:name 1000:\n`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return cmd.Help()
			}
			return runCapture(cmd.Context(), opt, args)
		},
	}

	cmd.Flags().StringVar(&opt.cwd, "cwd", "", "Directory to run the command in (default: current directory)")
	cmd.Flags().StringVar(&opt.freezeFlags, "freeze-flags", "", "Additional flags to pass to freeze (e.g., \"--theme dracula\")")
	cmd.Flags().BoolVar(&opt.saveFreezeFlags, "save-freeze-flags", false, "Save the provided freeze flags as default in labreport.json")
	return cmd
}

var reTimedInput = regexp.MustCompile(`^\d+:`)

func runCapture(ctx context.Context, opt captureOptions, args []string) error {
	cwd := opt.cwd
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	cwd, err := filepath.Abs(cwd)
	if err != nil {
		return err
	}

	cfg, _, err := config.ReadConfig(cwd)
	if err != nil {
		// Ignore error, use defaults
	}

	if opt.saveFreezeFlags && opt.freezeFlags != "" {
		cfg.FreezeFlags = strings.Fields(opt.freezeFlags)
		if err := config.WriteConfig(cwd, cfg); err != nil {
			return fmt.Errorf("failed to save freeze flags: %w", err)
		}
		fmt.Fprintln(os.Stdout, "Freeze flags saved to labreport.json")
	}

	outputFile := args[0]
	rest := args[1:]
	if len(rest) == 0 {
		return fmt.Errorf("missing command")
	}

	if err := fsutil.EnsureDir(filepath.Dir(outputFile)); err != nil && filepath.Dir(outputFile) != "." {
		return err
	}

	if err := fsutil.EnsureDir("capture_logs"); err != nil {
		return err
	}

	runID := strings.NewReplacer(":", "-", ".", "-").Replace(time.Now().Format(time.RFC3339Nano))
	logFile := filepath.Join("capture_logs", fmt.Sprintf("lastcmd_%s.log", runID))

	var out strings.Builder
	i := 0
	for i < len(rest) {
		cmdStr := rest[i]
		i++
		inputs := make([]capture.TimedInput, 0)
		for i < len(rest) && reTimedInput.MatchString(rest[i]) {
			in, err := parseTimedInput(rest[i])
			if err != nil {
				return err
			}
			inputs = append(inputs, in)
			i++
		}

		out.WriteString(formatPrompt(cfg, cmdStr))

		cmdOut, err := capture.RunPTY(ctx, cwd, cmdStr, inputs)
		cmdOut = strings.ReplaceAll(cmdOut, "\r", "")
		out.WriteString(cmdOut)
		if err != nil {
			// Still continue to render what we got.
			fmt.Fprintf(&out, "\n[command exited with error: %v]\n", err)
		}
	}

	full := out.String()
	if err := fsutil.WriteFileAtomic(logFile, []byte(full), 0o644); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Generating %s.png...\n", outputFile)

	pngPath := outputFile + ".png"

	freezeArgs := []string{
		"--width", "1000",
		"--output", pngPath,
		"--language", "ansi",
	}

	// Add saved flags
	freezeArgs = append(freezeArgs, cfg.FreezeFlags...)

	// Add live flags
	if opt.freezeFlags != "" {
		freezeArgs = append(freezeArgs, strings.Fields(opt.freezeFlags)...)
	}

	freezeCmd := exec.CommandContext(ctx, "freeze", freezeArgs...)
	freezeCmd.Stdin = bytes.NewReader([]byte(full))
	freezeCmd.Stdout = os.Stdout
	freezeCmd.Stderr = os.Stderr
	if err := freezeCmd.Run(); err != nil {
		return fmt.Errorf("freeze command failed: %w", err)
	}

	fmt.Fprintln(os.Stdout, "Success!")
	return nil
}

func parseTimedInput(arg string) (capture.TimedInput, error) {
	parts := strings.SplitN(arg, ":", 2)
	if len(parts) != 2 {
		return capture.TimedInput{}, fmt.Errorf("invalid timed input %q (expected ms:text)", arg)
	}
	ms, err := time.ParseDuration(parts[0] + "ms")
	if err != nil {
		return capture.TimedInput{}, fmt.Errorf("invalid delay in %q: %w", arg, err)
	}
	return capture.TimedInput{Delay: ms, Text: unescapeTimedText(parts[1])}, nil
}

func unescapeTimedText(s string) string {
	// Make it possible to send Enter and other control characters from a shell.
	// Example: 1000:"echo hi\\r" or 1000:"hello\\n".
	r := strings.NewReplacer(
		"\\\\", "\\",
		"\\r", "\r",
		"\\n", "\n",
		"\\t", "\t",
	)
	return r.Replace(s)
}

func formatPrompt(cfg config.LabReportConfig, command string) string {
	esc := "\x1b["
	reset := esc + cfg.CaptureColors["reset"] + "m"
	promptColor := esc + cfg.CaptureColors["prompt"] + "m"
	cmdColor := esc + cfg.CaptureColors["command"] + "m"
	argsColor := esc + cfg.CaptureColors["args"] + "m"

	prompt := cfg.CapturePrompt
	if prompt == "" {
		prompt = "❯ "
	}

	words := strings.Fields(command)
	first := command
	rest := ""
	if len(words) > 0 {
		first = words[0]
		rest = strings.TrimSpace(strings.TrimPrefix(command, first))
	}

	colored := cmdColor + first + reset
	if rest != "" {
		colored += " " + argsColor + rest + reset
	}
	return promptColor + prompt + colored + "\n" + reset
}

