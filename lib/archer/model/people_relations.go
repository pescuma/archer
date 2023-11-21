package model

type PeopleRelations struct {
	personFile map[UUID]map[UUID]*PersonFile
	filePerson map[UUID]map[UUID]*PersonFile

	personRepo map[UUID]map[UUID]*PersonRepository
	repoPerson map[UUID]map[UUID]*PersonRepository
}

func NewPeopleRelations() *PeopleRelations {
	return &PeopleRelations{
		personFile: make(map[UUID]map[UUID]*PersonFile),
		filePerson: make(map[UUID]map[UUID]*PersonFile),
		personRepo: make(map[UUID]map[UUID]*PersonRepository),
		repoPerson: make(map[UUID]map[UUID]*PersonRepository),
	}
}

func (p *PeopleRelations) GetOrCreatePersonFile(personID UUID, fileID UUID) *PersonFile {
	pf, ok := p.personFile[personID]
	if !ok {
		pf = make(map[UUID]*PersonFile)
		p.personFile[personID] = pf
	}

	result, ok := pf[fileID]
	if !ok {
		result = NewPersonFile(personID, fileID)
		pf[fileID] = result
	}

	fp, ok := p.filePerson[fileID]
	if !ok {
		fp = make(map[UUID]*PersonFile)
		p.filePerson[fileID] = fp
	}

	if _, ok = fp[personID]; !ok {
		fp[personID] = result
	}

	return result
}

func (p *PeopleRelations) ListFiles() []*PersonFile {
	result := make([]*PersonFile, 100)
	for _, files := range p.personFile {
		for _, file := range files {
			result = append(result, file)
		}
	}
	return result
}

func (p *PeopleRelations) ListPeopleByFile(fileID UUID) map[UUID]*PersonFile {
	ps, ok := p.filePerson[fileID]
	if !ok {
		return nil
	}

	return ps
}

func (p *PeopleRelations) ListFilesByPerson(personID UUID) map[UUID]*PersonFile {
	fs, ok := p.personFile[personID]
	if !ok {
		return nil
	}

	return fs
}

func (p *PeopleRelations) GetOrCreatePersonRepo(personID UUID, repoID UUID) *PersonRepository {
	pf, ok := p.personRepo[personID]
	if !ok {
		pf = make(map[UUID]*PersonRepository)
		p.personRepo[personID] = pf
	}

	result, ok := pf[repoID]
	if !ok {
		result = NewPersonRepository(personID, repoID)
		pf[repoID] = result
	}

	fp, ok := p.repoPerson[repoID]
	if !ok {
		fp = make(map[UUID]*PersonRepository)
		p.repoPerson[repoID] = fp
	}

	if _, ok = fp[personID]; !ok {
		fp[personID] = result
	}

	return result
}

func (p *PeopleRelations) ListRepositories() []*PersonRepository {
	result := make([]*PersonRepository, 100)
	for _, repos := range p.personRepo {
		for _, repo := range repos {
			result = append(result, repo)
		}
	}
	return result
}

func (p *PeopleRelations) ListPeopleByRepo(repoID UUID) map[UUID]*PersonRepository {
	ps, ok := p.repoPerson[repoID]
	if !ok {
		return nil
	}

	return ps
}

func (p *PeopleRelations) ListReposByPerson(personID UUID) map[UUID]*PersonRepository {
	fs, ok := p.personRepo[personID]
	if !ok {
		return nil
	}

	return fs
}
