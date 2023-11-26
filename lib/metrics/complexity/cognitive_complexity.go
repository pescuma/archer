package complexity

import (
	"github.com/pescuma/archer/lib/utils"
)

// https://www.sonarsource.com/docs/CognitiveComplexity.pdf

type CognitiveComplexity struct {
	complexity int
	nesting    int
}

func NewCognitiveComplexity() *CognitiveComplexity {
	return &CognitiveComplexity{
		complexity: 0,
		nesting:    -1, // for the initial function
	}
}

func (c *CognitiveComplexity) Compute() int {
	return c.complexity
}

func (c *CognitiveComplexity) addNestedComplexity() {
	c.complexity += 1 + utils.Max(c.nesting, 0)
}

func (c *CognitiveComplexity) addSimpleComplexity() {
	c.complexity++
}

func (c *CognitiveComplexity) OnEnterFunction() {
	c.nesting++
}

func (c *CognitiveComplexity) OnExitFunction() {
	c.nesting--
}

func (c *CognitiveComplexity) OnEnterLoop() {
	c.addNestedComplexity()
	c.nesting++
}

func (c *CognitiveComplexity) OnExitLoop() {
	c.nesting--
}

func (c *CognitiveComplexity) OnEnterConditional(first bool) {
	if first {
		c.addNestedComplexity()
	} else {
		c.addSimpleComplexity()
	}
	c.nesting++
}

func (c *CognitiveComplexity) OnExitConditional() {
	c.nesting--
}

func (c *CognitiveComplexity) OnEnterCatch() {
	c.addNestedComplexity()
	c.nesting++
}

func (c *CognitiveComplexity) OnExitCatch() {
	c.nesting--
}

func (c *CognitiveComplexity) OnEnterSwitch() {
	c.addNestedComplexity()
	c.nesting++
}

func (c *CognitiveComplexity) OnExitSwitch() {
	c.nesting--
}

func (c *CognitiveComplexity) OnSequenceOfLogicalOperators() {
	c.addSimpleComplexity()
}

func (c *CognitiveComplexity) OnRecursiveCall() {
	c.addSimpleComplexity()
}

func (c *CognitiveComplexity) OnJumpToLabel() {
	c.addSimpleComplexity()
}
