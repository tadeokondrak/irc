package irc

import (
	"bytes"
	"strings"
)

type Tags map[string]string

func (t Tags) Bytes() []byte {
	var buf bytes.Buffer

	i := 0
	for tag, value := range t {
		buf.WriteString(tag)
		if value != "" {
			buf.WriteByte('=')
		}
		for j := 0; j < len(value); j++ {
			c := value[j]
			switch c {
			case ';':
				buf.WriteString("\\:")
			case ' ':
				buf.WriteString("\\s")
			case '\\':
				buf.WriteString("\\\\")
			case '\r':
				buf.WriteString("\\r")
			case '\n':
				buf.WriteString("\\n")
			default:
				buf.WriteByte(c)
			}
		}
		if i != len(t)-1 {
			buf.WriteByte(';')
		}
		i++
	}

	return buf.Bytes()
}

func (t Tags) String() string {
	return string(t.Bytes())
}

type Prefix struct {
	Name string
	User string
	Host string
}

func (p Prefix) Bytes() []byte {
	var buf bytes.Buffer

	if p.Name != "" || p.User != "" || p.Host != "" {
		buf.WriteString(p.Name)
		if p.User != "" {
			buf.WriteByte('!')
			buf.WriteString(p.User)
		}
		if p.Host != "" {
			buf.WriteByte('@')
			buf.WriteString(p.Host)
		}
	}

	return buf.Bytes()
}

func (p Prefix) String() string {
	return string(p.Bytes())
}

type Message struct {
	Tags
	Prefix
	Command string
	Params  []string
}

func (m Message) Bytes() []byte {
	var buf bytes.Buffer

	if p := m.Tags.Bytes(); len(p) != 0 {
		buf.WriteByte('@')
		buf.Write(p)
		buf.WriteByte(' ')
	}

	if p := m.Prefix.Bytes(); len(p) != 0 {
		buf.WriteByte(':')
		buf.Write(p)
		buf.WriteByte(' ')
	}

	buf.WriteString(m.Command)

	for i, param := range m.Params {
		buf.WriteByte(' ')
		if i == len(m.Params)-1 &&
			(strings.ContainsAny(param, " :") ||
				len(param) == 0) {
			buf.WriteByte(':')
		}
		buf.WriteString(param)
	}

	buf.WriteString("\r\n")

	return buf.Bytes()
}

func (m Message) String() string {
	return string(m.Bytes())
}

func parseTags(p []byte) (Tags, int) {
	const (
		stKey = iota
		stValue
		stEscape
	)

	tags := Tags{}
	i := 0

	if len(p) == 0 || p[i] != '@' {
		return tags, i
	}
	i++

	var key, value strings.Builder
	st := stKey
	for _, b := range p[i:] {
		i++
		switch b {
		case ' ':
			if key.Len() != 0 {
				tags[key.String()] = value.String()
			}
			return tags, i
		case '\r', '\n':
			if key.Len() != 0 {
				tags[key.String()] = value.String()
			}
			return tags, i - 1
		case ';':
			if key.Len() != 0 {
				tags[key.String()] = value.String()
			}
			key.Reset()
			value.Reset()
			st = stKey
		case '=':
			st = stValue
		default:
			switch {
			case st == stKey:
				key.WriteByte(b)
			case st == stValue && b == '\\':
				st = stEscape
			case st == stValue:
				value.WriteByte(b)
			case st == stEscape && b == ':':
				value.WriteByte(';')
				st = stValue
			case st == stEscape && b == 's':
				value.WriteByte(' ')
				st = stValue
			case st == stEscape && b == '\\':
				value.WriteByte('\\')
				st = stValue
			case st == stEscape && b == 'r':
				value.WriteByte('\r')
				st = stValue
			case st == stEscape && b == 'n':
				value.WriteByte('\n')
				st = stValue
			case st == stEscape:
				value.WriteByte(b)
				st = stValue
			}
		}
	}

	if key.Len() != 0 {
		tags[key.String()] = value.String()
	}

	return tags, i
}

func parsePrefix(p []byte) (Prefix, int) {
	prefix := Prefix{}
	i := 0

	if len(p) == 0 || p[i] != ':' {
		return prefix, i
	}
	i++

	var name strings.Builder
nameloop:
	for _, b := range p[i:] {
		i++
		switch b {
		case '!':
			prefix.Name = name.String()
			i--
			break nameloop
		case '@':
			prefix.Name = name.String()
			i--
			break nameloop
		case ' ':
			prefix.Name = name.String()
			return prefix, i
		case '\r', '\n':
			prefix.Name = name.String()
			return prefix, i - 1
		default:
			name.WriteByte(b)
		}
	}
	prefix.Name = name.String()

	if i < len(p) && p[i] == '!' {
		i++
		var user strings.Builder
	userloop:
		for _, b := range p[i:] {
			i++
			switch b {
			case '@':
				i--
				prefix.User = user.String()
				break userloop
			case ' ':
				prefix.User = user.String()
				return prefix, i
			case '\r', '\n':
				prefix.User = user.String()
				return prefix, i - 1
			default:
				user.WriteByte(b)
			}
		}
		prefix.User = user.String()
	}

	if i < len(p) && p[i] == '@' {
		i++
		var host strings.Builder
		for _, b := range p[i:] {
			i++
			switch b {
			case ' ':
				prefix.Host = host.String()
				return prefix, i
			case '\r', '\n':
				prefix.Host = host.String()
				return prefix, i - 1
			default:
				host.WriteByte(b)
			}
		}
		prefix.Host = host.String()
	}

	return prefix, i
}

func parseCommand(p []byte) (string, int) {
	var command strings.Builder
	i := 0

	for _, b := range p[i:] {
		i++
		switch b {
		case ' ':
			return command.String(), i
		case '\r', '\n':
			return command.String(), i - 1
		default:
			if 'a' <= b && b <= 'z' {
				b -= 'a' - 'A'
			}
			command.WriteByte(b)
		}
	}

	return command.String(), i
}

func parseParams(p []byte) ([]string, int) {
	params := []string{}
	i := 0

	var param strings.Builder
	trailing := false
loop:
	for _, b := range p[i:] {
		i++
		switch b {
		case ' ':
			if param.Len() != 0 {
				params = append(params, param.String())
				param.Reset()
			}
		case '\r', '\n':
			if param.Len() != 0 {
				params = append(params, param.String())
				param.Reset()
			}
			return params, i - 1
		case ':':
			if param.Len() == 0 {
				trailing = true
				break loop
			}
		default:
			param.WriteByte(b)
		}
	}

	if trailing {
		for _, b := range p[i:] {
			i++
			switch b {
			case '\r', '\n':
				return append(params, param.String()), i - 1
			default:
				param.WriteByte(b)
			}
		}

		return append(params, param.String()), i
	}

	if param.Len() != 0 {
		params = append(params, param.String())
	}

	return params, i

}

// Parse parses an IRC message from p and returns it,
// along with how much of p it read.
func Parse(p []byte) (Message, int) {
	var message Message
	i, j := 0, 0
	message.Tags, j = parseTags(p[i:])
	i += j
	message.Prefix, j = parsePrefix(p[i:])
	i += j
	message.Command, j = parseCommand(p[i:])
	i += j
	message.Params, j = parseParams(p[i:])
	i += j
	if len(p)-i >= 2 && p[i+0] == '\r' && p[i+1] == '\n' {
		i += 2
	}
	return message, i
}

// ParseString converts s to a byte slice and calls Parse.
func ParseString(s string) Message {
	message, _ := Parse([]byte(s))
	return message
}
