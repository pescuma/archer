package consoles

import (
	"fmt"
	"strings"
	"time"
)

type stdoutConsole struct {
	prefixes []string
}

func NewStdOutConsole() Console {
	return &stdoutConsole{}
}

func (o *stdoutConsole) Printf(format string, a ...any) {
	print(o.Prepare(format, a...))
}

func (o *stdoutConsole) Prepare(format string, a ...any) string {
	builder := strings.Builder{}
	builder.WriteString("[")
	builder.WriteString(time.Now().Format("15:04:05"))
	builder.WriteString("] ")
	for _, prefix := range o.prefixes {
		builder.WriteString(prefix)
	}
	builder.WriteString(fmt.Sprintf(format, a...))
	return builder.String()
}

func (o *stdoutConsole) PushPrefix(format string, a ...any) {
	o.prefixes = append(o.prefixes, fmt.Sprintf(format, a...))
}

func (o *stdoutConsole) PopPrefix() {
	o.prefixes = o.prefixes[:len(o.prefixes)-1]
}
