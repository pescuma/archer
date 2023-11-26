package consoles

type Console interface {
	Printf(format string, a ...any)

	PushPrefix(format string, a ...any)
	PopPrefix()
}
