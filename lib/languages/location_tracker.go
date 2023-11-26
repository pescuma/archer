package languages

import (
	"strings"

	"github.com/pescuma/archer/lib/utils"
)

type LocationTracker struct {
	path      string
	pkg       string
	names     []string
	className []string
	function  []functionInfo
	fieldName string
}

type functionInfo struct {
	name   string
	params []string
	result string
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

func (l *LocationTracker) CurrentPackageName() string {
	return l.pkg
}

func (l *LocationTracker) IsInsideClass() bool {
	return len(l.className) > 1
}

func (l *LocationTracker) CurrentClassName() string {
	return strings.Join(l.names, ".")
}

func (l *LocationTracker) IsInsideFunction() bool {
	return utils.Last(l.function).name != ""
}

func (l *LocationTracker) CurrentFunctionName() string {
	return utils.Last(l.function).name
}

func (l *LocationTracker) CurrentFunctionParams() []string {
	return utils.Last(l.function).params
}

func (l *LocationTracker) CurrentFunctionResult() string {
	return utils.Last(l.function).result
}

func (l *LocationTracker) CurrentFunctionArity() int {
	return len(utils.Last(l.function).params)
}

func (l *LocationTracker) IsInsideField() bool {
	return l.fieldName != ""
}

func (l *LocationTracker) CurrentFieldName() string {
	return l.fieldName
}

func (l *LocationTracker) EnterPackage(name string) {
	l.pkg = name
}

func (l *LocationTracker) EnterClass(name string) {
	if len(l.className) > 1 {
		name = utils.Last(l.className) + "." + name
	}

	l.className = append(l.className, name)
	l.function = append(l.function, functionInfo{})
	l.names = append(l.names, name)
}

func (l *LocationTracker) ExitClass() {
	l.names = utils.RemoveLast(l.names)
	l.function = utils.RemoveLast(l.function)
	l.className = utils.RemoveLast(l.className)
}

func (l *LocationTracker) EnterFunction(name string, params []string, result string) {
	l.function = append(l.function, functionInfo{name, params, result})
	l.names = append(l.names, name)
}

func (l *LocationTracker) ExitFunction() {
	l.names = utils.RemoveLast(l.names)
	l.function = utils.RemoveLast(l.function)
}

func (l *LocationTracker) EnterField(name string) {
	l.fieldName = name
}

func (l *LocationTracker) ExitField() {
	l.fieldName = ""
}
