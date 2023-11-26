package gradle

import (
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/utils"
)

func listProjects(rootDir string) ([]string, error) {
	cmd := exec.Command(filepath.Join(rootDir, "gradlew"), "projects")
	cmd.Dir = rootDir

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	result, err := parseProjects(string(output))
	if err != nil {
		return nil, err
	}

	return result, nil
}

func parseProjects(content string) ([]string, error) {
	rootRE := regexp.MustCompile(`^Root project '([^']+)'$`)
	projRE := regexp.MustCompile(`^[-+\\| ]+Project '([^']+)'$`)

	var result []string

	rootAdded := false
	lines := strings.Split(content, "\n")
	for _, l := range lines {
		if !rootAdded {
			m := rootRE.FindStringSubmatch(l)
			if m != nil {
				result = append(result, m[1])
				rootAdded = true
			}
		}

		m := projRE.FindStringSubmatch(l)
		if m != nil {
			result = append(result, m[1])
		}
	}

	return result, nil
}

func parseDeps(projects *model.Projects, content string, rootProj string, projsInsideRoot map[string]bool) error {
	rootProjRE := regexp.MustCompile(`^(?:Root project|Project) '([^']+)'$`)
	depRE := regexp.MustCompile(`^([-+\\| ]+)(?:project )?([a-zA-Z0-9:._-]+)`)
	versionRE := regexp.MustCompile(`^([a-zA-Z0-9:._-]+):\d+[^:]+$`)

	state := waitingRoot
	var stack []pd

	lines := strings.Split(content, "\n")
	for _, l := range lines {
		if state == waitingRoot {
			rootMatches := rootProjRE.FindStringSubmatch(l)
			if rootMatches != nil {
				p := projects.GetOrCreate(rootProj, rootMatches[1])
				stack = append(stack, pd{p, 0})
				state = waitingDeps
			}
			continue
		}

		if state == waitingDeps {
			if strings.HasPrefix(l, "\\---") || strings.HasPrefix(l, "+---") {
				state = parsingDeps
			}
		}

		if state == parsingDeps {
			if len(l) == 0 {
				break
			}

			depMatches := depRE.FindStringSubmatch(l)
			if depMatches == nil {
				return errors.Errorf("invalid dependency line: %v", l)
			}

			depth := len(depMatches[1])
			depName := depMatches[2]

			root := rootProj
			if !projsInsideRoot[depName] {
				root = "external"
				m := versionRE.FindStringSubmatch(depName)
				if m != nil {
					depName = m[1]
				}
			}

			p := projects.GetOrCreate(root, depName)

			lp := utils.Last(stack)
			for depth <= lp.depth {
				stack = utils.RemoveLast(stack)
				lp = utils.Last(stack)
			}

			lp.proj.GetOrCreateDependency(p)
			stack = append(stack, pd{p, depth})
		}
	}

	return nil
}

type parseState int

const (
	waitingRoot parseState = iota
	waitingDeps
	parsingDeps
)

type pd struct {
	proj  *model.Project
	depth int
}
