package dbpets

import "context"

// CreatePetParams contains the data needed to create a new pet.
type CreatePetParams struct {
	ID   string
	Name string
	Type string
}

// UpdatePetNameParams contains the data needed to rename a pet.
type UpdatePetNameParams struct {
	ID   string
	Name string
}

// Queries exposes simple operations for working with pets.
type Queries struct{}

func (q *Queries) CreatePet(ctx context.Context, arg CreatePetParams) (Pet, error) {
	return Pet{ID: arg.ID, Name: arg.Name, Type: arg.Type}, nil
}

func (q *Queries) DeletePet(ctx context.Context, id string) error {
	return nil
}

func (q *Queries) GetPet(ctx context.Context, id string) (Pet, error) {
	return Pet{ID: id}, nil
}

func (q *Queries) ListPets(ctx context.Context) ([]Pet, error) {
	return nil, nil
}

func (q *Queries) UpdatePetName(ctx context.Context, arg UpdatePetNameParams) (Pet, error) {
	return Pet{ID: arg.ID, Name: arg.Name}, nil
}
