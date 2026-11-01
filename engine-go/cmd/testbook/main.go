// Command testbook escreve o PDF de teste que o e2e serve como se fosse o livro.
//
// O arquivo do livro não pode entrar no repositório, e sem ele a bancada pulava os casos do
// leitor: uma suíte verde não dizia nada sobre essa tela. Este PDF tem o mesmo número de
// páginas e, em cada uma, só os NOMES das condições e das criaturas que o catálogo diz estarem
// nela, para o selo de página abrir na página certa e o leitor ter o que marcar. Nenhum texto do
// livro entra aqui.
//
//	go run ./cmd/testbook data/e2e-book.pdf
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"t20engine/domain/book"
)

// pagesInFile cobre a última página impressa mais a abertura do arquivo.
const pagesInFile = 410

// openingPages é quantas páginas do arquivo vêm antes da página 1 impressa, o padrão de
// `LIVRO_ABERTURA`.
const openingPages = 6

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("uso: testbook <arquivo.pdf>; recebi %d argumentos", len(os.Args)-1)
	}
	// A pasta pode não existir: num clone limpo, `data/` só nasce quando o app abre o banco.
	if err := os.MkdirAll(filepath.Dir(os.Args[1]), 0o755); err != nil {
		log.Fatalf("criar a pasta de %s: %v", os.Args[1], err)
	}
	if err := os.WriteFile(os.Args[1], renderPDF(namesByFilePage(), pagesInFile), 0o600); err != nil {
		log.Fatalf("escrever %s: %v", os.Args[1], err)
	}
}

// namesByFilePage diz que nomes cada página do ARQUIVO leva.
func namesByFilePage() map[int][]string {
	names := map[int][]string{}
	add := func(printedPage int, name string) {
		if printedPage > 0 {
			names[printedPage+openingPages] = append(names[printedPage+openingPages], name)
		}
	}
	for _, c := range book.Catalogs().Conditions {
		add(c.BookPage, c.Name)
	}
	for _, m := range book.Creatures() {
		add(m.BookPage, m.Name)
	}
	return names
}

// renderPDF monta o arquivo à mão, porque a forma é pequena e o app não precisa de uma
// biblioteca de PDF só para o teste. Os objetos são o catálogo (1), a árvore (2), a fonte (3)
// e, para cada página, o objeto dela e o conteúdo.
func renderPDF(names map[int][]string, pages int) []byte {
	out := []byte("%PDF-1.4\n")
	offsets := make([]int, 0, 3+2*pages)
	object := func(body string) {
		offsets = append(offsets, len(out))
		out = append(out, fmt.Sprintf("%d 0 obj\n%s\nendobj\n", len(offsets), body)...)
	}

	object("<< /Type /Catalog /Pages 2 0 R >>")
	kids := ""
	for page := range pages {
		kids += fmt.Sprintf("%d 0 R ", 4+2*page)
	}
	object(fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", kids, pages))
	object("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")
	for page := range pages {
		object(fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] "+
			"/Resources << /Font << /F1 3 0 R >> >> /Contents %d 0 R >>", 5+2*page))
		content := pageContent(page+1, names[page+1])
		object(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content))
	}

	table := len(out)
	out = append(out, fmt.Sprintf("xref\n0 %d\n0000000000 65535 f \n", len(offsets)+1)...)
	for _, offset := range offsets {
		out = append(out, fmt.Sprintf("%010d 00000 n \n", offset)...)
	}
	return append(out, fmt.Sprintf("trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets)+1, table)...)
}

// pageContent é o texto de uma página: o aviso de que é o livro de teste, e os nomes.
func pageContent(filePage int, names []string) string {
	lines := append([]string{fmt.Sprintf("Livro de teste do e2e, pagina %d do arquivo.", filePage)}, names...)
	content := "BT /F1 16 Tf 72 760 Td 22 TL\n"
	for _, line := range lines {
		content += "(" + pdfString(line) + ") Tj T*\n"
	}
	return content + "ET"
}

// pdfString escreve o texto em WinAnsi, com o escape que o formato pede. Letra fora dessa
// tabela vira `?`: os nomes do catálogo são português, e uma exceção aqui só muda a marca.
func pdfString(text string) string {
	out := make([]byte, 0, len(text))
	for _, r := range text {
		switch {
		case r == '(' || r == ')' || r == '\\':
			out = append(out, '\\', byte(r))
		case r > 255:
			out = append(out, '?')
		default:
			out = append(out, byte(r))
		}
	}
	return string(out)
}
