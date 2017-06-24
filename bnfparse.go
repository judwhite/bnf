package bnf

import (
	"fmt"
	"strings"
)

type AST struct {
	Root ASTNode
}

func (a AST) Print() {
	a.Root.Print(0)
}

type ASTNode struct {
	Production *Production
	Rule       Rule
	Children   []ASTNode
	Literal    string
	Input      string
}

func (n ASTNode) Print(depth int) {
	if depth > 0 {
		fmt.Printf(strings.Repeat(" ", (depth-1)*2))
		fmt.Printf("|-- ")
	}

	if n.Production != nil {
		fmt.Printf("%s, rule: `%s`, consumed: %q\n", n.Production, n.Rule, n.Input)
		for _, child := range n.Children {
			child.Print(depth + 1)
		}
		return
	}

	fmt.Printf("terminal: %q\n", n.Literal)
}

func Parse(root string, prods map[string]Production, text string) (AST, error) {
	rootProd, ok := prods[root]
	if !ok {
		return AST{}, fmt.Errorf("production '%s' not found", root)
	}

	leftover, node, ok := rootProd.Parse(prods, text)
	if !ok {
		return AST{}, fmt.Errorf("production '%s' could not parse '%s'", root, leftover)
	}

	if len(leftover) > 0 {
		return AST{}, fmt.Errorf("unparsed text: '%s'", leftover)
	}

	return AST{Root: node}, nil
}

func (p Production) Parse(prodMap map[string]Production, text string) (string, ASTNode, bool) {
	for _, rule := range p.Rules {
		newText, children, ok := rule.Parse(prodMap, text)
		if ok {
			node := ASTNode{Production: &p, Rule: rule, Children: children, Input: text[:len(text)-len(newText)]}
			return newText, node, true
		}
	}
	return text, ASTNode{}, false
}

func (r Rule) Parse(prodMap map[string]Production, text string) (string, []ASTNode, bool) {
	ruleText := text

	var children []ASTNode
	for _, item := range r.Items {
		var (
			ok   bool
			node ASTNode
		)
		ruleText, node, ok = item.Consume(prodMap, ruleText)
		if !ok {
			return text, nil, false
		}
		children = append(children, node)
	}

	return ruleText, children, true
}

func (i Item) Consume(prodMap map[string]Production, text string) (string, ASTNode, bool) {
	if len(i.Expression) > 0 {
		for _, rule := range i.Expression {
			newText, nodes, ok := rule.Parse(prodMap, text)
			if ok {
				return newText, ASTNode{Rule: rule, Children: nodes}, ok
			}
		}
		return text, ASTNode{}, false
	}

	if i.IsProduction {
		prod := prodMap[i.Text]
		newText, node, ok := prod.Parse(prodMap, text)
		return newText, node, ok
	}

	fmt.Printf(i.Text)
	if strings.HasPrefix(text, i.Text) {
		return text[len(i.Text):], ASTNode{Literal: i.Text, Input: i.Text}, true
	}
	return text, ASTNode{}, false
}
