package layers

import "testing"

func TestCatalogueIsValid(t *testing.T) {
	if err := Validate(All()); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsDuplicateIDs(t *testing.T) {
	layer := All()[0]
	if err := Validate([]Layer{layer, layer}); err == nil {
		t.Fatal("expected an error for duplicate ids")
	}
}

func TestValidateRejectsStaleLimitBelowRefresh(t *testing.T) {
	layer := All()[0]
	layer.StaleAfterSeconds = layer.RefreshSeconds - 1
	if err := Validate([]Layer{layer}); err == nil {
		t.Fatal("expected an error when the stale limit is below the refresh interval")
	}
}

func TestAllReturnsACopy(t *testing.T) {
	first := All()
	first[0].Sources[0] = "changed"
	if All()[0].Sources[0] == "changed" {
		t.Fatal("All must not expose the catalogue's backing slices")
	}
}

func TestByID(t *testing.T) {
	if _, ok := ByID("river-levels"); !ok {
		t.Fatal("river-levels should exist")
	}
	if _, ok := ByID("missing"); ok {
		t.Fatal("missing should not exist")
	}
}
