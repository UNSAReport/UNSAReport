package templates

import (
	"fmt"
	"strings"
)

type Source struct {
	Owner string
	Repo  string
	Ref   string
}

func DefaultSource() Source {
	return Source{Owner: "christianmz565", Repo: "lab-report", Ref: "main"}
}

func ParseRepo(repo string) (owner string, name string, err error) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid repo %q (expected owner/repo)", repo)
	}
	return parts[0], parts[1], nil
}

func (s Source) ZipURL() string {
	return fmt.Sprintf("https://codeload.github.com/%s/%s/zip/%s", s.Owner, s.Repo, s.Ref)
}
