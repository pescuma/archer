package consoles

type Console interface {
	Printf(format string, a ...any)
	Prepare(format string, a ...any) string

	PushPrefix(format string, a ...any)
	PopPrefix()
}
