package cli

import (
	"fmt"
	"regexp"
	"strings"
)

var packageRefRE = regexp.MustCompile(`^([a-z0-9][a-z0-9-]*[a-z0-9]|[a-z0-9])/([a-z0-9][a-z0-9-]*[a-z0-9]|[a-z0-9])/([a-z0-9][a-z0-9-]*[a-z0-9]|[a-z0-9])@(.+)$`)

type PackageRef struct {
	Org       string
	Namespace string
	Name      string
	Selector  string
}

func ParsePackageRef(raw string) (PackageRef, error) {
	raw = strings.TrimSpace(raw)
	m := packageRefRE.FindStringSubmatch(raw)
	if m == nil {
		return PackageRef{}, fmt.Errorf("invalid package reference %q; expected org/namespace/package@selector", raw)
	}
	return PackageRef{
		Org:       m[1],
		Namespace: m[2],
		Name:      m[3],
		Selector:  m[4],
	}, nil
}

func (p PackageRef) String() string {
	return p.Org + "/" + p.Namespace + "/" + p.Name + "@" + p.Selector
}

func (p PackageRef) PackagePath() string {
	return p.Org + "/" + p.Namespace + "/" + p.Name
}
