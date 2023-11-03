package kotlin

import (
	"github.com/antlr/antlr4/runtime/Go/antlr/v4"

	"github.com/pescuma/archer/lib/archer/languages/kotlin_parser"
	"github.com/pescuma/archer/lib/archer/stucture"
)

func ImportStructure(path string, content kotlin_parser.IKotlinFileContext) *stucture.FileStructure {
	l := newStructureTreeListener(path)

	loadAllStructures(l, content)
	l.root.ResolveClasses()

	return l.root
}

func loadAllStructures(l *structureTreeListener, content kotlin_parser.IKotlinFileContext) {
	antlr.NewParseTreeWalker().Walk(l, content)
}

type structureTreeListener struct {
	*ASTListener

	root    *stucture.FileStructure
	current stucture.StructureElement
}

func newStructureTreeListener(path string) *structureTreeListener {
	s := &structureTreeListener{
		ASTListener: NewASTListener(path),
		root:        stucture.NewFileStructure(path),
	}

	s.current = s.root

	s.OnEnterClass = func(ctx antlr.Tree, pkg string, name string) {
		s.current = s.current.AddClass(pkg, name)
		s.root.AllStructures[ctx] = s.current
	}
	s.OnExitClass = func(ctx antlr.Tree, pkg string, name string) {
		s.current = s.current.GetParent()
	}

	s.OnEnterFunction = func(ctx antlr.Tree, name string, params []string, result string) {
		s.current = s.current.AddFunction(name, params, result)
		s.root.AllStructures[ctx] = s.current
	}
	s.OnExitFunction = func(ctx antlr.Tree, name string, params []string, result string) {
		s.current = s.current.GetParent()
	}

	return s
}

func (s *structureTreeListener) EnterEveryRule(ctx antlr.ParserRuleContext) {
	s.root.AllStructures[ctx] = s.current
}
