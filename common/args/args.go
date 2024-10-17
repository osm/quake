package args

import (
	"strings"

	"github.com/osm/quake/common/buffer"
)

type Arg struct {
	Cmd  string
	Args []string
}

func Parse(input string) []Arg {
	var arg string
	var args []string
	var ret []Arg
	var inQuotes bool

	for _, c := range input {
		switch c {
		case '\n', ';':
			if inQuotes {
				arg += string(c)
				continue
			}

			if arg != "" {
				args = append(args, strings.TrimSpace(arg))
			}

			ret = append(ret, Arg{Cmd: strings.TrimSpace(args[0]), Args: args[1:]})
			arg = ""
			args = []string{}
		case ' ':
			if inQuotes {
				arg += string(c)
				continue
			}

			if arg != "" {
				args = append(args, strings.TrimSpace(arg))
				arg = ""
			}
		case '"':
			inQuotes = !inQuotes
			arg += string(c)
		default:
			arg += string(c)
		}
	}

	if arg != "" {
		args = append(args, strings.TrimSpace(arg))
	}

	if len(args) > 0 {
		ret = append(ret, Arg{Cmd: strings.TrimSpace(args[0]), Args: args[1:]})
	}

	return ret
}

func (s Arg) Bytes() []byte {
	buf := buffer.New()

	buf.PutBytes([]byte(s.Cmd))

	if len(s.Args) > 0 {
		buf.PutByte(byte(' '))
		buf.PutBytes([]byte(strings.Join(s.Args, " ")))
	}

	return buf.Bytes()
}

func (s Arg) String() string {
	return string(s.Bytes())
}
