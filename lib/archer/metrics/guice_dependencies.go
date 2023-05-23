package metrics

import (
	"fmt"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
	"github.com/samber/lo"

	"github.com/Faire/archer/lib/archer/languages"
	"github.com/Faire/archer/lib/archer/languages/kotlin_parser"
)

func ComputeKotlinGuiceDependencies(path string, file kotlin_parser.IKotlinFileContext) int {
	l := &treeListener{
		location: languages.NewLocationTracker(path),
	}

	antlr.NewParseTreeWalker().Walk(l, file)

	return l.constructorDependencies + l.fieldsDependencies
}

type treeListener struct {
	*kotlin_parser.BaseKotlinParserListener

	constructorDependencies int
	fieldsDependencies      int

	location             *languages.LocationTracker
	constructors         []int
	constructorArguments int
	hasInjectAnnotation  bool
}

func (l *treeListener) EnterClassDeclaration(ctx *kotlin_parser.ClassDeclarationContext) {
	var name string
	if ctx.SimpleIdentifier() == nil {
		name = "???"
	} else {
		ctx.SimpleIdentifier().GetText()
	}

	l.location.EnterClass(name)
}

func (l *treeListener) ExitClassDeclaration(_ *kotlin_parser.ClassDeclarationContext) {
	l.constructorDependencies += lo.Max(l.constructors)

	l.location.ExitClass()
}

func (l *treeListener) EnterPrimaryConstructor(_ *kotlin_parser.PrimaryConstructorContext) {
	l.location.EnterFunction("constructor")

	l.constructorArguments = 0
	l.hasInjectAnnotation = false
}

func (l *treeListener) ExitPrimaryConstructor(_ *kotlin_parser.PrimaryConstructorContext) {
	if l.hasInjectAnnotation {
		l.constructors = append(l.constructors, l.constructorArguments)
	}

	l.location.ExitFunction()
}

func (l *treeListener) EnterClassParameter(_ *kotlin_parser.ClassParameterContext) {
	l.constructorArguments++
}

func (l *treeListener) EnterSecondaryConstructor(_ *kotlin_parser.SecondaryConstructorContext) {
	l.location.EnterFunction("constructor")

	l.constructorArguments = 0
	l.hasInjectAnnotation = false
}

func (l *treeListener) ExitSecondaryConstructor(_ *kotlin_parser.SecondaryConstructorContext) {
	if l.hasInjectAnnotation {
		l.constructors = append(l.constructors, l.constructorArguments)
	}

	l.location.ExitFunction()
}

func (l *treeListener) EnterFunctionValueParameter(_ *kotlin_parser.FunctionValueParameterContext) {
	l.constructorArguments++
}

func (l *treeListener) EnterFunctionDeclaration(ctx *kotlin_parser.FunctionDeclarationContext) {
	name := ctx.SimpleIdentifier().GetText()

	l.location.EnterFunction(name)
}

func (l *treeListener) ExitFunctionDeclaration(_ *kotlin_parser.FunctionDeclarationContext) {
	l.location.ExitFunction()
}

func (l *treeListener) EnterPropertyDeclaration(ctx *kotlin_parser.PropertyDeclarationContext) {
	if l.location.IsInsideFunction() {
		return
	}

	if ctx.VariableDeclaration() == nil {
		panic(fmt.Sprintf("Only supported one variable per property declaration (in %v class %v line %v)",
			l.location.Path(), l.location.CurrentClassName(), ctx.GetStart().GetLine()))
	}

	name := ctx.VariableDeclaration().GetText()

	l.location.EnterField(name)

	l.hasInjectAnnotation = false
}

func (l *treeListener) ExitPropertyDeclaration(_ *kotlin_parser.PropertyDeclarationContext) {
	if l.location.IsInsideFunction() {
		return
	}

	if l.hasInjectAnnotation {
		l.fieldsDependencies++
	}

	l.location.ExitField()
}

func (l *treeListener) EnterPropertyDelegate(ctx *kotlin_parser.PropertyDelegateContext) {
	l.location.EnterFunction("by delegate")
}

func (l *treeListener) ExitPropertyDelegate(ctx *kotlin_parser.PropertyDelegateContext) {
	l.location.ExitFunction()
}

func (l *treeListener) EnterUnescapedAnnotation(ctx *kotlin_parser.UnescapedAnnotationContext) {
	text := ctx.GetText()

	if text == "Inject" {
		l.hasInjectAnnotation = true
	}
}
