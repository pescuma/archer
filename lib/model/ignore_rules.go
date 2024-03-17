package model

type IgnoreRules struct {
	CommitRules []string
	FileRules   []string
}

func (i *IgnoreRules) AddCommitRule(rule string) {
	i.CommitRules = append(i.CommitRules, rule)
}

func (i *IgnoreRules) AddFileRule(rule string) {
	i.FileRules = append(i.FileRules, rule)
}
