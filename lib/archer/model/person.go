package model

type Person struct {
	Name string
	ID   UUID

	email map[string]bool
}

func NewPerson(name string) *Person {
	return &Person{
		Name:  name,
		email: map[string]bool{},
		ID:    NewUUID("a"),
	}
}

func (p Person) AddEmail(email string) {
	p.email[email] = true
}
