package languages

import (
	"github.com/Faire/archer/lib/archer/utils"
)

type LocationTracker struct {
	path      string
	className []string
	function  []functionInfo
	fieldName string
}

type functionInfo struct {
	name  string
	arity int
}

func NewLocationTracker(path string) *LocationTracker {
	return &LocationTracker{
		path:      path,
		className: []string{""},
		function:  []functionInfo{{}},
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
	return utils.Last(l.function).name
}

func (l *LocationTracker) CurrentFunctionArity() int {
	return utils.Last(l.function).arity
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
	l.function = append(l.function, functionInfo{})
}

func (l *LocationTracker) ExitClass() {
	l.function = utils.RemoveLast(l.function)
	l.className = utils.RemoveLast(l.className)
}

func (l *LocationTracker) EnterFunction(name string, arity int) {
	l.function = append(l.function, functionInfo{name, arity})
}

func (l *LocationTracker) ExitFunction() {
	l.function = utils.RemoveLast(l.function)
}

func (l *LocationTracker) EnterField(name string) {
	l.fieldName = name
}

func (l *LocationTracker) ExitField() {
	l.fieldName = ""
}
