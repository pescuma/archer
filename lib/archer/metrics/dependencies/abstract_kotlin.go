package dependencies

import (
	"github.com/antlr/antlr4/runtime/Go/antlr/v4"

	"github.com/Faire/archer/lib/archer/languages/kotlin"
	"github.com/Faire/archer/lib/archer/languages/kotlin_parser"
	"github.com/Faire/archer/lib/archer/stucture"
)

func ComputeKotlinAbstracts(path string, structure *stucture.FileStructure, file kotlin_parser.IKotlinFileContext) int {
	l := &abstractsTreeListener{
		ASTListener: kotlin.NewASTListener(path),
	}

	antlr.NewParseTreeWalker().Walk(l, file)

	return l.abstractMethods + l.abstractProperties
}

type abstractsTreeListener struct {
	*kotlin.ASTListener

	abstractMethods    int
	abstractProperties int
}

func (l *abstractsTreeListener) EnterFunctionDeclaration(ctx *kotlin_parser.FunctionDeclarationContext) {
	l.ASTListener.EnterFunctionDeclaration(ctx)

	if isAbstract(ctx) {
		l.abstractMethods++
	}
}

func (l *abstractsTreeListener) EnterClassMemberDeclaration(ctx *kotlin_parser.ClassMemberDeclarationContext) {
	if ctx.Declaration() != nil {
		prop := ctx.Declaration().(*kotlin_parser.DeclarationContext).PropertyDeclaration()
		if prop != nil && isAbstract(prop) {
			l.abstractProperties++
		}
	}
}

func isAbstract(ctx antlr.Tree) bool {
	l := &abstractTreeListener{}
	antlr.NewParseTreeWalker().Walk(l, ctx)
	return l.hasAbstract
}

type abstractTreeListener struct {
	kotlin_parser.BaseKotlinParserListener

	hasAbstract bool
}

func (l *abstractTreeListener) EnterInheritanceModifier(ctx *kotlin_parser.InheritanceModifierContext) {
	if ctx.ABSTRACT() != nil {
		l.hasAbstract = true
	}
}
