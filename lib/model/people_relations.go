package model

type PeopleRelations struct {
	personFile map[ID]map[ID]*PersonFile
	filePerson map[ID]map[ID]*PersonFile

	personRepo map[ID]map[UUID]*PersonRepository
	repoPerson map[UUID]map[ID]*PersonRepository
}

func NewPeopleRelations() *PeopleRelations {
	return &PeopleRelations{
		personFile: make(map[ID]map[ID]*PersonFile),
		filePerson: make(map[ID]map[ID]*PersonFile),
		personRepo: make(map[ID]map[UUID]*PersonRepository),
		repoPerson: make(map[UUID]map[ID]*PersonRepository),
	}
}

func (p *PeopleRelations) GetOrCreatePersonFile(personID ID, fileID ID) *PersonFile {
	pf, ok := p.personFile[personID]
	if !ok {
		pf = make(map[ID]*PersonFile)
		p.personFile[personID] = pf
	}

	result, ok := pf[fileID]
	if !ok {
		result = NewPersonFile(personID, fileID)
		pf[fileID] = result
	}

	fp, ok := p.filePerson[fileID]
	if !ok {
		fp = make(map[ID]*PersonFile)
		p.filePerson[fileID] = fp
	}

	if _, ok = fp[personID]; !ok {
		fp[personID] = result
	}

	return result
}

func (p *PeopleRelations) ListFiles() []*PersonFile {
	var result []*PersonFile
	for _, files := range p.personFile {
		for _, file := range files {
			result = append(result, file)
		}
	}
	return result
}

func (p *PeopleRelations) ListPeopleByFile(fileID ID) map[ID]*PersonFile {
	ps, ok := p.filePerson[fileID]
	if !ok {
		return nil
	}

	return ps
}

func (p *PeopleRelations) ListFilesByPerson(personID ID) map[ID]*PersonFile {
	fs, ok := p.personFile[personID]
	if !ok {
		return nil
	}

	return fs
}

func (p *PeopleRelations) GetOrCreatePersonRepo(personID ID, repoID UUID) *PersonRepository {
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
		fp = make(map[ID]*PersonRepository)
		p.repoPerson[repoID] = fp
	}

	if _, ok = fp[personID]; !ok {
		fp[personID] = result
	}

	return result
}

func (p *PeopleRelations) ListRepositories() []*PersonRepository {
	var result []*PersonRepository
	for _, repos := range p.personRepo {
		for _, repo := range repos {
			result = append(result, repo)
		}
	}
	return result
}

func (p *PeopleRelations) ListPeopleByRepo(repoID UUID) map[ID]*PersonRepository {
	ps, ok := p.repoPerson[repoID]
	if !ok {
		return nil
	}

	return ps
}

func (p *PeopleRelations) ListReposByPerson(personID ID) map[UUID]*PersonRepository {
	fs, ok := p.personRepo[personID]
	if !ok {
		return nil
	}

	return fs
}
