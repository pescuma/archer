package languages

import (
	"github.com/Faire/archer/lib/archer/utils"
)

type LocationTracker struct {
	path         string
	className    []string
	functionName []string
	fieldName    string
}

func NewLocationTracker(path string) *LocationTracker {
	return &LocationTracker{
		path:         path,
		className:    []string{""},
		functionName: []string{""},
	}
}

func (l *LocationTracker) Path() string {
	return l.path
}

func (l *LocationTracker) IsInsideClass() bool {
	return l.CurrentClassName() != ""
}

func (l *LocationTracker) CurrentClassName() string {
	return utils.Last(l.className)
}

func (l *LocationTracker) IsInsideFunction() bool {
	return l.CurrentFunctionName() != ""
}

func (l *LocationTracker) CurrentFunctionName() string {
	return utils.Last(l.functionName)
}

func (l *LocationTracker) IsInsideField() bool {
	return l.fieldName != ""
}

func (l *LocationTracker) CurrentFieldName() string {
	return l.fieldName
}

func (l *LocationTracker) EnterClass(name string) {
	if len(l.className) > 1 {
		name = utils.Last(l.className) + "." + name
	}

	l.className = append(l.className, name)
	l.functionName = append(l.functionName, "")
}

func (l *LocationTracker) ExitClass() {
	l.functionName = utils.RemoveLast(l.functionName)
	l.className = utils.RemoveLast(l.className)
}

func (l *LocationTracker) EnterFunction(name string) {
	l.functionName = append(l.functionName, name)
}

func (l *LocationTracker) ExitFunction() {
	l.functionName = utils.RemoveLast(l.functionName)
}

func (l *LocationTracker) EnterField(name string) {
	l.fieldName = name
}

func (l *LocationTracker) ExitField() {
	l.fieldName = ""
}
