package portage

import (
	"fmt"
	"regexp"
	"strings"
)

type RequiredUseNodeKind string

const (
	RequiredUseAll         RequiredUseNodeKind = "all"
	RequiredUseFlag        RequiredUseNodeKind = "flag"
	RequiredUseConditional RequiredUseNodeKind = "conditional"
	RequiredUseAnyOf       RequiredUseNodeKind = "any-of"
	RequiredUseExactlyOne  RequiredUseNodeKind = "exactly-one-of"
	RequiredUseAtMostOne   RequiredUseNodeKind = "at-most-one-of"
)

type RequiredUseNode struct {
	Kind     RequiredUseNodeKind
	Flag     string
	Negated  bool
	Children []*RequiredUseNode
}

type RequiredUseExpression struct {
	Raw  string
	Root *RequiredUseNode
}

type RequiredUseCondition struct {
	Flag    string
	Enabled bool
}

type RequiredUseViolation struct {
	Kind        RequiredUseNodeKind
	Flag        string
	Negated     bool
	Terms       []string
	SimpleFlags bool
	Context     []RequiredUseCondition
	Expression  string
}

type RequiredUseFailure struct {
	PackageSpec           string
	Package               string
	ExactAtom             string
	UnsatisfiedExpression string
	CompleteExpression    string
	CurrentUse            map[string]bool
	RequiredBy            []string
	Raw                   string
}

func ParseRequiredUseExpression(raw string) (*RequiredUseExpression, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return &RequiredUseExpression{
			Raw:  "",
			Root: &RequiredUseNode{Kind: RequiredUseAll},
		}, nil
	}

	tokens := tokenizeRequiredUse(raw)
	parser := &requiredUseParser{tokens: tokens}

	root, err := parser.parseSequence(false)
	if err != nil {
		return nil, err
	}

	if parser.pos != len(parser.tokens) {
		return nil, fmt.Errorf("unexpected REQUIRED_USE token %q", parser.tokens[parser.pos])
	}

	return &RequiredUseExpression{
		Raw:  raw,
		Root: root,
	}, nil
}

func (e *RequiredUseExpression) Violations(enabled map[string]bool) []RequiredUseViolation {
	if e == nil || e.Root == nil {
		return nil
	}

	return evaluateRequiredUseViolations(e.Root, enabled, nil)
}

func (e *RequiredUseExpression) Satisfied(enabled map[string]bool) bool {
	if e == nil || e.Root == nil {
		return true
	}

	return requiredUseNodeSatisfied(e.Root, enabled)
}

func ParseRequiredUseFailure(raw string) *RequiredUseFailure {
	if !strings.Contains(raw, "The following REQUIRED_USE flag constraints are unsatisfied:") {
		return nil
	}

	lines := strings.Split(raw, "\n")
	failure := &RequiredUseFailure{
		CurrentUse: make(map[string]bool),
		Raw:        raw,
	}

	packageLinePattern := regexp.MustCompile(`^\s*-\s+(\S+)\s+.*USE="([^"]*)"`)
	requiredByPattern := regexp.MustCompile(`^\s*\(dependency required by\s+"([^"]+)".*$`)

	for _, line := range lines {
		if failure.PackageSpec == "" {
			if matches := packageLinePattern.FindStringSubmatch(line); matches != nil {
				failure.PackageSpec = strings.TrimSpace(matches[1])
				failure.Package = PackageNameFromSpec(failure.PackageSpec)
				failure.ExactAtom = exactAtomFromPackageSpec(failure.PackageSpec)
				failure.CurrentUse = parseUseAssignment(matches[2])
			}
		}

		if matches := requiredByPattern.FindStringSubmatch(strings.TrimSpace(line)); matches != nil {
			failure.RequiredBy = append(failure.RequiredBy, strings.TrimSpace(matches[1]))
		}
	}

	failure.UnsatisfiedExpression = collectRequiredUseBlock(
		lines,
		"The following REQUIRED_USE flag constraints are unsatisfied:",
		"The above constraints are a subset of the following complete expression:",
	)

	failure.CompleteExpression = collectRequiredUseBlock(
		lines,
		"The above constraints are a subset of the following complete expression:",
		"",
	)

	if failure.Package == "" && failure.PackageSpec != "" {
		failure.Package = PackageNameFromSpec(failure.PackageSpec)
	}

	return failure
}

func (f *RequiredUseFailure) Expression() string {
	if f == nil {
		return ""
	}

	if strings.TrimSpace(f.CompleteExpression) != "" {
		return strings.TrimSpace(f.CompleteExpression)
	}

	return strings.TrimSpace(f.UnsatisfiedExpression)
}

func PackageNameFromSpec(spec string) string {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return ""
	}

	for len(spec) > 0 {
		switch spec[0] {
		case '=', '<', '>', '~':
			spec = strings.TrimLeft(spec, "=<>~")
		default:
			goto operatorsDone
		}
	}

operatorsDone:
	if beforeRepo, _, found := strings.Cut(spec, "::"); found {
		spec = beforeRepo
	}

	if slash := strings.Index(spec, "/"); slash >= 0 {
		if colon := strings.Index(spec[slash+1:], ":"); colon >= 0 {
			spec = spec[:slash+1+colon]
		}
	}

	slash := strings.Index(spec, "/")
	if slash < 0 {
		return spec
	}

	category := spec[:slash]
	nameAndVersion := spec[slash+1:]
	versionStart := findVersionStart(nameAndVersion)
	if versionStart < 0 {
		return spec
	}

	name := strings.TrimSuffix(nameAndVersion[:versionStart], "-")
	if name == "" {
		return spec
	}

	return category + "/" + name
}

type requiredUseParser struct {
	tokens []string
	pos    int
}

func (p *requiredUseParser) parseSequence(expectClose bool) (*RequiredUseNode, error) {
	node := &RequiredUseNode{Kind: RequiredUseAll}

	for p.pos < len(p.tokens) {
		if p.tokens[p.pos] == ")" {
			if !expectClose {
				return nil, fmt.Errorf("unexpected ')' in REQUIRED_USE")
			}

			p.pos++
			return node, nil
		}

		child, err := p.parseTerm()
		if err != nil {
			return nil, err
		}

		node.Children = append(node.Children, child)
	}

	if expectClose {
		return nil, fmt.Errorf("unterminated '(' in REQUIRED_USE")
	}

	return node, nil
}

func (p *requiredUseParser) parseTerm() (*RequiredUseNode, error) {
	if p.pos >= len(p.tokens) {
		return nil, fmt.Errorf("unexpected end of REQUIRED_USE")
	}

	token := p.tokens[p.pos]
	p.pos++

	if token == "(" {
		return p.parseSequence(true)
	}

	if kind, ok := requiredUseOperatorKind(token); ok {
		children, err := p.parseParenthesizedChildren(token)
		if err != nil {
			return nil, err
		}

		return &RequiredUseNode{
			Kind:     kind,
			Children: children,
		}, nil
	}

	if strings.HasSuffix(token, "?") {
		condition := strings.TrimSuffix(token, "?")
		negated := strings.HasPrefix(condition, "!")
		condition = strings.TrimPrefix(condition, "!")
		if strings.TrimSpace(condition) == "" {
			return nil, fmt.Errorf("invalid REQUIRED_USE conditional %q", token)
		}

		children, err := p.parseParenthesizedChildren(token)
		if err != nil {
			return nil, err
		}

		return &RequiredUseNode{
			Kind:     RequiredUseConditional,
			Flag:     condition,
			Negated:  negated,
			Children: children,
		}, nil
	}

	if token == ")" {
		return nil, fmt.Errorf("unexpected ')' in REQUIRED_USE")
	}

	negated := strings.HasPrefix(token, "!")
	flag := strings.TrimPrefix(token, "!")
	if strings.TrimSpace(flag) == "" {
		return nil, fmt.Errorf("invalid REQUIRED_USE flag %q", token)
	}

	return &RequiredUseNode{
		Kind:    RequiredUseFlag,
		Flag:    flag,
		Negated: negated,
	}, nil
}

func (p *requiredUseParser) parseParenthesizedChildren(owner string) ([]*RequiredUseNode, error) {
	if p.pos >= len(p.tokens) || p.tokens[p.pos] != "(" {
		return nil, fmt.Errorf("expected '(' after %q in REQUIRED_USE", owner)
	}

	p.pos++
	sequence, err := p.parseSequence(true)
	if err != nil {
		return nil, err
	}

	return sequence.Children, nil
}

func tokenizeRequiredUse(raw string) []string {
	replacer := strings.NewReplacer(
		"(", " ( ",
		")", " ) ",
	)

	return strings.Fields(replacer.Replace(raw))
}

func requiredUseOperatorKind(token string) (RequiredUseNodeKind, bool) {
	switch token {
	case "||", "any-of":
		return RequiredUseAnyOf, true
	case "^^", "exactly-one-of":
		return RequiredUseExactlyOne, true
	case "??", "at-most-one-of":
		return RequiredUseAtMostOne, true
	default:
		return "", false
	}
}

func requiredUseNodeSatisfied(node *RequiredUseNode, enabled map[string]bool) bool {
	if node == nil {
		return true
	}

	switch node.Kind {
	case RequiredUseAll:
		for _, child := range node.Children {
			if !requiredUseNodeSatisfied(child, enabled) {
				return false
			}
		}
		return true

	case RequiredUseFlag:
		value := enabled[node.Flag]
		if node.Negated {
			return !value
		}
		return value

	case RequiredUseConditional:
		condition := enabled[node.Flag]
		if node.Negated {
			condition = !condition
		}
		if !condition {
			return true
		}
		for _, child := range node.Children {
			if !requiredUseNodeSatisfied(child, enabled) {
				return false
			}
		}
		return true

	case RequiredUseAnyOf:
		for _, child := range node.Children {
			if requiredUseNodeSatisfied(child, enabled) {
				return true
			}
		}
		return false

	case RequiredUseExactlyOne:
		count := 0
		for _, child := range node.Children {
			if requiredUseNodeSatisfied(child, enabled) {
				count++
			}
		}
		return count == 1

	case RequiredUseAtMostOne:
		count := 0
		for _, child := range node.Children {
			if requiredUseNodeSatisfied(child, enabled) {
				count++
			}
		}
		return count <= 1

	default:
		return false
	}
}

func evaluateRequiredUseViolations(
	node *RequiredUseNode,
	enabled map[string]bool,
	context []RequiredUseCondition,
) []RequiredUseViolation {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case RequiredUseAll:
		var out []RequiredUseViolation
		for _, child := range node.Children {
			out = append(out, evaluateRequiredUseViolations(child, enabled, context)...)
		}
		return out

	case RequiredUseFlag:
		if requiredUseNodeSatisfied(node, enabled) {
			return nil
		}

		return []RequiredUseViolation{{
			Kind:       RequiredUseFlag,
			Flag:       node.Flag,
			Negated:    node.Negated,
			Context:    cloneRequiredUseContext(context),
			Expression: requiredUseNodeString(node),
		}}

	case RequiredUseConditional:
		condition := enabled[node.Flag]
		if node.Negated {
			condition = !condition
		}
		if !condition {
			return nil
		}

		childContext := append(cloneRequiredUseContext(context), RequiredUseCondition{
			Flag:    node.Flag,
			Enabled: !node.Negated,
		})

		var out []RequiredUseViolation
		for _, child := range node.Children {
			out = append(out, evaluateRequiredUseViolations(child, enabled, childContext)...)
		}
		return out

	case RequiredUseAnyOf, RequiredUseExactlyOne, RequiredUseAtMostOne:
		if requiredUseNodeSatisfied(node, enabled) {
			return nil
		}

		terms, simpleFlags := describeRequiredUseTerms(node.Children)
		return []RequiredUseViolation{{
			Kind:        node.Kind,
			Terms:       terms,
			SimpleFlags: simpleFlags,
			Context:     cloneRequiredUseContext(context),
			Expression:  requiredUseNodeString(node),
		}}

	default:
		return []RequiredUseViolation{{
			Kind:       node.Kind,
			Context:    cloneRequiredUseContext(context),
			Expression: requiredUseNodeString(node),
		}}
	}
}

func describeRequiredUseTerms(nodes []*RequiredUseNode) ([]string, bool) {
	terms := make([]string, 0, len(nodes))
	simpleFlags := true

	for _, node := range nodes {
		terms = append(terms, requiredUseNodeString(node))
		if node == nil || node.Kind != RequiredUseFlag || node.Negated {
			simpleFlags = false
		}
	}

	return terms, simpleFlags
}

func requiredUseNodeString(node *RequiredUseNode) string {
	if node == nil {
		return ""
	}

	switch node.Kind {
	case RequiredUseAll:
		parts := make([]string, 0, len(node.Children))
		for _, child := range node.Children {
			parts = append(parts, requiredUseNodeString(child))
		}
		return strings.Join(parts, " ")

	case RequiredUseFlag:
		if node.Negated {
			return "!" + node.Flag
		}
		return node.Flag

	case RequiredUseConditional:
		condition := node.Flag
		if node.Negated {
			condition = "!" + condition
		}
		children := &RequiredUseNode{Kind: RequiredUseAll, Children: node.Children}
		return fmt.Sprintf("%s? ( %s )", condition, requiredUseNodeString(children))

	case RequiredUseAnyOf, RequiredUseExactlyOne, RequiredUseAtMostOne:
		operator := string(node.Kind)
		switch node.Kind {
		case RequiredUseAnyOf:
			operator = "||"
		case RequiredUseExactlyOne:
			operator = "^^"
		case RequiredUseAtMostOne:
			operator = "??"
		}
		children := &RequiredUseNode{Kind: RequiredUseAll, Children: node.Children}
		return fmt.Sprintf("%s ( %s )", operator, requiredUseNodeString(children))
	}

	return ""
}

func cloneRequiredUseContext(context []RequiredUseCondition) []RequiredUseCondition {
	if len(context) == 0 {
		return nil
	}

	return append([]RequiredUseCondition(nil), context...)
}

func collectRequiredUseBlock(lines []string, heading string, stopHeading string) string {
	collecting := false
	var collected []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !collecting {
			if trimmed == heading {
				collecting = true
			}
			continue
		}

		if stopHeading != "" && trimmed == stopHeading {
			break
		}

		if trimmed == "" {
			if len(collected) > 0 {
				break
			}
			continue
		}

		if strings.HasPrefix(trimmed, "(dependency required by") ||
			strings.HasPrefix(trimmed, "!!!") ||
			strings.HasPrefix(trimmed, "* ") {
			break
		}

		collected = append(collected, trimmed)
	}

	return strings.Join(collected, " ")
}

func parseUseAssignment(raw string) map[string]bool {
	out := make(map[string]bool)

	for _, token := range strings.Fields(raw) {
		token = strings.TrimSpace(token)
		token = strings.Trim(token, "()")
		if token == "" {
			continue
		}

		enabled := !strings.HasPrefix(token, "-")
		flag := strings.TrimPrefix(token, "-")
		if flag == "" {
			continue
		}

		out[flag] = enabled
	}

	return out
}

func exactAtomFromPackageSpec(spec string) string {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return ""
	}

	if strings.HasPrefix(spec, "=") {
		return spec
	}

	for _, prefix := range []string{">=", "<=", ">", "<", "~"} {
		if strings.HasPrefix(spec, prefix) {
			return spec
		}
	}

	if PackageNameFromSpec(spec) == spec {
		return spec
	}

	return "=" + spec
}
