package bnf

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
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
	Text     string
	Terminal bool
}

func (i Item) String() string {
	if i.Terminal {
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
				if !ruleItem.Terminal {
					if _, ok := prods[ruleItem.Text]; !ok {
						if errs.Len() != 0 {
							errs.WriteString("; ")
						}
						msg := fmt.Sprintf("production '%s' rule '%s': production '%s' not found",
							prod.Name, rule, ruleItem.Text)
						errs.WriteString(msg)
					} else if ruleItem.Text == prod.Name && (i == 0 || i != len(rule.Items)-1) {
						if errs.Len() != 0 {
							errs.WriteString("; ")
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
	for len(line) > 0 {
		line = strings.TrimSpace(line)
		var items []Item
		for len(line) > 0 {
			var idx int
			if strings.HasPrefix(line, `"`) {
				idx = strings.Index(line[1:], `"`) + 1
				if idx == -1 {
					return nil, fmt.Errorf("no closing '\"' found")
				}
				items = append(items, Item{Text: line[1:idx], Terminal: true})
			} else if strings.HasPrefix(line, `<`) {
				idx = strings.Index(line, `>`)
				if idx == -1 {
					return nil, fmt.Errorf("no closing '>' found")
				}
				terminal := false
				text := line[1:idx]
				switch text {
				case "tab":
					text = "\t"
					terminal = true
				case "cr":
					text = "\r"
					terminal = true
				case "lf":
					text = "\n"
					terminal = true
				}

				items = append(items, Item{Text: text, Terminal: terminal})
			} else if strings.HasPrefix(line, `;`) {
				// comment
				break
			} else if strings.HasPrefix(line, "|") {
				// new rule
				line = line[1:]
				break
			} else {
				return nil, fmt.Errorf("expected '\"' or '<', found '%c'", line[0])
			}
			line = line[idx+1:]
			if strings.HasPrefix(line, " ") {
				items = append(items, Item{Text: "opt-ws", Terminal: false})
				line = strings.TrimSpace(line)
			}
		}
		if len(items) > 0 {
			for items[len(items)-1].Text == "opt-ws" && !items[len(items)-1].Terminal {
				items = items[:len(items)-1]
			}
			rules = append(rules, Rule{Items: items})
		}
	}
	return rules, nil
}

func parseProduction(line string) (Production, error) {
	line = strings.TrimSpace(line)
	if line == "" || line[0] != '<' {
		return Production{}, fmt.Errorf("col 1: expected '<'")
	}
	idx := strings.Index(line, ">")
	if idx == -1 {
		return Production{}, fmt.Errorf("expected '>'")
	}
	prod := Production{Name: line[1:idx]}

	rhs := strings.TrimSpace(line[idx+1:])
	if !strings.HasPrefix(rhs, "::=") {
		return Production{}, fmt.Errorf("expected '::='")
	}

	rules, err := parseRules(rhs[3:])
	if err != nil {
		return Production{}, err
	}
	prod.Rules = append(prod.Rules, rules...)
	return prod, nil
}
