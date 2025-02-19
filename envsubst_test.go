package envsubst

import (
	"os"
	"testing"
)

func init() {
	os.Setenv("BAR", "bar")
}

// Basic integration tests. because we  already test the
// templating processing in envsubst/parse;
func TestIntegration(t *testing.T) {
	input, expected := "foo $BAR", "foo bar"
	str, err := String(input)
	if str != expected || err != nil {
		t.Error("Expect string integration test to pass")
	}
	bytes, err := Bytes([]byte(input))
	if string(bytes) != expected || err != nil {
		t.Error("Expect bytes integration test to pass")
	}
	bytes, err = ReadFile("testdata/file.tmpl")
	fexpected, err := os.ReadFile("testdata/file.out")
	if string(bytes) != string(fexpected) || err != nil {
		t.Error("Expect ReadFile integration test to pass")
	}
}
