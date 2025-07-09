package config

import (
	"os"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

type rule struct {
	ident       string
	parsedIdent []int
}

func TestSortedTrapRules(t *testing.T) {
	f, err := os.ReadFile("testdata/traprule.yml")
	if err != nil {
		t.Error(err)
	}

	var conf Config
	err = yaml.Unmarshal(f, &conf)
	if err != nil {
		t.Error(err)
	}

	expected := []rule{
		{
			ident:       ".1.3.6.1.6.3.1.1.5.1",
			parsedIdent: []int{1, 3, 6, 1, 6, 3, 1, 1, 5, 1},
		},
		{
			ident:       ".1.3.6.1.6.3.1.1.5.2",
			parsedIdent: []int{1, 3, 6, 1, 6, 3, 1, 1, 5, 2},
		},
		{
			ident:       ".1.3.6.1.6.3.1.1.5.3",
			parsedIdent: []int{1, 3, 6, 1, 6, 3, 1, 1, 5, 3},
		},
		{
			ident:       ".1.3.6.1.6.3.1.1.5.4",
			parsedIdent: []int{1, 3, 6, 1, 6, 3, 1, 1, 5, 4},
		},
		{
			ident:       ".1.3.6.1.6.3.1.1.4",
			parsedIdent: []int{1, 3, 6, 1, 6, 3, 1, 1, 4},
		},
		{
			ident:       ".1.3.6.1.6.3.1.1.5",
			parsedIdent: []int{1, 3, 6, 1, 6, 3, 1, 1, 5},
		},
		{
			ident:       ".1.3.6.1.6.3.1.1",
			parsedIdent: []int{1, 3, 6, 1, 6, 3, 1, 1},
		},
		{
			ident:       ".1.3.6.1.6.3.1.2",
			parsedIdent: []int{1, 3, 6, 1, 6, 3, 1, 2},
		},
		{
			ident:       ".1.3.6.1.6.3.1",
			parsedIdent: []int{1, 3, 6, 1, 6, 3, 1},
		},
		{
			ident:       ".1.3.6.1.6.3.11",
			parsedIdent: []int{1, 3, 6, 1, 6, 3, 11},
		},
		{
			ident:       ".1.3.6.1.6.3.1111",
			parsedIdent: []int{1, 3, 6, 1, 6, 3, 1111},
		},
		{
			ident:       ".1.3.6.1.6.3.9",
			parsedIdent: []int{1, 3, 6, 1, 6, 3, 9},
		},
		{
			ident:       ".1.3.6.1.6.1",
			parsedIdent: []int{1, 3, 6, 1, 6, 1},
		},
		{
			ident:       ".1.3.6.1.6.3",
			parsedIdent: []int{1, 3, 6, 1, 6, 3},
		},
		{
			ident:       ".1.3.6.1.6.4",
			parsedIdent: []int{1, 3, 6, 1, 6, 4},
		},
		{
			ident:       ".1.3.6.1",
			parsedIdent: []int{1, 3, 6, 1},
		},
	}
	var actual []rule

	rules, err := conf.SortedTrapRules()
	if err != nil {
		t.Error(err)
	}
	for _, r := range rules {
		actual = append(actual, rule{ident: r.Ident, parsedIdent: r.ParsedIdent})
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Error("invalid")
	}
}
