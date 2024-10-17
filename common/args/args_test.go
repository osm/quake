package args

import (
	"encoding/base64"
	"reflect"
	"testing"
)

type argsTest struct {
	name     string
	input    string
	expected []Arg
}

var argsTests = []argsTest{
	{
		name:  "foo bar baz",
		input: "Zm9vIGJhciBiYXo=",
		expected: []Arg{
			{
				Cmd:  "foo",
				Args: []string{"bar", "baz"},
			},
		},
	},
	{
		name:  "foo bar baz\n",
		input: "Zm9vIGJhciBiYXoK",
		expected: []Arg{
			{
				Cmd:  "foo",
				Args: []string{"bar", "baz"},
			},
		},
	},
	{
		name:  "foo \"bar baz\"",
		input: "Zm9vICJiYXIgYmF6Ig==",
		expected: []Arg{
			{
				Cmd:  "foo",
				Args: []string{"\"bar baz\""},
			},
		},
	},
	{
		name:  "foo \"bar baz\";foo bar baz\nfoo bar",
		input: "Zm9vICJiYXIgYmF6Ijtmb28gYmFyIGJhegpmb28gYmFy",
		expected: []Arg{
			{
				Cmd:  "foo",
				Args: []string{"\"bar baz\""},
			},
			{
				Cmd:  "foo",
				Args: []string{"bar", "baz"},
			},
			{
				Cmd:  "foo",
				Args: []string{"bar"},
			},
		},
	},
}

func TestArgs(t *testing.T) {
	for _, at := range argsTests {
		t.Run(at.name, func(t *testing.T) {
			input, err := base64.StdEncoding.DecodeString(at.input)
			if err != nil {
				t.Errorf("unable to decode input: %#v", err)
			}

			args := Parse(string(input))

			if !reflect.DeepEqual(at.expected, args) {
				t.Errorf("parsed args output didn't match")
				t.Logf("output: %#v", args)
				t.Logf("expected: %#v", at.expected)
			}
		})
	}
}
