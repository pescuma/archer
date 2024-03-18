package model

type IgnoreRules struct {
	maxID ID

	rules []*IgnoreRule
}

func NewIgnoreRules() *IgnoreRules {
	return &IgnoreRules{}
}

func (i *IgnoreRules) ListRules() []*IgnoreRule {
	return i.rules
}

func (i *IgnoreRules) AddRuleEx(rule *IgnoreRule) {
	if rule.ID > i.maxID {
		i.maxID = rule.ID
	}

	i.rules = append(i.rules, rule)
}

func (i *IgnoreRules) AddCommitRule(rule string) {
	i.maxID++
	i.rules = append(i.rules, &IgnoreRule{
		ID:   i.maxID,
		Type: CommitRule,
		Rule: rule,
	})
}

func (i *IgnoreRules) AddFileRule(rule string) {
	i.maxID++
	i.rules = append(i.rules, &IgnoreRule{
		ID:   i.maxID,
		Type: FileRule,
		Rule: rule,
	})
}
