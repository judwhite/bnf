package bnf

import "testing"

func TestLoadFileMath(t *testing.T) {
	_, err := LoadFile("math.bnf")
	if err != nil {
		t.Fatal(err)
	}
}

/*
func TestLoadFileSQL(t *testing.T) {
	_, err := LoadFile("sql.bnf")
	if err != nil {
		t.Fatal(err)
	}
}
*/
