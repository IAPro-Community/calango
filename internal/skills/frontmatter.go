package skills

import (
	"bytes"
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// errFrontmatter marks a SKILL.md whose frontmatter cannot be read.
var errFrontmatter = errors.New("invalid frontmatter")

// errNoFrontmatter marks a SKILL.md that does not start with a frontmatter block.
var errNoFrontmatter = errors.New("missing frontmatter")

var frontmatterKey = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]*$`)

// parseFrontmatter reads the YAML frontmatter of a SKILL.md. It understands the subset
// skills use: top-level "key: value" pairs with plain, single- or double-quoted
// scalars, literal (|) and folded (>) block scalars, and comments. Nested maps and
// sequences are accepted and skipped; their top-level key maps to "". Anything else
// (tabs as indentation, duplicate keys, an unterminated block) is errFrontmatter.
// The parser never evaluates tags, anchors or aliases.
func parseFrontmatter(data []byte) (map[string]string, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], " ") != "---" {
		return nil, errNoFrontmatter
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if l := strings.TrimRight(lines[i], " "); l == "---" || l == "..." {
			end = i
			break
		}
	}
	if end < 0 {
		return nil, errFrontmatter
	}
	return parseBlock(lines[1:end])
}

func parseBlock(lines []string) (map[string]string, error) {
	fields := map[string]string{}
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(line, "\t") {
			return nil, errFrontmatter
		}
		if strings.HasPrefix(line, " ") {
			// Indented content outside a nested value or block scalar.
			return nil, errFrontmatter
		}
		key, rest, ok := strings.Cut(line, ":")
		key = strings.TrimSpace(key)
		if !ok || !frontmatterKey.MatchString(key) {
			return nil, errFrontmatter
		}
		if _, dup := fields[key]; dup {
			return nil, errFrontmatter
		}
		if rest != "" && rest[0] != ' ' {
			return nil, errFrontmatter
		}
		value := strings.TrimSpace(rest)
		// Collect the indented lines that belong to this key.
		var nested []string
		for i+1 < len(lines) {
			next := lines[i+1]
			if strings.TrimSpace(next) == "" {
				nested = append(nested, "")
				i++
				continue
			}
			if strings.HasPrefix(next, "\t") {
				return nil, errFrontmatter
			}
			if !strings.HasPrefix(next, " ") && !strings.HasPrefix(next, "- ") && next != "-" {
				break
			}
			nested = append(nested, next)
			i++
		}
		for len(nested) > 0 && nested[len(nested)-1] == "" {
			nested = nested[:len(nested)-1]
		}
		parsed, err := scalar(value, nested)
		if err != nil {
			return nil, err
		}
		fields[key] = parsed
	}
	return fields, nil
}

// scalar decodes one top-level value. nested holds the indented lines after the key.
func scalar(value string, nested []string) (string, error) {
	switch {
	case value == "":
		// A nested map or sequence: accepted, not interpreted.
		return "", nil
	case value[0] == '|' || value[0] == '>':
		header := value[1:]
		if i := strings.Index(header, "#"); i >= 0 {
			header = header[:i]
		}
		header = strings.TrimSpace(header)
		if strings.Trim(header, "+-0123456789") != "" {
			return "", errFrontmatter
		}
		return blockScalar(value[0] == '>', strings.Contains(header, "-"), nested)
	case len(nested) > 0:
		// A plain or quoted scalar continued on indented lines.
		parts := []string{value}
		for _, n := range nested {
			parts = append(parts, strings.TrimSpace(n))
		}
		value = strings.Join(parts, " ")
	}
	switch value[0] {
	case '"':
		out, err := strconv.Unquote(value)
		if err != nil {
			return "", errFrontmatter
		}
		return out, nil
	case '\'':
		if len(value) < 2 || value[len(value)-1] != '\'' {
			return "", errFrontmatter
		}
		return strings.ReplaceAll(value[1:len(value)-1], "''", "'"), nil
	case '&', '*', '!':
		// Anchors, aliases and tags are not supported.
		return "", errFrontmatter
	case '[', '{':
		// Flow collections are kept raw.
		return value, nil
	}
	if i := strings.Index(value, " #"); i >= 0 {
		value = strings.TrimSpace(value[:i])
	}
	return value, nil
}

func blockScalar(folded, strip bool, nested []string) (string, error) {
	indent := -1
	for _, n := range nested {
		if n == "" {
			continue
		}
		w := len(n) - len(strings.TrimLeft(n, " "))
		if indent < 0 || w < indent {
			indent = w
		}
	}
	if indent <= 0 {
		if len(nested) == 0 {
			return "", nil
		}
		return "", errFrontmatter
	}
	out := make([]string, 0, len(nested))
	for _, n := range nested {
		if n == "" {
			out = append(out, "")
			continue
		}
		out = append(out, n[indent:])
	}
	var text string
	if folded {
		var b strings.Builder
		for i, l := range out {
			switch {
			case i == 0:
			case l == "" || out[i-1] == "":
				b.WriteString("\n")
			default:
				b.WriteString(" ")
			}
			b.WriteString(l)
		}
		text = b.String()
	} else {
		text = strings.Join(out, "\n")
	}
	text = strings.TrimRight(text, "\n")
	if strip {
		return text, nil
	}
	// Clip and keep chomping: one final line break.
	return text + "\n", nil
}
