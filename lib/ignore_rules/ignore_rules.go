package ignore_rules

import (
	"sync"

	"github.com/pescuma/archer/lib/consoles"
	"github.com/pescuma/archer/lib/filters"
	"github.com/pescuma/archer/lib/model"
	"github.com/pescuma/archer/lib/storages"
)

type IgnoreRules struct {
	mutex sync.RWMutex

	console consoles.Console
	storage storages.Storage

	rules        *model.IgnoreRules
	fileFilter   filters.FileFilter
	commitFilter filters.CommitFilter
}

func New(console consoles.Console, storage storages.Storage) (*IgnoreRules, error) {
	rules, err := storage.LoadIgnoreRules()
	if err != nil {
		return nil, err
	}

	result := &IgnoreRules{
		console: console,
		storage: storage,
		rules:   rules,
	}

	err = result.parseRules()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (i *IgnoreRules) AddFileRule(rule string) error {
	_, err := filters.ParseFileFilter(rule)
	if err != nil {
		return err
	}

	changed, err := i.addFileRule(rule)
	if err != nil {
		return err
	}

	if !changed {
		i.console.Printf("Ignoring duplicated rule: %v\n", rule)
		return nil
	}

	files, err := i.storage.LoadFiles()
	if err != nil {
		return err
	}

	i.console.Printf("Updating files with new ignore information...\n")

	for _, file := range files.List() {
		file.Ignore = i.IgnoreFile(file)
	}

	return nil
}

func (i *IgnoreRules) addFileRule(rule string) (bool, error) {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	for _, r := range i.rules.ListRules() {
		if r.Type == model.FileRule && r.Rule == rule {
			return false, nil
		}
	}

	i.rules.AddFileRule(rule)

	err := i.parseRules()
	return true, err
}

func (i *IgnoreRules) AddCommitRule(rule string) error {
	_, err := filters.ParseCommitFilter(rule)
	if err != nil {
		return err
	}

	changed, err := i.addCommitRule(rule)
	if err != nil {
		return err
	}

	if !changed {
		i.console.Printf("Ignoring duplicated rule: %v\n", rule)
		return nil
	}

	repos, err := i.storage.LoadRepositories()
	if err != nil {
		return err
	}

	i.console.Printf("Updating commits with new ignore information...\n")

	for _, repo := range repos.List() {
		for _, commit := range repo.ListCommits() {
			commit.Ignore = i.IgnoreCommit(repo, commit)
		}
	}

	return nil
}

func (i *IgnoreRules) addCommitRule(rule string) (bool, error) {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	for _, r := range i.rules.ListRules() {
		if r.Type == model.CommitRule && r.Rule == rule {
			return false, nil
		}
	}

	i.rules.AddCommitRule(rule)

	err := i.parseRules()
	return true, err
}

func (i *IgnoreRules) IgnoreFile(file *model.File) bool {
	i.mutex.RLock()
	defer i.mutex.RUnlock()

	return !i.fileFilter(file)
}

func (i *IgnoreRules) IgnoreCommit(repo *model.Repository, commit *model.RepositoryCommit) bool {
	i.mutex.RLock()
	defer i.mutex.RUnlock()

	return !i.commitFilter(repo, commit)
}

func (i *IgnoreRules) parseRules() error {
	cs := make([]filters.CommitFilterWithUsage, 0, 10)
	fs := make([]filters.FileFilterWithUsage, 0, 10)

	for _, r := range i.rules.ListRules() {
		if r.Deleted {
			continue
		}

		//goland:noinspection GoSwitchMissingCasesForIotaConsts
		switch r.Type {
		case model.FileRule:
			f, err := filters.ParseFileFilterWithUsage(r.Rule, filters.Exclude)
			if err != nil {
				return err
			}

			fs = append(fs, f)

		case model.CommitRule:
			f, err := filters.ParseCommitFilterWithUsage(r.Rule, filters.Exclude)
			if err != nil {
				return err
			}

			cs = append(cs, f)
		}
	}

	i.fileFilter = filters.UnliftFileFilter(filters.GroupFileFilters(fs...))
	i.commitFilter = filters.UnliftCommitFilter(filters.GroupCommitFilters(cs...))

	return nil
}
