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
	"github.com/christianmz565/lab-report/internal/svg"
	"github.com/spf13/cobra"
)

type captureOptions struct {
	cwd string
}

func newCaptureCmd() *cobra.Command {
	opt := captureOptions{}

	cmd := &cobra.Command{
		Use:   "capture [flags] <output-file> <command> [ms:input]...",
		Short: "Capture terminal output and render it to a PNG via freeze",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCapture(cmd.Context(), opt, args)
		},
	}

	cmd.Flags().StringVar(&opt.cwd, "cwd", "", "Directory to run the command in (default: current directory)")
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

	outputFile := args[0]
	rest := args[1:]
	if len(rest) == 0 {
		return fmt.Errorf("missing command")
	}

	if err := EnsureDir(filepath.Dir(outputFile)); err != nil && filepath.Dir(outputFile) != "." {
		return err
	}

	if err := EnsureDir("capture_logs"); err != nil {
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

		out.WriteString(formatPrompt(cmdStr))

		cmdOut, err := capture.RunPTY(ctx, cwd, cmdStr, inputs)
		cmdOut = strings.ReplaceAll(cmdOut, "\r", "")
		out.WriteString(cmdOut)
		if err != nil {
			// Still continue to render what we got.
			out.WriteString(fmt.Sprintf("\n[command exited with error: %v]\n", err))
		}
	}

	full := out.String()
	if err := WriteFileAtomic(logFile, []byte(full), 0o644); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Generating %s.png...\n", outputFile)

	svgPath := outputFile + ".svg"
	pngPath := outputFile + ".png"

	freezeCmd := exec.CommandContext(ctx, "freeze",
		"--width", "1000",
		"--output", svgPath,
		"--language", "ansi",
		"-c", "user",
	)
	freezeCmd.Stdin = bytes.NewReader([]byte(full))
	freezeCmd.Stdout = os.Stdout
	freezeCmd.Stderr = os.Stderr
	if err := freezeCmd.Run(); err != nil {
		return fmt.Errorf("freeze command failed: %w", err)
	}

	// Convert SVG to PNG using canvas library
	if err := svg.ConvertSVGToPNG(svgPath, pngPath, 300); err != nil {
		return fmt.Errorf("SVG to PNG conversion failed: %w", err)
	}

	_ = os.Remove(svgPath)
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

func formatPrompt(command string) string {
	esc := "\x1b"
	reset := esc + "[0m"
	green := esc + "[38;5;114m"
	blue := esc + "[38;5;111m"
	flamingo := esc + "[38;5;217m"

	words := strings.Fields(command)
	first := command
	rest := ""
	if len(words) > 0 {
		first = words[0]
		rest = strings.TrimSpace(strings.TrimPrefix(command, first))
	}

	colored := blue + first + reset
	if rest != "" {
		colored += " " + flamingo + rest + reset
	}
	return green + "❯ " + colored + "\n" + reset
}
