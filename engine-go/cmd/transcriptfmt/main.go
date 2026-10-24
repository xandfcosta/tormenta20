// Command transcriptfmt reescreve os arquivos de dados do livro transcrito na
// forma canônica — o `gofmt` do catálogo.
//
// Rode depois de editar um verbete à mão, quando o
// `TestTheTranscriptIsInCanonicalForm` reprovar:
//
//	cd engine-go && go run ./cmd/transcriptfmt
package main

import (
	"fmt"
	"os"

	"t20engine/cmd/internal/transcriptdir"
	"t20engine/domain/catalog/transcript"
)

func main() {
	dir, err := transcriptdir.Find()
	if err == nil {
		err = transcript.WriteSource(dir)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "transcriptfmt:", err)
		os.Exit(1)
	}
	fmt.Println("livro transcrito reescrito em", dir)
}
