package irc

import (
	"strings"
)

type Tags map[string]string

type Prefix struct {
	Name string
	User string
	Host string
}

type Message struct {
	Tags
	Prefix
	Command string
	Params  []string
}

func (m Message) String() string {
	var sb strings.Builder

	if len(m.Tags) != 0 {
		sb.WriteByte('@')
		i := 0
		for tag, value := range m.Tags {
			sb.WriteString(tag)
			if value != "" {
				sb.WriteByte('=')
			}
			for j := 0; j < len(value); j++ {
				c := value[j]
				switch c {
				case ';':
					sb.WriteString("\\:")
				case ' ':
					sb.WriteString("\\s")
				case '\\':
					sb.WriteString("\\\\")
				case '\r':
					sb.WriteString("\\r")
				case '\n':
					sb.WriteString("\\n")
				default:
					sb.WriteByte(c)
				}
			}
			if i != len(m.Tags)-1 {
				sb.WriteByte(';')
			}
			i++
		}
		sb.WriteByte(' ')
	}

	if m.Name != "" || m.User != "" || m.Host != "" {
		sb.WriteByte(':')
		sb.WriteString(m.Name)
		if m.User != "" {
			sb.WriteByte('!')
			sb.WriteString(m.User)
		}
		if m.Host != "" {
			sb.WriteByte('@')
			sb.WriteString(m.Host)
		}
		sb.WriteByte(' ')
	}

	sb.WriteString(m.Command)

	for i, param := range m.Params {
		sb.WriteByte(' ')
		if i == len(m.Params)-1 &&
			(strings.ContainsAny(param, " :") ||
				len(param) == 0) {
			sb.WriteByte(':')
		}
		sb.WriteString(param)
	}

	return sb.String()
}

func parseTags(p []byte) (Tags, []byte) {
	const (
		stKey = iota
		stValue
		stEscape
	)

	tags := Tags{}

	if p[0] != '@' {
		return tags, p
	}
	p = p[1:]

	var key, value strings.Builder
	st := stKey
	for _, b := range p {
		p = p[1:]
		switch b {
		case ' ', '\r', '\n':
			if key.Len() != 0 {
				tags[key.String()] = value.String()
			}
			return tags, p
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

	return tags, p
}

func parsePrefix(p []byte) (Prefix, []byte) {
	prefix := Prefix{}

	if p[0] != ':' {
		return prefix, p
	}
	p = p[1:]

	var name strings.Builder
nameloop:
	for _, b := range p {
		p = p[1:]
		switch b {
		case '!':
			prefix.Name = name.String()
			break nameloop
		case ' ', '\r', '\n':
			prefix.Name = name.String()
			return prefix, p
		default:
			name.WriteByte(b)
		}
	}

	var user strings.Builder
userloop:
	for _, b := range p {
		p = p[1:]
		switch b {
		case '@':
			prefix.User = user.String()
			break userloop
		case ' ', '\r', '\n':
			prefix.User = user.String()
			return prefix, p
		default:
			user.WriteByte(b)
		}
	}

	var host strings.Builder
	for _, b := range p {
		p = p[1:]
		switch b {
		case ' ', '\r', '\n':
			prefix.Host = host.String()
			return prefix, p
		default:
			host.WriteByte(b)
		}
	}

	return prefix, p
}

func parseCommand(p []byte) (string, []byte) {
	var command strings.Builder

	for _, b := range p {
		p = p[1:]
		switch b {
		case ' ', '\r', '\n':
			return command.String(), p
		default:
			if 'a' <= b && b <= 'z' {
				b -= 'a' - 'A'
			}
			command.WriteByte(b)
		}
	}

	return command.String(), p
}

func parseParams(p []byte) ([]string, []byte) {
	params := []string{}

	var param strings.Builder
	trailing := false
loop:
	for _, b := range p {
		p = p[1:]
		switch b {
		case ' ':
			if param.Len() != 0 {
				params = append(params, param.String())
				param.Reset()
			}
		case '\r', '\n':
			return append(params, param.String()), p
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
		for _, b := range p {
			p = p[1:]
			switch b {
			case '\r', '\n':
				return append(params, param.String()), p
			default:
				param.WriteByte(b)
			}
		}

		return append(params, param.String()), p
	}

	if param.Len() != 0 {
		params = append(params, param.String())
	}

	return params, p

}

func Parse(p []byte) Message {
	var message Message
	message.Tags, p = parseTags(p)
	message.Prefix, p = parsePrefix(p)
	message.Command, p = parseCommand(p)
	message.Params, p = parseParams(p)
	return message
}

func ParseString(s string) Message {
	return Parse([]byte(s))
}
