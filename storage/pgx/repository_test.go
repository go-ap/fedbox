//go:build storage_pgx || storage_all || (!storage_boltdb && !storage_fs && !storage_badger && !storage_sqlite)

package pgx

import "testing"

func TestNew(t *testing.T) {
	t.Skipf("TODO")
}

func TestRepo_Open(t *testing.T) {
	t.Skipf("TODO")
}

func TestRepo_Close(t *testing.T) {
	t.Skipf("TODO")
}

func TestRepo_GenerateID(t *testing.T) {
	t.Skipf("TODO")
}

func TestRepo_Load(t *testing.T) {
	t.Skipf("TODO")
}

func TestRepo_LoadActivities(t *testing.T) {
	t.Skipf("TODO")
}

func TestRepo_LoadActors(t *testing.T) {
	t.Skipf("TODO")
}

func TestRepo_LoadCollection(t *testing.T) {
	t.Skipf("TODO")
}

func TestRepo_LoadObjects(t *testing.T) {
	t.Skipf("TODO")
}

func TestRepo_UpdateObject(t *testing.T) {
	t.Skipf("TODO")
}

func TestRepo_DeleteObject(t *testing.T) {
	t.Skipf("TODO")
}

func TestRepo_SaveObject(t *testing.T) {
	t.Skipf("TODO")
}

func TestRepo_SaveActivity(t *testing.T) {
	t.Skipf("TODO")
}

func TestRepo_DeleteActor(t *testing.T) {
	t.Skipf("TODO")
}

func TestRepo_IsLocalIRI(t *testing.T) {
	t.Skipf("TODO")
}

func TestRepo_SaveActor(t *testing.T) {
	t.Skipf("TODO")
}

func TestRepo_UpdateActor(t *testing.T) {
	t.Skipf("TODO")
}
