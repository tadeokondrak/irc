package irc

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// https://github.com/ircdocs/parser-tests/blob/master/tests/msg-split.yaml
var splitTests = []struct {
	Input           string
	ExpectedRest    string
	ExpectedTags    Tags
	ExpectedPrefix  Prefix
	ExpectedCommand string
	ExpectedParams  []string
}{
	{"foo bar baz asdf", "",
		Tags{},
		Prefix{},
		"FOO",
		[]string{"bar", "baz", "asdf"}},
	{":coolguy foo bar baz asdf", "",
		Tags{},
		Prefix{Name: "coolguy"},
		"FOO",
		[]string{"bar", "baz", "asdf"}},
	{":coolguy foo bar baz :asdf quux", "",
		Tags{},
		Prefix{Name: "coolguy"},
		"FOO",
		[]string{"bar", "baz", "asdf quux"}},
	{":coolguy foo bar baz :", "",
		Tags{},
		Prefix{Name: "coolguy"},
		"FOO",
		[]string{"bar", "baz", ""}},
	{":coolguy foo bar baz ::asdf", "",
		Tags{},
		Prefix{Name: "coolguy"},
		"FOO",
		[]string{"bar", "baz", ":asdf"}},
	{":coolguy foo bar baz :asdf quux", "",
		Tags{},
		Prefix{Name: "coolguy"},
		"FOO",
		[]string{"bar", "baz", "asdf quux"}},
	{":coolguy foo bar baz :  asdf quux ", "",
		Tags{},
		Prefix{Name: "coolguy"},
		"FOO",
		[]string{"bar", "baz", "  asdf quux "}},
	{":coolguy PRIVMSG bar :lol :) ", "",
		Tags{},
		Prefix{Name: "coolguy"},
		"PRIVMSG",
		[]string{"bar", "lol :) "}},
	{":coolguy foo bar baz :", "",
		Tags{},
		Prefix{Name: "coolguy"},
		"FOO",
		[]string{"bar", "baz", ""}},
	{":coolguy foo bar baz :  ", "",
		Tags{},
		Prefix{Name: "coolguy"},
		"FOO",
		[]string{"bar", "baz", "  "}},
	{"@a=b;c=32;k;rt=ql7 foo", "",
		Tags{"a": "b", "c": "32", "k": "", "rt": "ql7"},
		Prefix{},
		"FOO",
		[]string{}},
	{"@a=b\\\\and\\nk;c=72\\s45;d=gh\\:764 foo", "",
		Tags{"a": "b\\and\nk", "c": "72 45", "d": "gh;764"},
		Prefix{},
		"FOO",
		[]string{}},
	{"@c;h=;a=b :quux ab cd", "",
		Tags{"c": "", "h": "", "a": "b"},
		Prefix{Name: "quux"},
		"AB",
		[]string{"cd"}},
	{":src JOIN #chan", "",
		Tags{},
		Prefix{Name: "src"},
		"JOIN",
		[]string{"#chan"}},
	{":src JOIN :#chan", "",
		Tags{},
		Prefix{Name: "src"},
		"JOIN",
		[]string{"#chan"}},
	{":src AWAY", "",
		Tags{},
		Prefix{Name: "src"},
		"AWAY",
		[]string{}},
	{":src AWAY ", "",
		Tags{},
		Prefix{Name: "src"},
		"AWAY",
		[]string{}},
	{":cool\tguy foo bar baz", "",
		Tags{},
		Prefix{Name: "cool\tguy"},
		"FOO",
		[]string{"bar", "baz"}},
	{":coolguy!ag@net\x035w\x03ork.admin PRIVMSG foo :bar baz", "",
		Tags{},
		Prefix{Name: "coolguy", User: "ag",
			Host: "net\x035w\x03ork.admin"},
		"PRIVMSG",
		[]string{"foo", "bar baz"}},
	{":coolguy!~ag@n\x02et\x0305w\x0fork.admin PRIVMSG foo :bar baz", "",
		Tags{},
		Prefix{Name: "coolguy", User: "~ag",
			Host: "n\x02et\x0305w\x0fork.admin"},
		"PRIVMSG",
		[]string{"foo", "bar baz"}},
	{"@tag1=value1;tag2;vendor1/tag3=value2;vendor2/tag4= " +
		":irc.example.com COMMAND param1 param2 :param3 param3", "",
		Tags{"tag1": "value1", "tag2": "",
			"vendor1/tag3": "value2", "vendor2/tag4": ""},
		Prefix{Name: "irc.example.com"},
		"COMMAND",
		[]string{"param1", "param2", "param3 param3"}},
	{"@tag1=value1;tag2;vendor1/tag3=value2;vendor2/tag4 " +
		"COMMAND param1 param2 :param3 param3", "",
		Tags{"tag1": "value1", "tag2": "",
			"vendor1/tag3": "value2", "vendor2/tag4": ""},
		Prefix{},
		"COMMAND",
		[]string{"param1", "param2", "param3 param3"}},
	{"@foo=\\\\\\\\\\:\\\\s\\s\\r\\n COMMAND", "",
		Tags{"foo": "\\\\;\\s \r\n"},
		Prefix{},
		"COMMAND",
		[]string{}},
	{":gravel.mozilla.org MODE #tckk +n ", "",
		Tags{},
		Prefix{Name: "gravel.mozilla.org"},
		"MODE",
		[]string{"#tckk", "+n"}},
	{":services.esper.net MODE #foo-bar +o foobar  ", "",
		Tags{},
		Prefix{Name: "services.esper.net"},
		"MODE",
		[]string{"#foo-bar", "+o", "foobar"}},
	{"@tag1=value\\\\ntest COMMAND", "",
		Tags{"tag1": "value\\ntest"},
		Prefix{},
		"COMMAND",
		[]string{}},
	{"@tag1=value\\1 COMMAND", "",
		Tags{"tag1": "value1"},
		Prefix{},
		"COMMAND",
		[]string{}},
	{"@tag1=value1\\ COMMAND", "",
		Tags{"tag1": "value1"},
		Prefix{},
		"COMMAND",
		[]string{}},
	{"@tag1=1;tag2=3;tag3=4;tag1=5 COMMAND", "",
		Tags{"tag1": "5", "tag2": "3", "tag3": "4"},
		Prefix{},
		"COMMAND",
		[]string{}},
	{"@tag1=1;tag2=3;tag3=4;tag1=5;vendor/tag2=8 COMMAND", "",
		Tags{"tag1": "5", "tag2": "3", "tag3": "4", "vendor/tag2": "8"},
		Prefix{},
		"COMMAND",
		[]string{}},
	{":SomeOp MODE #channel :+i", "",
		Tags{},
		Prefix{Name: "SomeOp"},
		"MODE",
		[]string{"#channel", "+i"}},
	{":SomeOp MODE #channel +oo SomeUser :AnotherUser", "",
		Tags{},
		Prefix{Name: "SomeOp"},
		"MODE",
		[]string{"#channel", "+oo", "SomeUser", "AnotherUser"}},

	{"COMMAND with utf-8 param €", "",
		Tags{},
		Prefix{},
		"COMMAND",
		[]string{"with", "utf-8", "param", "€"}},
	{"COMMAND with crlf\r\n", "",
		Tags{},
		Prefix{},
		"COMMAND",
		[]string{"with", "crlf"}},
	{":prefix-name-with-crlf\r\n", "",
		Tags{},
		Prefix{Name: "prefix-name-with-crlf"},
		"",
		[]string{}},
	{":!prefix-user-with-crlf\r\n", "",
		Tags{},
		Prefix{User: "prefix-user-with-crlf"},
		"",
		[]string{}},
	{":@prefix-host-with-crlf\r\n", "",
		Tags{},
		Prefix{Host: "prefix-host-with-crlf"},
		"",
		[]string{}},
	{"", "",
		Tags{},
		Prefix{},
		"",
		[]string{}},
	{":test", "",
		Tags{},
		Prefix{Name: "test"},
		"",
		[]string{}},
	{":!", "",
		Tags{},
		Prefix{Name: ""},
		"",
		[]string{}},
}

func TestSplit(t *testing.T) {
	t.Parallel()
	for _, test := range splitTests {
		test := test
		t.Run(test.Input, func(t *testing.T) {
			t.Parallel()
			input := []byte(test.Input)
			parsed, n := Parse(input)
			if !reflect.DeepEqual(parsed.Tags, test.ExpectedTags) {
				t.Logf("tags: expected %#v but got %#v",
					test.ExpectedTags, parsed.Tags)
				t.Fail()
			}
			if !reflect.DeepEqual(parsed.Prefix, test.ExpectedPrefix) {
				t.Logf("prefix: expected %#v but got %#v",
					test.ExpectedPrefix, parsed.Prefix)
				t.Fail()
			}
			if parsed.Command != test.ExpectedCommand {
				t.Logf("command: expected %#v but got %#v",
					test.ExpectedCommand, parsed.Command)
				t.Fail()
			}
			if !reflect.DeepEqual(parsed.Params, test.ExpectedParams) {
				t.Logf("params: expected %#v but got %#v",
					test.ExpectedParams, parsed.Params)
				t.Fail()
			}
			if !reflect.DeepEqual([]byte(test.ExpectedRest), input[n:]) {
				t.Logf("rest: expected %#v but got %#v",
					[]byte(test.ExpectedRest), input[n:])
				t.Fail()
			}
		})
	}
}

// https://github.com/ircdocs/parser-tests/blob/master/tests/msg-join.yaml
var joinTests = []struct {
	Input   Message
	Allowed []string
}{
	{Message{
		Command: "FOO",
		Params:  []string{"bar", "baz", "asdf"},
	}, []string{"FOO bar baz asdf", "FOO bar baz :asdf"}},
	{Message{
		Prefix:  Prefix{Name: "src"},
		Command: "AWAY",
	}, []string{":src AWAY"}},
	{Message{
		Prefix:  Prefix{Name: "src"},
		Command: "AWAY",
		Params:  []string{""},
	}, []string{":src AWAY :"}},
	{Message{
		Prefix:  Prefix{Name: "coolguy"},
		Command: "FOO",
		Params:  []string{"bar", "baz", "asdf"},
	}, []string{":coolguy FOO bar baz asdf", ":coolguy FOO bar baz :asdf"}},
	{Message{
		Prefix:  Prefix{Name: "coolguy"},
		Command: "FOO",
		Params:  []string{"bar", "baz", "asdf quux"},
	}, []string{":coolguy FOO bar baz :asdf quux"}},
	{Message{
		Command: "FOO",
		Params:  []string{"bar", "baz", ""},
	}, []string{"FOO bar baz :"}},
	{Message{
		Command: "FOO",
		Params:  []string{"bar", "baz", ":asdf"},
	}, []string{"FOO bar baz ::asdf"}},
	{Message{
		Prefix:  Prefix{Name: "coolguy"},
		Command: "FOO",
		Params:  []string{"bar", "baz", "asdf quux"},
	}, []string{":coolguy FOO bar baz :asdf quux"}},
	{Message{
		Prefix:  Prefix{Name: "coolguy"},
		Command: "FOO",
		Params:  []string{"bar", "baz", "  asdf quux "},
	}, []string{":coolguy FOO bar baz :  asdf quux "}},
	{Message{
		Prefix:  Prefix{Name: "coolguy"},
		Command: "PRIVMSG",
		Params:  []string{"bar", "lol :) "},
	}, []string{":coolguy PRIVMSG bar :lol :) "}},
	{Message{
		Prefix:  Prefix{Name: "coolguy"},
		Command: "FOO",
		Params:  []string{"bar", "baz", ""},
	}, []string{":coolguy FOO bar baz :"}},
	{Message{
		Prefix:  Prefix{Name: "coolguy"},
		Command: "FOO",
		Params:  []string{"bar", "baz", "  "},
	}, []string{":coolguy FOO bar baz :  "}},
	{Message{
		Prefix:  Prefix{Name: "coolguy"},
		Command: "FOO",
		Params:  []string{"b\tar", "baz"},
	}, []string{":coolguy FOO b\tar baz", ":coolguy FOO b\tar :baz"}},
	{Message{
		Tags:    Tags{"asd": ""},
		Prefix:  Prefix{Name: "coolguy"},
		Command: "FOO",
		Params:  []string{"bar", "baz", "  "},
	}, []string{"@asd :coolguy FOO bar baz :  "}},
	{Message{
		Tags:    Tags{"a": "b\\and\nk", "d": "gh;764"},
		Command: "FOO",
	}, []string{
		"@a=b\\\\and\\nk;d=gh\\:764 FOO",
		"@d=gh\\:764;a=b\\\\and\\nk FOO"}},
	{Message{
		Tags:    Tags{"a": "b\\and\nk", "d": "gh;764"},
		Command: "FOO",
		Params:  []string{"par1", "par2"},
	}, []string{
		"@a=b\\\\and\\nk;d=gh\\:764 FOO par1 par2",
		"@a=b\\\\and\\nk;d=gh\\:764 FOO par1 :par2",
		"@d=gh\\:764;a=b\\\\and\\nk FOO par1 par2",
		"@d=gh\\:764;a=b\\\\and\\nk FOO par1 :par2"}},
	{Message{
		Tags:    Tags{"foo": "\\\\;\\s \r\n"},
		Command: "COMMAND",
	}, []string{"@foo=\\\\\\\\\\:\\\\s\\s\\r\\n COMMAND"}},
}

func TestJoin(t *testing.T) {
	t.Parallel()
	for _, test := range joinTests {
		test := test
		t.Run(test.Allowed[0], func(t *testing.T) {
			t.Parallel()
			found := false
			joined := test.Input.String()
			for _, allowed := range test.Allowed {
				if joined == allowed {
					found = true
				}
			}
			if !found {
				var either strings.Builder
				for i, allowed := range test.Allowed {
					fmt.Fprintf(&either, "'%s'", allowed)
					if i != len(test.Allowed)-1 {
						either.WriteString(" or ")
					}
				}
				t.Logf("expected %s but got '%s'",
					either.String(), joined)
				t.Fail()
			}
		})
	}

}
