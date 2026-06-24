package cmd

import "strings"

// parseArg splits a "name@range" argument into name and rangeSpec.
// If no "@" is present, rangeSpec defaults to "latest".
func parseArg(arg string) (name, rangeSpec string) {
	if i := strings.Index(arg, "@"); i != -1 {
		return arg[:i], arg[i+1:]
	}
	return arg, "latest"
}
