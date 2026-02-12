//go:build ignore

package pet_store

import (
	dbpets "github.com/vburenin/ifacemaker/__fixtures__/1.nested-modules/input/dbgo/pets"
	dbnotes "github.com/vburenin/ifacemaker/__fixtures__/1.nested-modules/input/dbgo/notes"
)

type storage struct {
	dbpets.Queries
	dbnotes.Queries
}
