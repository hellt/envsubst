package parse

import (
	"testing"
)

var FakeEnv = []string{
	"BAR=bar",
	"FOO=foo",
	"EMPTY=",
	"ALSO_EMPTY=",
}

type mode int

const (
	relaxed mode = iota
	noUnset
	noEmpty
	strict
)

var restrict = map[mode]*Restrictions{
	relaxed: Relaxed,
	noUnset: NoUnset,
	noEmpty: NoEmpty,
	strict:  Strict,
}

var errNone = map[mode]bool{}
var errUnset = map[mode]bool{noUnset: true, strict: true}
var errEmpty = map[mode]bool{noEmpty: true, strict: true}
var errAll = map[mode]bool{relaxed: true, noUnset: true, noEmpty: true, strict: true}
var errAllFull = map[mode]bool{relaxed: true, noUnset: true, noEmpty: true, strict: true}

type parseTest struct {
	name     string
	input    string
	expected string
	hasErr   map[mode]bool
}

var parseTests = []parseTest{
	{"empty", "", "", errNone},
	{"env only", "$BAR", "bar", errNone},
	{"with text", "$BAR baz", "bar baz", errNone},
	{"concatenated", "$BAR$FOO", "barfoo", errNone},
	{"2 env var", "$BAR - $FOO", "bar - foo", errNone},
	{"invalid var", "$_ bar", "$_ bar", errNone},
	{"invalid subst var", "${_} bar", "${_} bar", errNone},
	{"value of $var", "${BAR}baz", "barbaz", errNone},
	{"$var not set -", "${NOTSET-$BAR}", "bar", errNone},
	{"$var not set =", "${NOTSET=$BAR}", "bar", errNone},
	{"$var set but empty -", "${EMPTY-$BAR}", "", errEmpty},
	{"$var set but empty =", "${EMPTY=$BAR}", "", errEmpty},
	{"$var not set or empty :-", "${EMPTY:-$BAR}", "bar", errNone},
	{"$var not set or empty :=", "${EMPTY:=$BAR}", "bar", errNone},
	{"if $var set evaluate expression as $other +", "${EMPTY+hello}", "hello", errNone},
	{"if $var set evaluate expression as $other :+", "${EMPTY:+hello}", "hello", errNone},
	{"if $var not set, use empty string +", "${NOTSET+hello}", "", errNone},
	{"if $var not set, use empty string :+", "${NOTSET:+hello}", "", errNone},
	{"multi line string", "hello $BAR\nhello ${EMPTY:=$FOO}", "hello bar\nhello foo", errNone},
	{"issue #1", "${hello:=wo_rld} ${foo:=bar_baz}", "wo_rld bar_baz", errNone},
	{"issue #2", "name: ${NAME:=foo_qux}, key: ${EMPTY:=baz_bar}", "name: foo_qux, key: baz_bar", errNone},
	{"gh-issue-8", "prop=${HOME_URL-http://localhost:8080}", "prop=http://localhost:8080", errNone},
	// bad substitution
	{"closing brace expected", "hello ${", "", errAll},

	// test specifically for failure modes
	{"$var not set", "${NOTSET}", "", errUnset},
	{"$var set to empty", "${EMPTY}", "", errEmpty},
	// restrictions for plain variables without braces
	{"gh-issue-9", "$NOTSET", "", errUnset},
	{"gh-issue-9", "$EMPTY", "", errEmpty},

	{"$var and $DEFAULT not set -", "${NOTSET-$ALSO_NOTSET}", "", errUnset},
	{"$var and $DEFAULT not set :-", "${NOTSET:-$ALSO_NOTSET}", "", errUnset},
	{"$var and $DEFAULT not set =", "${NOTSET=$ALSO_NOTSET}", "", errUnset},
	{"$var and $DEFAULT not set :=", "${NOTSET:=$ALSO_NOTSET}", "", errUnset},
	{"$var and $OTHER not set +", "${NOTSET+$ALSO_NOTSET}", "", errNone},
	{"$var and $OTHER not set :+", "${NOTSET:+$ALSO_NOTSET}", "", errNone},

	{"$var empty and $DEFAULT not set -", "${EMPTY-$NOTSET}", "", errEmpty},
	{"$var empty and $DEFAULT not set :-", "${EMPTY:-$NOTSET}", "", errUnset},
	{"$var empty and $DEFAULT not set =", "${EMPTY=$NOTSET}", "", errEmpty},
	{"$var empty and $DEFAULT not set :=", "${EMPTY:=$NOTSET}", "", errUnset},
	{"$var empty and $OTHER not set +", "${EMPTY+$NOTSET}", "", errUnset},
	{"$var empty and $OTHER not set :+", "${EMPTY:+$NOTSET}", "", errUnset},

	{"$var not set and $DEFAULT empty -", "${NOTSET-$EMPTY}", "", errEmpty},
	{"$var not set and $DEFAULT empty :-", "${NOTSET:-$EMPTY}", "", errEmpty},
	{"$var not set and $DEFAULT empty =", "${NOTSET=$EMPTY}", "", errEmpty},
	{"$var not set and $DEFAULT empty :=", "${NOTSET:=$EMPTY}", "", errEmpty},
	{"$var not set and $OTHER empty +", "${NOTSET+$EMPTY}", "", errNone},
	{"$var not set and $OTHER empty :+", "${NOTSET:+$EMPTY}", "", errNone},

	{"$var and $DEFAULT empty -", "${EMPTY-$ALSO_EMPTY}", "", errEmpty},
	{"$var and $DEFAULT empty :-", "${EMPTY:-$ALSO_EMPTY}", "", errEmpty},
	{"$var and $DEFAULT empty =", "${EMPTY=$ALSO_EMPTY}", "", errEmpty},
	{"$var and $DEFAULT empty :=", "${EMPTY:=$ALSO_EMPTY}", "", errEmpty},
	{"$var and $OTHER empty +", "${EMPTY+$ALSO_EMPTY}", "", errEmpty},
	{"$var and $OTHER empty :+", "${EMPTY:+$ALSO_EMPTY}", "", errEmpty},

	// escaping.
	{"escape $$var", "FOO $$BAR BAZ", "FOO $BAR BAZ", errNone},
	{"escape $${subst}", "FOO $${BAR} BAZ", "FOO ${BAR} BAZ", errNone},
	{"escape $$$var", "$$$BAR", "$bar", errNone},
	{"escape $$${subst}", "$$${BAZ:-baz}", "$baz", errNone},
}

var negativeParseTests = []parseTest{
	{"$NOTSET and EMPTY are displayed as in full error output", "${NOTSET} and $EMPTY", "variable ${NOTSET} not set\nvariable ${EMPTY} set but empty", errAllFull},
}

func TestParse(t *testing.T) {
	doTest(t, relaxed)
}

func TestParseNoUnset(t *testing.T) {
	doTest(t, noUnset)
}

func TestParseNoEmpty(t *testing.T) {
	doTest(t, noEmpty)
}

func TestParseStrict(t *testing.T) {
	doTest(t, strict)
}

func TestParseStrictNoFailFast(t *testing.T) {
	doNegativeAssertTest(t, strict)
}

func doTest(t *testing.T, m mode) {
	for _, test := range parseTests {
		result, err := New(test.name, FakeEnv, restrict[m]).Parse(test.input)
		hasErr := err != nil
		if hasErr != test.hasErr[m] {
			t.Errorf("%s=(error): got\n\t%v\nexpected\n\t%v\ninput: %s\nresult: %s\nerror: %v",
				test.name, hasErr, test.hasErr[m], test.input, result, err)
		}
		if result != test.expected {
			t.Errorf("%s=(%q): got\n\t%v\nexpected\n\t%v", test.name, test.input, result, test.expected)
		}
	}
}

func doNegativeAssertTest(t *testing.T, m mode) {
	for _, test := range negativeParseTests {
		result, err := (&Parser{Name: test.name, Env: FakeEnv, Restrict: restrict[m], Mode: AllErrors}).Parse(test.input)
		hasErr := err != nil
		if hasErr != test.hasErr[m] {
			t.Errorf("%s=(error): got\n\t%v\nexpected\n\t%v\ninput: %s\nresult: %s\nerror: %v",
				test.name, hasErr, test.hasErr[m], test.input, result, err)
		}
		if err.Error() != test.expected {
			t.Errorf("%s=(%q): got\n\t%v\nexpected\n\t%v", test.name, test.input, err.Error(), test.expected)
		}
	}
}

func TestNoReplace(t *testing.T) {
	ttests := map[string]struct {
		input        string
		env          []string
		restrictions *Restrictions
		expected     string
	}{
		"correctly set restrictions for no replace": {
			`Some: $REPLACE
		NoReplace: Stuff$ToIgnore!d`,
			[]string{"REPLACE=bar"},
			&Restrictions{false, false, true, true},
			`Some: bar
		NoReplace: Stuff$ToIgnore!d`,
		},

		"if default is set, noreplace should not be used": {
			input: `ExistingEnvVarIsReplaced: $REPLACE
		NoReplaceNotToBeUsedWithDefault: ${someVarWithDefault:=myDefault}
		NoReplaceShouldNotReplaceNonExistingEnvVar: $ToIgnore`,
			env:          []string{"REPLACE=bar"},
			restrictions: &Restrictions{NoUnset: false, NoEmpty: false, NoDigit: true, NoReplace: true},
			expected: `ExistingEnvVarIsReplaced: bar
		NoReplaceNotToBeUsedWithDefault: myDefault
		NoReplaceShouldNotReplaceNonExistingEnvVar: $ToIgnore`,
		},

		"if unset true - should error": {
			`Some: $REPLACE
		NoReplace: Stuff$ToIgnore!d`,
			[]string{"REPLACE=bar"},
			&Restrictions{true, false, true, true},
			`variable ${ToIgnore} not set`,
		},

		"noreplace set to false should return empty string": {
			`Some: $REPLACE
		NoReplace: Stuff$ToIgnore!d`,
			[]string{"REPLACE=bar"},
			&Restrictions{false, false, true, false},
			`Some: bar
		NoReplace: Stuff!d`,
		},
	}
	for name, test := range ttests {
		t.Run(name, func(t *testing.T) {
			result, err := (&Parser{Name: name, Env: test.env, Restrict: test.restrictions, Mode: AllErrors}).Parse(test.input)
			if err != nil {
				if err.Error() != test.expected {
					t.Errorf("error\n%s=(%q): got\n\t%v\nexpected\n\t%v", name, test.input, err.Error(), test.expected)
				}
				return
			}
			if result != test.expected {
				t.Errorf("%s=(%q): got\n\t%v\nexpected\n\t%v", name, test.input, result, test.expected)
			}
		})
	}
}
