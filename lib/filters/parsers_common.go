package filters

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

func ParseStringFilter(rule string) (func(string) bool, error) {
	rule = strings.TrimSpace(rule)

	if rule == "" {
		return func(s string) bool {
			return true
		}, nil

	} else if strings.HasPrefix(rule, "re:") {
		re, err := regexp.Compile("(?i)" + strings.TrimPrefix(rule, "re:"))
		if err != nil {
			return nil, errors.Wrapf(err, "invalid project RE: %v", rule)
		}

		return re.MatchString, nil

	} else if strings.Index(rule, "*") >= 0 {
		filterRE := strings.ReplaceAll(rule, `\`, `\\`)
		filterRE = strings.ReplaceAll(filterRE, `.`, `\.`)
		filterRE = strings.ReplaceAll(filterRE, `(`, `\(`)
		filterRE = strings.ReplaceAll(filterRE, `)`, `\)`)
		filterRE = strings.ReplaceAll(filterRE, `^`, `\^`)
		filterRE = strings.ReplaceAll(filterRE, `$`, `\$`)
		filterRE = strings.ReplaceAll(filterRE, `*`, `.*`)

		re, err := regexp.Compile("(?i)^" + filterRE + "$")
		if err != nil {
			return nil, errors.Wrapf(err, "invalid project filter: %v", rule)
		}

		return re.MatchString, nil

	} else {
		return func(s string) bool {
			return strings.EqualFold(s, rule)
		}, nil
	}
}
