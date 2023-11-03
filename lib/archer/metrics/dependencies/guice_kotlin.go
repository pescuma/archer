package dependencies

import (
	"strings"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
	"github.com/samber/lo"

	"github.com/pescuma/archer/lib/archer/languages/kotlin"
	"github.com/pescuma/archer/lib/archer/languages/kotlin_parser"
	"github.com/pescuma/archer/lib/archer/stucture"
	"github.com/pescuma/archer/lib/archer/utils"
)

func ComputeKotlinGuiceDependencies(path string, structure *stucture.FileStructure, file kotlin_parser.IKotlinFileContext) int {
	l := &guiceTreeListener{
		ASTListener: kotlin.NewASTListener(path),
	}

	antlr.NewParseTreeWalker().Walk(l, file)

	return l.constructorDependencies + l.fieldsDependencies
}

type guiceTreeListener struct {
	*kotlin.ASTListener

	constructorDependencies int
	fieldsDependencies      int

	classes             []*classData
	hasInjectAnnotation bool
}

type classData struct {
	constructors         []int
	constructorArguments int
}

func (l *guiceTreeListener) EnterClassDeclaration(ctx *kotlin_parser.ClassDeclarationContext) {
	l.ASTListener.EnterClassDeclaration(ctx)

	l.classes = append(l.classes, &classData{})
}

func (l *guiceTreeListener) ExitClassDeclaration(ctx *kotlin_parser.ClassDeclarationContext) {
	cd := utils.Last(l.classes)
	l.constructorDependencies += lo.Max(cd.constructors)
	l.classes = utils.RemoveLast(l.classes)

	l.ASTListener.ExitClassDeclaration(ctx)
}

func (l *guiceTreeListener) EnterPrimaryConstructor(ctx *kotlin_parser.PrimaryConstructorContext) {
	l.ASTListener.EnterPrimaryConstructor(ctx)

	cd := utils.Last(l.classes)
	cd.constructorArguments = 0
	l.hasInjectAnnotation = false
}

func (l *guiceTreeListener) ExitPrimaryConstructor(ctx *kotlin_parser.PrimaryConstructorContext) {
	if l.hasInjectAnnotation {
		cd := utils.Last(l.classes)
		cd.constructors = append(cd.constructors, cd.constructorArguments)
	}

	l.ASTListener.ExitPrimaryConstructor(ctx)
}

func (l *guiceTreeListener) EnterClassParameter(ctx *kotlin_parser.ClassParameterContext) {
	l.ASTListener.EnterClassParameter(ctx)

	cd := utils.Last(l.classes)
	cd.constructorArguments++
}

func (l *guiceTreeListener) EnterSecondaryConstructor(ctx *kotlin_parser.SecondaryConstructorContext) {
	l.ASTListener.EnterSecondaryConstructor(ctx)

	cd := utils.Last(l.classes)
	cd.constructorArguments = 0
	l.hasInjectAnnotation = false
}

func (l *guiceTreeListener) ExitSecondaryConstructor(ctx *kotlin_parser.SecondaryConstructorContext) {
	if l.hasInjectAnnotation {
		cd := utils.Last(l.classes)
		cd.constructors = append(cd.constructors, cd.constructorArguments)
	}

	l.ASTListener.ExitSecondaryConstructor(ctx)
}

func (l *guiceTreeListener) EnterFunctionValueParameter(ctx *kotlin_parser.FunctionValueParameterContext) {
	l.ASTListener.EnterFunctionValueParameter(ctx)

	if strings.HasPrefix(l.Location.CurrentFunctionName(), "<constructor:") {
		cd := utils.Last(l.classes)
		cd.constructorArguments++
	}
}

func (l *guiceTreeListener) EnterClassMemberDeclaration(ctx *kotlin_parser.ClassMemberDeclarationContext) {
	l.ASTListener.EnterClassMemberDeclaration(ctx)

	if ctx.Declaration() != nil {
		prop := ctx.Declaration().(*kotlin_parser.DeclarationContext).PropertyDeclaration()
		if prop != nil {
			l.hasInjectAnnotation = false
		}
	}
}

func (l *guiceTreeListener) ExitClassMemberDeclaration(ctx *kotlin_parser.ClassMemberDeclarationContext) {
	if ctx.Declaration() != nil {
		prop := ctx.Declaration().(*kotlin_parser.DeclarationContext).PropertyDeclaration()
		if prop != nil {
			if l.hasInjectAnnotation {
				l.fieldsDependencies++
			}
		}
	}

	l.ASTListener.ExitClassMemberDeclaration(ctx)
}

func (l *guiceTreeListener) EnterUnescapedAnnotation(ctx *kotlin_parser.UnescapedAnnotationContext) {
	l.ASTListener.EnterUnescapedAnnotation(ctx)

	text := ctx.GetText()

	if text == "Inject" {
		l.hasInjectAnnotation = true
	}
}
