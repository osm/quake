package death

import "strings"

type parser interface {
	parse(msg string) (string, string, bool)
}

type suffixParser struct {
	suffix string
}

func (m suffixParser) parse(msg string) (x, y string, ok bool) {
	if !strings.HasSuffix(msg, m.suffix) {
		return "", "", false
	}

	x = strings.TrimSpace(strings.TrimSuffix(msg, m.suffix))
	if x == "" {
		return "", "", false
	}

	return x, "", true
}

type infixParser struct {
	sep    string
	suffix string
}

func (m infixParser) parse(msg string) (x, y string, ok bool) {
	before, after, found := strings.Cut(msg, m.sep)
	if !found {
		return "", "", false
	}

	x = strings.TrimSpace(before)
	if x == "" {
		return "", "", false
	}

	if m.suffix != "" {
		if !strings.HasSuffix(after, m.suffix) {
			return "", "", false
		}
		after = strings.TrimSuffix(after, m.suffix)
	}

	y = strings.TrimSpace(after)
	if y == "" {
		return "", "", false
	}

	return x, y, true
}
