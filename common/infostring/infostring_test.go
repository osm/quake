package infostring

import (
	"reflect"
	"testing"
)

type infoStringTest struct {
	name     string
	input    string
	expected InfoString
}

var infoStringTests = []infoStringTest{
	{
		name:  "foo",
		input: "\"\\FOO\\foo\\BAR\\bar\\BAZ\\baz\"",
		expected: InfoString{
			Info: []Info{
				Info{Key: "FOO", Value: "foo"},
				Info{Key: "BAR", Value: "bar"},
				Info{Key: "BAZ", Value: "baz"},
			},
		},
	},
	{
		name:  "\\foo\\with spaces",
		input: "\"\\foo\\with spaces\"",
		expected: InfoString{
			Info: []Info{
				Info{Key: "foo", Value: "with spaces"},
			},
		},
	},
}

func TestInfoString(t *testing.T) {
	for _, is := range infoStringTests {
		t.Run(is.name, func(t *testing.T) {
			infoString := Parse(is.input)

			if !reflect.DeepEqual(is.expected.Bytes(), infoString.Bytes()) {
				t.Errorf("parsed infostring output didn't match")
				t.Logf("output: %#v\n", infoString)
				t.Logf("expected: %#v\n", is.expected)
			}
		})
	}
}
