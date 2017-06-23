package bnf

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode"
)

type Production struct {
	Name  string
	Rules []Rule
}

func (p Production) String() string {
	return fmt.Sprintf("<%s>", p.Name)
}

type Rule struct {
	Items []Item
}

func (r Rule) String() string {
	var buf bytes.Buffer
	for i, item := range r.Items {
		if i != 0 {
			buf.WriteRune(' ')
		}
		buf.WriteString(item.String())
	}
	return buf.String()
}

type Item struct {
	Text         string
	IsProduction bool
	Subtract     bool
	Many         bool
	Optional     bool
	Expression   []Rule
}

func (i Item) String() string {
	if !i.IsProduction {
		return fmt.Sprintf("\"%s\"", i.Text)
	}

	return i.Text
}

func LoadFile(filename string) (map[string]Production, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	prods, err := getProductions(s)
	if err != nil {
		return nil, err
	}
	if err = validateProductions(prods); err != nil {
		return nil, err
	}
	return prods, nil
}

func validateProductions(prods map[string]Production) error {
	var errs bytes.Buffer
	for _, prod := range prods {
		for _, rule := range prod.Rules {
			for i, ruleItem := range rule.Items {
				if len(ruleItem.Expression) != 0 {
					// TODO (judwhite)
				} else if ruleItem.IsProduction {
					if _, ok := prods[ruleItem.Text]; !ok {
						if errs.Len() != 0 {
							errs.WriteString("\n")
						}
						msg := fmt.Sprintf("production '%s' rule '%s': production '%s' not found",
							prod.Name, rule, ruleItem.Text)
						errs.WriteString(msg)
					} else if ruleItem.Text == prod.Name && (i == 0 || i != len(rule.Items)-1) {
						if errs.Len() != 0 {
							errs.WriteString("\n")
						}
						msg := fmt.Sprintf("production '%s' rule '%s': rule has left tail recursion",
							prod.Name, rule)
						errs.WriteString(msg)
					}
				}
			}
		}
	}

	if errs.Len() > 0 {
		return errors.New(errs.String())
	}
	return nil
}

func getProductions(s *bufio.Scanner) (map[string]Production, error) {
	prods := make(map[string]Production)

	var prod Production
	for i := 0; s.Scan(); i++ {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "|") {
			if prod.Name == "" {
				return nil, fmt.Errorf("line %d: expected '<production>'", i+1)
			}
			rules, err := parseRules(line[1:])
			if err != nil {
				return nil, fmt.Errorf("line %d: %v", i+1, err)
			}
			prod.Rules = append(prod.Rules, rules...)
		} else {
			if prod.Name != "" {
				prods[prod.Name] = prod
			}
			var err error
			if prod, err = parseProduction(line); err != nil {
				return nil, fmt.Errorf("line %d: %v", i+1, err)
			}
			if _, ok := prods[prod.Name]; ok {
				return nil, fmt.Errorf("line %d: duplicate production name '%s'", i+1, prod.Name)
			}
		}
	}
	prods[prod.Name] = prod

	if err := s.Err(); err != nil {
		return nil, err
	}

	return prods, nil
}

func parseRules(line string) ([]Rule, error) {
	var rules []Rule
	line = strings.Replace(line, "\t", " ", -1)
	for len(line) > 0 {
		newline, set, err := parseSet(line)
		if err != nil {
			return nil, err
		}
		rules = append(rules, set...)
		line = newline
	}
	return rules, nil
}

func parseSet(line string) (string, []Rule, error) {
	var rules []Rule
	origLine := line
	for len(line) > 0 {

		var items []Item
		for len(line) > 0 {
			var idx int
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, ")") {
				// TODO (judwhite): make sure it had an open (
				return line[1:], rules, nil
			} else if strings.HasPrefix(line, `"`) {
				idx = strings.Index(line[1:], `"`) + 1
				if idx == 0 {
					return "", nil, fmt.Errorf("no closing \" found: %s", origLine)
				}
				items = append(items, Item{Text: line[1:idx]})
				line = line[idx+1:]
			} else if strings.HasPrefix(line, "'") {
				idx = strings.Index(line[1:], "'") + 1
				if idx == 0 {
					return "", nil, fmt.Errorf("no closing ' found: %s", origLine)
				}
				items = append(items, Item{Text: line[1:idx]})
				line = line[idx+1:]
			} else if strings.HasPrefix(line, ";") {
				// comment
				break
			} else if strings.HasPrefix(line, "#") {
				// code point
				if idx = strings.Index(line, "x"); idx != 1 {
					return "", nil, fmt.Errorf("expected 'x' after '#': %s", origLine)
				}

				var text string
				if idx = strings.Index(line, " "); idx == -1 {
					text = line
					line = ""
				} else {
					text = line[:idx]
					line = line[idx:]
				}
				// TODO (judwhite): validate hex
				items = append(items, Item{Text: text})
			} else if strings.HasPrefix(line, "[") {
				// sets, ranges
				// prefixed with ^, excludes set(s)
				idx = strings.Index(line, "]")
				if idx == -1 {
					return "", nil, fmt.Errorf("no closing ] found: %s", origLine)
				}
				items = append(items, Item{Text: line[:idx]})
				line = line[idx+1:]
			} else if strings.HasPrefix(line, "(") {
				// expression unit
				newline, expr, err := parseSet(line[1:])
				if err != nil {
					return "", nil, err
				}
				line = newline
				items = append(items, Item{Text: "", Expression: expr, IsProduction: true})
			} else if strings.HasPrefix(line, "|") {
				// new rule
				line = line[1:]
				break
				//} else {
				//	items = append(items, Item{Text: text, Terminal: terminal})
			} else if line[0] == '*' {
				items = append(items, Item{Many: true})
				line = line[1:]
			} else if line[0] == '?' {
				items = append(items, Item{Optional: true})
				line = line[1:]
			} else if line[0] == '-' {
				items = append(items, Item{Subtract: true})
				line = line[1:]
			} else if unicode.IsLetter(rune(line[0])) {
				// production
				for idx = 1; idx < len(line); idx++ {
					if !(line[idx] >= 'a' && line[idx] <= 'z' || line[idx] >= 'A' && line[idx] <= 'Z' ||
						line[idx] == '_' || line[idx] == '-' || line[idx] >= '0' && line[idx] <= '9') {
						break
					}
				}
				items = append(items, Item{Text: line[:idx], IsProduction: true})
				line = line[idx:]
			} else {
				return "", nil, fmt.Errorf("invalid character '%c': %s", line[0], origLine)
			}
		}
		if len(items) > 0 {
			rules = append(rules, Rule{Items: items})
		}
	}
	return "", rules, nil
}

func parseProduction(line string) (Production, error) {
	line = strings.TrimSpace(line)
	line = strings.Replace(line, "\t", " ", -1)
	if line == "" {
		return Production{}, fmt.Errorf("col 1: expected production name: %q", line)
	}
	idx := strings.Index(line, " ")
	if idx == -1 {
		return Production{}, fmt.Errorf("expected ' ': %q", line)
	}
	prod := Production{Name: line[:idx]}

	rhs := strings.TrimSpace(line[idx+1:])
	if !strings.HasPrefix(rhs, "::=") {
		return Production{}, fmt.Errorf("expected '::=': %q", line)
	}

	rules, err := parseRules(rhs[3:])
	if err != nil {
		return Production{}, err
	}
	prod.Rules = append(prod.Rules, rules...)
	return prod, nil
}
