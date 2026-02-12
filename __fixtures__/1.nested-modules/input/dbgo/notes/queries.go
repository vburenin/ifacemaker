package dbnotes

import "context"

// AddNoteParams contains the data needed to attach a note to a pet.
type AddNoteParams struct {
	PetID string
	Text  string
}

// GetNotesLimitParams restricts how many notes are returned.
type GetNotesLimitParams struct {
	PetID string
	Limit int32
}

// Queries exposes simple operations for working with pet notes.
type Queries struct{}

func (q *Queries) AddNote(ctx context.Context, arg AddNoteParams) (Note, error) {
	return Note{PetID: arg.PetID, Text: arg.Text}, nil
}

func (q *Queries) GetNotes(ctx context.Context, petID string) ([]Note, error) {
	return nil, nil
}

func (q *Queries) GetNotesLimit(ctx context.Context, arg GetNotesLimitParams) ([]Note, error) {
	return nil, nil
}
