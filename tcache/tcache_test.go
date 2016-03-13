package tcache

import (
	"log"
	"testing"
)

func TestNew(t *testing.T) {
	tc, err := New("../templates/html")
	if err != nil {
		t.Fatal(err)
	}
	log.Printf("%#v\n", tc["faq/staycurrent.inc"])
}
