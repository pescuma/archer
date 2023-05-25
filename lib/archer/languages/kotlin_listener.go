package languages

import (
	"github.com/Faire/archer/lib/archer/languages/kotlin_parser"
)

type KotlinASTListener struct {
	kotlin_parser.BaseKotlinParserListener

	Location *LocationTracker
}

func NewKotlinASTListener(path string) *KotlinASTListener {
	return &KotlinASTListener{
		Location: NewLocationTracker(path),
	}
}

func (l *KotlinASTListener) EnterClassDeclaration(ctx *kotlin_parser.ClassDeclarationContext) {
	var name string
	if ctx.SimpleIdentifier() == nil {
		name = "???"
	} else {
		name = ctx.SimpleIdentifier().GetText()
	}

	l.Location.EnterClass(name)
}

func (l *KotlinASTListener) ExitClassDeclaration(_ *kotlin_parser.ClassDeclarationContext) {
	l.Location.ExitClass()
}

func (l *KotlinASTListener) EnterAnonymousInitializer(ctx *kotlin_parser.AnonymousInitializerContext) {
	l.Location.EnterFunction("<init>", 0)
}

func (l *KotlinASTListener) ExitAnonymousInitializer(ctx *kotlin_parser.AnonymousInitializerContext) {
	l.Location.ExitFunction()
}

func (l *KotlinASTListener) EnterPrimaryConstructor(ctx *kotlin_parser.PrimaryConstructorContext) {
	arity := len(ctx.ClassParameters().(*kotlin_parser.ClassParametersContext).AllClassParameter())
	l.Location.EnterFunction("<constructor>", arity)
}

func (l *KotlinASTListener) ExitPrimaryConstructor(_ *kotlin_parser.PrimaryConstructorContext) {
	l.Location.ExitFunction()
}

func (l *KotlinASTListener) EnterSecondaryConstructor(ctx *kotlin_parser.SecondaryConstructorContext) {
	arity := len(ctx.FunctionValueParameters().(*kotlin_parser.FunctionValueParametersContext).AllFunctionValueParameter())
	l.Location.EnterFunction("<constructor>", arity)
}

func (l *KotlinASTListener) ExitSecondaryConstructor(_ *kotlin_parser.SecondaryConstructorContext) {
	l.Location.ExitFunction()
}

func (l *KotlinASTListener) EnterFunctionDeclaration(ctx *kotlin_parser.FunctionDeclarationContext) {
	arity := len(ctx.FunctionValueParameters().(*kotlin_parser.FunctionValueParametersContext).AllFunctionValueParameter())
	l.Location.EnterFunction(ctx.SimpleIdentifier().GetText(), arity)
}

func (l *KotlinASTListener) ExitFunctionDeclaration(ctx *kotlin_parser.FunctionDeclarationContext) {
	l.Location.ExitFunction()
}

func (l *KotlinASTListener) EnterLambdaLiteral(ctx *kotlin_parser.LambdaLiteralContext) {
	arity := 1
	if ctx.LambdaParameters() != nil {
		arity = len(ctx.LambdaParameters().(*kotlin_parser.LambdaParametersContext).AllLambdaParameter())
	}
	l.Location.EnterFunction("<lambda>", arity)
}

func (l *KotlinASTListener) ExitLambdaLiteral(ctx *kotlin_parser.LambdaLiteralContext) {
	l.Location.ExitFunction()
}

func (l *KotlinASTListener) EnterClassMemberDeclaration(ctx *kotlin_parser.ClassMemberDeclarationContext) {
	if ctx.Declaration() != nil {
		prop := ctx.Declaration().(*kotlin_parser.DeclarationContext).PropertyDeclaration()
		if prop != nil {
			prop := prop.(*kotlin_parser.PropertyDeclarationContext)
			l.Location.EnterField(prop.VariableDeclaration().GetText())
		}
	}
}

func (l *KotlinASTListener) ExitClassMemberDeclaration(ctx *kotlin_parser.ClassMemberDeclarationContext) {
	if ctx.Declaration() != nil {
		prop := ctx.Declaration().(*kotlin_parser.DeclarationContext).PropertyDeclaration()
		if prop != nil {
			l.Location.ExitField()
		}
	}
}

func (l *KotlinASTListener) EnterPropertyDelegate(ctx *kotlin_parser.PropertyDelegateContext) {
	l.Location.EnterFunction("by delegate", 0)
}

func (l *KotlinASTListener) ExitPropertyDelegate(ctx *kotlin_parser.PropertyDelegateContext) {
	l.Location.ExitFunction()
}
