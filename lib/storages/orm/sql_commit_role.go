package orm

import "strconv"

type sqlCommitRole int

const (
	CommitRoleAuthor    sqlCommitRole = iota
	CommitRoleCommitter sqlCommitRole = iota
)

func (r sqlCommitRole) String() string {
	return strconv.Itoa(int(r))
}
