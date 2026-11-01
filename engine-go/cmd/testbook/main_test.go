package main

import (
	"strings"
	"testing"
)

// O nome vai na página do ARQUIVO, e não na impressa: é ela que o leitor abre. A página impressa
// 394 é a do Abalado (p394), e com seis de abertura ela é a 400 do arquivo.
func TestAConditionGoesOnItsPageOfTheFile(t *testing.T) {
	names := namesByFilePage()
	found := false
	for _, name := range names[400] {
		found = found || name == "Abalado"
	}
	if !found {
		t.Errorf("a página 400 do arquivo não traz Abalado: %v", names[400])
	}
}

// O arquivo tem de ser um PDF que se abre, e o que importa nele é o que o servidor confere na
// porta (o cabeçalho) e o que o pdf.js lê (a contagem e a tabela no fim). Quem prova que ele abre
// de verdade é o e2e do leitor.
func TestThePDFHasTheShapeAReaderNeeds(t *testing.T) {
	pdf := string(renderPDF(map[int][]string{2: {"Lobo (grande)"}}, 3))

	if !strings.HasPrefix(pdf, "%PDF-") {
		t.Errorf("o arquivo não começa como PDF: %q", pdf[:8])
	}
	if !strings.Contains(pdf, "/Count 3") || strings.Count(pdf, "/Type /Page ") != 3 {
		t.Errorf("a árvore não tem as 3 páginas")
	}
	if !strings.Contains(pdf, `(Lobo \(grande\)) Tj`) {
		t.Error("o nome não saiu com o escape que o formato pede")
	}
	if !strings.HasSuffix(pdf, "%%EOF\n") || !strings.Contains(pdf, "startxref") {
		t.Error("o arquivo não fecha com a tabela de referências")
	}
}
