package transcript

import (
	"bytes"
	"os"
	"sort"
	"testing"
)

// TestTheTranscriptIsInCanonicalForm prende que cada arquivo de dados é
// exatamente o que o emissor escreveria para os valores dele.
//
// Sem isto, a primeira ferramenta que reescrevesse o catálogo (`cmd/mintuids`,
// `cmd/bookpages`) arrastaria junto, no mesmo diff, toda edição à mão que
// tivesse saído da forma — e a mudança dela se perderia no meio.
func TestTheTranscriptIsInCanonicalForm(t *testing.T) {
	want, err := Emit()
	if err != nil {
		t.Fatal(err)
	}
	onDisk, err := EmittedFilesOnDisk(".")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(onDisk)
	for _, name := range onDisk {
		if _, ok := want[name]; !ok {
			t.Errorf("%s parece um arquivo de dados, e o emissor não o escreve — "+
				"apague-o ou rode `go run ./cmd/transcriptfmt`", name)
		}
	}
	checked := 0
	for name, body := range want {
		got, err := os.ReadFile(name)
		if err != nil {
			t.Errorf("%s: o emissor o escreve e ele não existe (%v) — rode `go run ./cmd/transcriptfmt`", name, err)
			continue
		}
		checked++
		if !bytes.Equal(got, body) {
			t.Errorf("%s não está na forma canônica — rode `go run ./cmd/transcriptfmt` "+
				"(cd engine-go) e confira o diff", name)
		}
	}
	if checked < 50 {
		t.Fatalf("só %d arquivos conferidos — o emissor deixou de ver o catálogo", checked)
	}
}
