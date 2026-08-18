package shardblob

import (
	"testing"

	"github.com/LYH2263/go-shardblob/internal/hashx"
)

func TestBug10_PutHasherResetBetweenObjects(t *testing.T) {
	s, err := OpenMemory(WithChunkSize(8))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a := []byte("alpha-object-data")
	b := []byte("beta-object-data!")
	id1, err := s.PutBytes(a)
	if err != nil {
		t.Fatal(err)
	}
	id2, err := s.PutBytes(b)
	if err != nil {
		t.Fatal(err)
	}
	want1 := fromHX(hashx.ObjectID(hashx.Default(), a))
	want2 := fromHX(hashx.ObjectID(hashx.Default(), b))
	if id1 != want1 {
		t.Fatalf("first object id %s want %s", id1, want1)
	}
	if id2 != want2 {
		t.Fatalf("second object hash mixed first: got %s want %s", id2, want2)
	}
}
