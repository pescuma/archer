package complexity

type CyclomaticComplexity struct {
	complexity int
}

func NewCyclomaticComplexity() *CyclomaticComplexity {
	return &CyclomaticComplexity{
		complexity: 0,
	}
}

func (c *CyclomaticComplexity) Compute() int {
	return c.complexity
}

func (c *CyclomaticComplexity) OnEnterFunction() {
	c.complexity++
}

func (c *CyclomaticComplexity) OnLogicalOperators(operators int) {
	c.complexity += operators

}

func (c *CyclomaticComplexity) OnJump() {
	c.complexity++
}

func (c *CyclomaticComplexity) OnConditional() {
	c.complexity++
}

func (c *CyclomaticComplexity) OnLoop() {
	c.complexity++
}
