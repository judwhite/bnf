package bnf

import (
	"fmt"
	"strconv"
	"testing"
)

func TestMathParse(t *testing.T) {
	prods, err := LoadFile("math.bnf")
	if err != nil {
		t.Fatal(err)
	}
	ast, err := Parse("real", prods, "-3.14")
	if err != nil {
		t.Fatal(err)
	}
	ast.Print()

	expr := eval(ast.Root)
	fmt.Printf("%v %T\n", expr, expr)
}

func eval(node ASTNode) interface{} {
	if node.Production != nil {
		switch node.Production.Name {
		case "real":
			return eval(node.Children[0])
		case "integer":
			return evalInteger(node)
		case "fraction":
			return evalFraction(node)
		}
	}
	return nil
}

func evalInteger(node ASTNode) int {
	v, err := strconv.Atoi(node.Input)
	if err != nil {
		panic(err)
	}
	return v
}

func evalFraction(node ASTNode) float64 {
	v, err := strconv.ParseFloat(node.Input, 64)
	if err != nil {
		panic(err)
	}
	return v
}

func TestXMLParse(t *testing.T) {
	prods, err := LoadFile("xml_test.bnf")
	if err != nil {
		t.Fatal(err)
	}
	ast, err := Parse("Comment", prods, "<!-- declarations for <head> & <body> -->")
	if err != nil {
		t.Fatal(err)
	}
	ast.Print()
}
