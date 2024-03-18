package model

import "time"

type IgnoreRule struct {
	ID   ID
	Type IgnoreRuleType
	Rule string

	Deleted   bool
	DeletedAt time.Time
}

type IgnoreRuleType int

const (
	UnknownRule IgnoreRuleType = iota
	CommitRule
	FileRule
)
