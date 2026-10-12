package testdb_test

import (
	"testing"

	"github.com/jameynakama/reallylonglink/internal/testdb"
)

func TestSwapDBName(t *testing.T) {
	expected := "pg://hello:moto@some.place/woof?one=1&two=2"
	if r := testdb.SwapDBName("pg://hello:moto@some.place/meow?one=1&two=2", "woof"); r != expected {
		t.Errorf("wanted %s but got %s", expected, r)
	}
}

func TestGetDBName(t *testing.T) {
	if r := testdb.GetDBName("pg://hello:moto@some.place/woof?one=1&two=2"); r != "woof" {
		t.Errorf("wanted woof but got %s", r)
	}
}
