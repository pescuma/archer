package kotlin

import (
	"fmt"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"

	"github.com/pescuma/archer/lib/languages"
	"github.com/pescuma/archer/lib/languages/kotlin_parser"
	"github.com/pescuma/archer/lib/utils"
)

type ASTListener struct {
	kotlin_parser.BaseKotlinParserListener

	Location *languages.LocationTracker

	inits        []int
	constructors []int
	lambdas      []int
	delegates    []int

	OnEnterClass    func(ctx antlr.Tree, pkg string, name string)
	OnExitClass     func(ctx antlr.Tree, pkg string, name string)
	OnEnterFunction func(ctx antlr.Tree, name string, params []string, result string)
	OnExitFunction  func(ctx antlr.Tree, name string, params []string, result string)
}

func NewASTListener(path string) *ASTListener {
	return &ASTListener{
		Location:     languages.NewLocationTracker(path),
		inits:        []int{0},
		constructors: []int{0},
		lambdas:      []int{0},
		delegates:    []int{0},
	}
}

func (l *ASTListener) EnterPackageHeader(ctx *kotlin_parser.PackageHeaderContext) {
	if ctx.Identifier() != nil {
		l.Location.EnterPackage(ctx.Identifier().GetText())
	}
}

func (l *ASTListener) EnterClassDeclaration(ctx *kotlin_parser.ClassDeclarationContext) {
	var name string
	if ctx.SimpleIdentifier() == nil {
		name = "???"
	} else {
		name = ctx.SimpleIdentifier().GetText()
	}

	l.Location.EnterClass(name)
	l.inits = append(l.inits, 0)
	l.constructors = append(l.constructors, 0)
	l.lambdas = append(l.lambdas, 0)
	l.delegates = append(l.delegates, 0)

	if l.OnEnterClass != nil {
		l.OnEnterClass(ctx, l.Location.CurrentPackageName(), l.Location.CurrentClassName())
	}
}

func (l *ASTListener) ExitClassDeclaration(ctx *kotlin_parser.ClassDeclarationContext) {
	if l.OnExitClass != nil {
		l.OnExitClass(ctx, l.Location.CurrentPackageName(), l.Location.CurrentClassName())
	}

	l.inits = utils.RemoveLast(l.inits)
	l.constructors = utils.RemoveLast(l.constructors)
	l.lambdas = utils.RemoveLast(l.lambdas)
	l.delegates = utils.RemoveLast(l.delegates)
	l.Location.ExitClass()
}

func (l *ASTListener) EnterAnonymousInitializer(ctx *kotlin_parser.AnonymousInitializerContext) {
	l.Location.EnterFunction(fmt.Sprintf("<init_%v>", utils.Last(l.inits)), []string{}, "")
	l.inits[len(l.inits)-1]++

	if l.OnEnterFunction != nil {
		l.OnEnterFunction(ctx, l.Location.CurrentFunctionName(), l.Location.CurrentFunctionParams(), l.Location.CurrentFunctionResult())
	}
}

func (l *ASTListener) ExitAnonymousInitializer(ctx *kotlin_parser.AnonymousInitializerContext) {
	if l.OnExitFunction != nil {
		l.OnExitFunction(ctx, l.Location.CurrentFunctionName(), l.Location.CurrentFunctionParams(), l.Location.CurrentFunctionResult())
	}

	l.Location.ExitFunction()
}

func (l *ASTListener) EnterPrimaryConstructor(ctx *kotlin_parser.PrimaryConstructorContext) {
	ps := ctx.ClassParameters().AllClassParameter()

	params := make([]string, 0, len(ps))
	for _, p := range ps {
		t := l.GetTypeName(p.Type_())
		params = append(params, t)
	}

	l.Location.EnterFunction(fmt.Sprintf("<constructor_%v>", utils.Last(l.constructors)), params, l.Location.CurrentClassName())
	l.constructors[len(l.constructors)-1]++
}

func (l *ASTListener) ExitPrimaryConstructor(_ *kotlin_parser.PrimaryConstructorContext) {
	l.Location.ExitFunction()
}

func (l *ASTListener) EnterSecondaryConstructor(ctx *kotlin_parser.SecondaryConstructorContext) {
	ps := ctx.FunctionValueParameters().AllFunctionValueParameter()

	params := make([]string, 0, len(ps))
	for _, p := range ps {
		t := l.GetTypeName(p.Parameter().Type_())
		params = append(params, t)
	}

	l.Location.EnterFunction(fmt.Sprintf("<constructor_%v>", utils.Last(l.constructors)), params, l.Location.CurrentClassName())
	l.constructors[len(l.constructors)-1]++
}

func (l *ASTListener) ExitSecondaryConstructor(_ *kotlin_parser.SecondaryConstructorContext) {
	l.Location.ExitFunction()
}

func (l *ASTListener) EnterFunctionDeclaration(ctx *kotlin_parser.FunctionDeclarationContext) {
	ps := ctx.FunctionValueParameters().AllFunctionValueParameter()

	params := make([]string, 0, len(ps))
	for _, p := range ps {
		t := l.GetTypeName(p.Parameter().Type_())
		params = append(params, t)
	}

	result := l.GetTypeName(ctx.Type_())

	l.Location.EnterFunction(ctx.SimpleIdentifier().GetText(), params, result)
}

func (l *ASTListener) ExitFunctionDeclaration(_ *kotlin_parser.FunctionDeclarationContext) {
	l.Location.ExitFunction()
}

func (l *ASTListener) EnterLambdaLiteral(ctx *kotlin_parser.LambdaLiteralContext) {
	var params []string

	if ctx.LambdaParameters() != nil {
		ps := ctx.LambdaParameters().AllLambdaParameter()
		for _, p := range ps {
			if p.VariableDeclaration() != nil {
				t := "?"
				vd := p.VariableDeclaration()
				if vd.Type_() != nil {
					t = l.GetTypeName(vd.Type_())
				}
				params = append(params, t)

			} else if p.MultiVariableDeclaration() != nil {
				vds := p.MultiVariableDeclaration().AllVariableDeclaration()
				for _, vd := range vds {
					t := "?"
					if vd.Type_() != nil {
						t = l.GetTypeName(vd.Type_())
					}
					params = append(params, t)
				}

			} else if p.Type_() != nil {
				t := l.GetTypeName(p.Type_())
				params = append(params, t)

			} else {
				panic(fmt.Sprintf("%v %v: %v: not implemented: %v %t",
					l.Location.Path(), ctx.GetSourceInterval().String(), ctx.GetText(), p, p))
			}
		}
	}

	l.Location.EnterFunction(fmt.Sprintf("<lambda_%v>", utils.Last(l.lambdas)), params, "")
	l.lambdas[len(l.lambdas)-1]++
}

func (l *ASTListener) ExitLambdaLiteral(_ *kotlin_parser.LambdaLiteralContext) {
	l.Location.ExitFunction()
}

func (l *ASTListener) EnterClassMemberDeclaration(ctx *kotlin_parser.ClassMemberDeclarationContext) {
	if ctx.Declaration() != nil {
		prop := ctx.Declaration().(*kotlin_parser.DeclarationContext).PropertyDeclaration()
		if prop != nil {
			prop := prop.(*kotlin_parser.PropertyDeclarationContext)
			l.Location.EnterField(prop.VariableDeclaration().GetText())
		}
	}
}

func (l *ASTListener) ExitClassMemberDeclaration(ctx *kotlin_parser.ClassMemberDeclarationContext) {
	if ctx.Declaration() != nil {
		prop := ctx.Declaration().(*kotlin_parser.DeclarationContext).PropertyDeclaration()
		if prop != nil {
			l.Location.ExitField()
		}
	}
}

func (l *ASTListener) EnterPropertyDelegate(_ *kotlin_parser.PropertyDelegateContext) {
	l.Location.EnterFunction(fmt.Sprintf("<by delegate_%v>", utils.Last(l.delegates)), []string{}, "")
	l.delegates[len(l.delegates)-1]++
}

func (l *ASTListener) ExitPropertyDelegate(_ *kotlin_parser.PropertyDelegateContext) {
	l.Location.ExitFunction()
}

func (l *ASTListener) GetTypeName(t kotlin_parser.ITypeContext) string {
	if t == nil {
		return "void"

	} else if t.FunctionType() != nil {
		ft := t.FunctionType().(*kotlin_parser.FunctionTypeContext)
		return ft.GetText()

	} else if t.ParenthesizedType() != nil {
		return l.GetTypeName(t.ParenthesizedType().Type_())

	} else if t.NullableType() != nil {
		tr := t.NullableType().(*kotlin_parser.NullableTypeContext)

		if tr.TypeReference() != nil {
			return tr.TypeReference().GetText()
		} else {
			return l.GetTypeName(tr.ParenthesizedType().Type_())
		}

	} else if t.TypeReference() != nil {
		return t.TypeReference().GetText()

	} else {
		panic(fmt.Sprintf("%v %v: %v: not implemented: %v %t",
			l.Location.Path(), t.GetSourceInterval().String(), t.GetText(), t, t))
	}
}
