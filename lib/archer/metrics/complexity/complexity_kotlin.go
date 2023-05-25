package complexity

import (
	"github.com/antlr/antlr4/runtime/Go/antlr/v4"

	"github.com/Faire/archer/lib/archer/languages"
	"github.com/Faire/archer/lib/archer/languages/kotlin_parser"
	"github.com/Faire/archer/lib/archer/utils"
)

type Result struct {
	CyclomaticComplexity int
	CognitiveComplexity  int
}

func ComputeKotlinComplexity(path string, file kotlin_parser.IKotlinFileContext) Result {
	l := &complexityTreeListener{
		KotlinASTListener: languages.NewKotlinASTListener(path),
		cognitive:         NewCognitiveComplexity(),
		cyclomatic:        NewCyclomaticComplexity(),
	}

	antlr.NewParseTreeWalker().Walk(l, file)

	return Result{l.cyclomatic.Compute(), l.cognitive.Compute()}
}

type complexityTreeListener struct {
	*languages.KotlinASTListener

	cognitive  *CognitiveComplexity
	cyclomatic *CyclomaticComplexity
}

func (l *complexityTreeListener) EnterAnonymousInitializer(ctx *kotlin_parser.AnonymousInitializerContext) {
	l.KotlinASTListener.EnterAnonymousInitializer(ctx)

	l.cognitive.OnEnterFunction()
}

func (l *complexityTreeListener) ExitAnonymousInitializer(ctx *kotlin_parser.AnonymousInitializerContext) {
	l.cognitive.OnExitFunction()

	l.KotlinASTListener.ExitAnonymousInitializer(ctx)
}

func (l *complexityTreeListener) EnterSecondaryConstructor(ctx *kotlin_parser.SecondaryConstructorContext) {
	l.KotlinASTListener.EnterSecondaryConstructor(ctx)

	l.cognitive.OnEnterFunction()
}

func (l *complexityTreeListener) ExitSecondaryConstructor(ctx *kotlin_parser.SecondaryConstructorContext) {
	l.cognitive.OnExitFunction()

	l.KotlinASTListener.ExitSecondaryConstructor(ctx)
}

func (l *complexityTreeListener) EnterFunctionDeclaration(ctx *kotlin_parser.FunctionDeclarationContext) {
	l.KotlinASTListener.EnterFunctionDeclaration(ctx)

	l.cyclomatic.OnEnterFunction()
	l.cognitive.OnEnterFunction()
}

func (l *complexityTreeListener) ExitFunctionDeclaration(ctx *kotlin_parser.FunctionDeclarationContext) {
	l.cognitive.OnExitFunction()

	l.KotlinASTListener.ExitFunctionDeclaration(ctx)
}

func (l *complexityTreeListener) EnterLambdaLiteral(ctx *kotlin_parser.LambdaLiteralContext) {
	l.KotlinASTListener.EnterLambdaLiteral(ctx)

	l.cognitive.OnEnterFunction()
}

func (l *complexityTreeListener) ExitLambdaLiteral(ctx *kotlin_parser.LambdaLiteralContext) {
	l.cognitive.OnExitFunction()

	l.KotlinASTListener.ExitLambdaLiteral(ctx)
}

func (l *complexityTreeListener) EnterLoopStatement(ctx *kotlin_parser.LoopStatementContext) {
	l.KotlinASTListener.EnterLoopStatement(ctx)

	l.cyclomatic.OnLoop()
	l.cognitive.OnEnterLoop()
}

func (l *complexityTreeListener) ExitLoopStatement(ctx *kotlin_parser.LoopStatementContext) {
	l.cognitive.OnExitLoop()

	l.KotlinASTListener.ExitLoopStatement(ctx)
}

func (l *complexityTreeListener) EnterWhenExpression(ctx *kotlin_parser.WhenExpressionContext) {
	l.KotlinASTListener.EnterWhenExpression(ctx)

	l.cognitive.OnEnterSwitch()
}

func (l *complexityTreeListener) ExitWhenExpression(ctx *kotlin_parser.WhenExpressionContext) {
	l.cognitive.OnExitSwitch()

	l.KotlinASTListener.ExitWhenExpression(ctx)
}

func (l *complexityTreeListener) EnterWhenEntry(ctx *kotlin_parser.WhenEntryContext) {
	l.KotlinASTListener.EnterWhenEntry(ctx)

	l.cyclomatic.OnConditional()
}

func (l *complexityTreeListener) EnterCatchBlock(ctx *kotlin_parser.CatchBlockContext) {
	l.KotlinASTListener.EnterCatchBlock(ctx)

	l.cyclomatic.OnConditional()
	l.cognitive.OnEnterCatch()
}

func (l *complexityTreeListener) ExitCatchBlock(ctx *kotlin_parser.CatchBlockContext) {
	l.cognitive.OnExitCatch()

	l.KotlinASTListener.ExitCatchBlock(ctx)
}

func (l *complexityTreeListener) EnterIfExpression(ctx *kotlin_parser.IfExpressionContext) {
	l.KotlinASTListener.EnterIfExpression(ctx)

	l.cyclomatic.OnConditional()
	l.cognitive.OnEnterConditional(!isElseIf(ctx))
}

func isElseIf(ctx *kotlin_parser.IfExpressionContext) bool {
	var cur antlr.Tree
	cur = ctx
	parent := ctx.GetParent()
	for parent != nil {
		_, ok := parent.(*kotlin_parser.StatementsContext)
		if ok {
			return false
		}

		parentIf, ok := parent.(*kotlin_parser.IfExpressionContext)
		if ok {
			parentElse := parentIf.ControlStructureBody(1)
			return parentElse == cur
		}

		cur = parent
		parent = cur.GetParent()
	}

	return false
}

func (l *complexityTreeListener) ExitIfExpression(ctx *kotlin_parser.IfExpressionContext) {
	l.cognitive.OnExitConditional()

	l.KotlinASTListener.ExitIfExpression(ctx)
}

func (l *complexityTreeListener) EnterJumpExpression(ctx *kotlin_parser.JumpExpressionContext) {
	l.KotlinASTListener.EnterJumpExpression(ctx)

	if ctx.CONTINUE() != nil || ctx.CONTINUE_AT() != nil || ctx.BREAK() != nil || ctx.BREAK_AT() != nil {
		l.cyclomatic.OnJump()
	}

	if ctx.BREAK_AT() != nil || ctx.CONTINUE_AT() != nil || ctx.RETURN_AT() != nil {
		l.cognitive.OnJumpToLabel()
	}
}

var nestingFunctions = map[string]bool{
	"run":       true,
	"let":       true,
	"apply":     true,
	"with":      true,
	"also":      true,
	"use":       true,
	"forEach":   true,
	"isNotNull": true,
	"ifNull":    true,
}

func (l *complexityTreeListener) EnterCallSuffix(ctx *kotlin_parser.CallSuffixContext) {
	l.KotlinASTListener.EnterCallSuffix(ctx)

	expr := getParentExpression[*kotlin_parser.PostfixUnaryExpressionContext](ctx)

	ps := expr.AllPostfixUnarySuffix()
	if len(ps) > 0 {
		f := utils.Last(ps).GetText()
		if nestingFunctions[f] {
			l.cyclomatic.OnEnterFunction()
		}
	}

	// Only direct recursive function calls for now
	if len(ps) == 1 {
		name := expr.PrimaryExpression().GetText()
		arity := 0
		if ctx.ValueArguments() != nil {
			arity = len(ctx.ValueArguments().(*kotlin_parser.ValueArgumentsContext).AllValueArgument())
		}

		if l.Location.CurrentFunctionName() == name && l.Location.CurrentFunctionArity() == arity {
			l.cognitive.OnRecursiveCall()
		}
	}

}

func getParentExpression[T any](ctx *kotlin_parser.CallSuffixContext) T {
	var result T

	var cur antlr.Tree
	cur = ctx
	parent := ctx.GetParent()
	for parent != nil {
		_, ok := parent.(*kotlin_parser.StatementsContext)
		if ok {
			return result
		}

		result, ok := parent.(T)
		if ok {
			return result
		}

		cur = parent
		parent = cur.GetParent()
	}

	return result
}

func (l *complexityTreeListener) EnterDisjunction(ctx *kotlin_parser.DisjunctionContext) {
	l.KotlinASTListener.EnterDisjunction(ctx)

	operators := len(ctx.AllConjunction()) - 1
	if operators > 0 {
		l.cyclomatic.OnLogicalOperators(operators)
		l.cognitive.OnSequenceOfLogicalOperators()
	}
}

func (l *complexityTreeListener) EnterConjunction(ctx *kotlin_parser.ConjunctionContext) {
	l.KotlinASTListener.EnterConjunction(ctx)

	operators := len(ctx.AllEquality()) - 1
	if operators > 0 {
		l.cyclomatic.OnLogicalOperators(operators)
		l.cognitive.OnSequenceOfLogicalOperators()
	}
}

func (l *complexityTreeListener) EnterElvis(ctx *kotlin_parser.ElvisContext) {
	l.KotlinASTListener.EnterElvis(ctx)

	l.cyclomatic.OnLogicalOperators(1)
}