// Command spells confere as magias contra o livro: a COBERTURA e os
// APRIMORAMENTOS (ALE-340, ALE-456).
//
// Rodar é ATO DELIBERADO, como o `genoracle`: a ferramenta PROPÕE, o diff se
// revisa contra o PDF, e é a revisão que decide. Ele NÃO escreve no catálogo.
//
//	go run ./cmd/audit/spells                 # cobertura + todas
//	go run ./cmd/audit/spells --cobertura     # só "falta alguma?"
//	go run ./cmd/audit/spells --suspeitas     # só as que divergem
//	go run ./cmd/audit/spells --magia luz     # uma, com o texto do livro
//	go run ./cmd/audit/spells --orfaos        # cada aprimoramento faltante no catálogo INTEIRO
//
// São DUAS perguntas. "As magias que eu tenho estão certas?" percorre o
// CATÁLOGO; "o livro tem alguma que eu não tenho?" é a COBERTURA, e percorre o
// LIVRO. Uma corrida com `0 divergentes` não diz nada sobre a segunda — e a
// épica carregou por semanas um "199 contra 198" que ninguém tinha medido. São
// 198: as duas listas do índice somam 226 entradas, e 28 magias estão nas DUAS.
//
// Como ele sabe que leu a magia certa: o cabeçalho `Lista N (Escola)` tem de
// bater com círculo e escola do catálogo. Sem ele a magia sai NÃO MEDIDA, que
// não é "correta".
//
// Compara TEXTO e não só CONTAGEM: o `+1 PM` da Invisibilidade (p195) tem um
// aprimoramento dos dois lados, e o catálogo dizia "alcance curto" onde o livro
// diz "o alcance para toque".
//
// A leitura é por COORDENADA (`infra/bookpdf`): `pdftotext -layout` junta
// colunas vizinhas, e foi isso que derrubou 15 magias na medição da issue.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"t20engine/domain/catalog/transcript"
	"t20engine/infra/bookpdf"
)

// firstPage e lastPage são o capítulo das magias no PDF (p178-211).
const (
	firstPage = 184
	lastPage  = 217
)

// pdfOffset é a distância entre a página do PDF e a do livro.
const pdfOffset = 6

func main() {
	spell := flag.String("magia", "", "confere UMA magia e mostra o texto do livro")
	// --suspeitas é aceito e não muda nada: o relatório já só lista as que
	// divergem. Fica pela compatibilidade com quem o chama.
	flag.Bool("suspeitas", false, "só as magias que divergem")
	orphans := flag.Bool("orfaos", false, "procura cada aprimoramento faltante no catálogo INTEIRO")
	coverageOnly := flag.Bool("cobertura", false, "só a cobertura: quantas o livro tem × quantas entraram")
	flag.Parse()

	book, err := bookpdf.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	catalog := catalogByID()

	// A COBERTURA VEM PRIMEIRO, e sempre: ela responde "falta alguma?".
	if *coverageOnly {
		os.Exit(min(1, coverageReport(book, catalog)))
	}
	if *spell == "" && !*orphans {
		coverageReport(book, catalog)
		fmt.Println()
	}
	names := map[string]string{}
	for _, s := range transcript.Spells {
		names[bookpdf.Key(s.Name)] = s.ID
	}
	read := spellsInBook(book, names)

	var unmeasured []string
	for _, s := range transcript.Spells {
		if _, ok := read[s.ID]; !ok {
			unmeasured = append(unmeasured, s.ID)
		}
	}
	sort.Strings(unmeasured)
	var readIDs []string
	for id := range read {
		readIDs = append(readIDs, id)
	}
	sort.Strings(readIDs)
	var results, divergent []checkResult
	for _, id := range readIDs {
		r := check(catalog[id], read[id])
		results = append(results, r)
		if len(r.complaints) > 0 {
			divergent = append(divergent, r)
		}
	}

	if *orphans {
		orphanReport(results)
		return
	}
	if *spell != "" {
		singleSpellReport(*spell, results)
		return
	}

	fmt.Printf("magias no catálogo:  %d\n", len(transcript.Spells))
	fmt.Printf("casadas com o livro: %d\n", len(read))
	fmt.Printf("NÃO MEDIDAS:         %d\n", len(unmeasured))
	fmt.Printf("com divergência:     %d\n", len(divergent))
	if len(unmeasured) > 0 {
		fmt.Println("\n--- NÃO MEDIDAS (o cabeçalho não foi achado; NÃO quer dizer corretas) ---")
		for _, id := range unmeasured {
			fmt.Printf("  %s  (livro p%d)\n", id, catalog[id].BookPage)
		}
	}
	fmt.Println("\n--- DIVERGENTES ---")
	for _, r := range divergent {
		fmt.Printf("\n%s  (p%d)  livro=%d catálogo=%d\n", r.id, r.page, len(r.book), len(r.catalog))
		for _, q := range r.complaints {
			fmt.Println("   -", q)
		}
	}
}

// catalogByID é o catálogo pelo id; a ordem de iteração, quando importa, é a
// de `transcript.Spells`.
func catalogByID() map[string]*transcript.Spell {
	out := make(map[string]*transcript.Spell, len(transcript.Spells))
	for i := range transcript.Spells {
		out[transcript.Spells[i].ID] = &transcript.Spells[i]
	}
	return out
}
