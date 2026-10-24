package main

import (
	"fmt"
	"sort"
	"strings"
	"t20engine/cmd/internal/pyfmt"

	"t20engine/domain/catalog/transcript"
	"t20engine/infra/bookpdf"
)

// checkResult é uma magia conferida: o que o livro e o catálogo têm, e o que
// diverge.
type checkResult struct {
	id         string
	title      string
	page       int
	book       []bookAugment
	catalog    []transcript.SpellAugment
	complaints []string
}

// check compara o catálogo com o livro.
func check(spell *transcript.Spell, read *bookSpell) checkResult {
	fromBook := augmentsOfBody(read.body)
	fromCatalog := spell.Augments
	var complaints []string

	// ÂNCORA: círculo e escola. Sem ela, não é esta magia.
	if read.circle != spell.Circle {
		complaints = append(complaints, fmt.Sprintf("círculo: livro %d, catálogo %d", read.circle, spell.Circle))
	}
	if bookpdf.Key(read.school) != bookpdf.Key(spell.School) {
		complaints = append(complaints, fmt.Sprintf("escola: livro %s, catálogo %s", pyfmt.Repr(read.school), pyfmt.Repr(spell.School)))
	}
	if len(fromBook) != len(fromCatalog) {
		complaints = append(complaints, fmt.Sprintf("aprimoramentos: livro %d, catálogo %d", len(fromBook), len(fromCatalog)))
	} else {
		// Par a par SÓ quando as contagens batem: com uma faltando, parear por
		// índice inventa uma queixa de custo para cada um depois do buraco.
		for i := range fromBook {
			b, c := fromBook[i], fromCatalog[i]
			if b.pmCost != c.PMCost {
				complaints = append(complaints, fmt.Sprintf("[%d] custo: livro +%d PM, catálogo +%d PM", i, b.pmCost, c.PMCost))
			}
			if (b.classOnly == "") != (c.ClassOnly == "") {
				complaints = append(complaints, fmt.Sprintf("[%d] classOnly: livro %s, catálogo %s",
					i, pyfmt.OptionalRepr(b.classOnly), pyfmt.OptionalRepr(c.ClassOnly)))
			}
		}
	}
	return checkResult{spell.ID, read.title, read.page - pdfOffset, fromBook, fromCatalog, complaints}
}

// similarity é quanto duas frases dividem de vocabulário, de 0 a 1. Jaccard e
// não distância de edição: o catálogo REESCREVE as frases em forma curta, e o
// que sobrevive é o vocabulário, não a sequência.
func similarity(a, b string) float64 {
	wordsA := wordSet(bookpdf.NormalizePhrase(a))
	wordsB := wordSet(bookpdf.NormalizePhrase(b))
	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0.0
	}
	shared := 0
	for w := range wordsA {
		if wordsB[w] {
			shared++
		}
	}
	return float64(shared) / float64(len(wordsA)+len(wordsB)-shared)
}

func wordSet(phrase string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.Fields(phrase) {
		out[w] = true
	}
	return out
}

// similarityFloor separa duas frases que por acaso dividem "muda o alcance
// para" da MESMA frase reescrita. Calibrado na Âncora Dimensional, na Potência
// Divina e na Consagrar, que dão 0,45+.
const similarityFloor = 0.42

// candidate é um aprimoramento do catálogo e quanto ele parece o faltante.
type candidate struct {
	score float64
	id    string
	index int
	desc  string
}

// orphanReport procura cada aprimoramento que falta na magia dele no catálogo
// INTEIRO: um aprimoramento pendurado na magia ERRADA é transcrição a MOVER, e
// conta duas vezes no relatório normal.
//
// O terreno é só quem tem MENOS no catálogo que no livro, e isso não é
// economia: rodando sobre todas, ele acusou 105 "a transcrever" contra as 44 da
// contagem, porque o catálogo CONDENSA as frases de propósito.
//
// O que ele devolve é PISTA: "aumenta o número de alvos em +1" aparece igual em
// várias magias, e só ler as DUAS no livro diz se é deslocamento.
func orphanReport(results []checkResult) {
	var all []candidate
	for _, s := range transcript.Spells {
		for i, a := range s.Augments {
			all = append(all, candidate{0, s.ID, i, a.Description})
		}
	}
	found, ownerless := 0, 0
	fmt.Print("--- APRIMORAMENTOS QUE FALTAM, e onde eles podem estar ---\n\n")
	for _, r := range results {
		if len(r.book) <= len(r.catalog) {
			continue
		}
		var fromCatalog []string
		for _, a := range r.catalog {
			fromCatalog = append(fromCatalog, bookpdf.NormalizePhrase(a.Description))
		}
		for _, a := range r.book {
			if anySimilar(a.text, fromCatalog) {
				continue
			}
			best := candidate{0.0, "", 0, ""}
			first := true
			for _, c := range all {
				if c.id == r.id {
					continue
				}
				c.score = similarity(a.text, c.desc)
				if first || greaterCandidate(c, best) {
					best, first = c, false
				}
			}
			mark := fmt.Sprintf("+%d PM", a.pmCost)
			if a.cantrip {
				mark = "Truque"
			}
			if best.score >= similarityFloor {
				found++
				fmt.Printf("%s (p%d) %s\n", r.id, r.page, mark)
				fmt.Printf("   ESTÁ EM %s[%d] (parecença %.2f)\n", best.id, best.index, best.score)
				fmt.Printf("   catálogo: %s\n", pyfmt.FirstRunes(best.desc, 110))
			} else {
				ownerless++
				fmt.Printf("%s (p%d) %s: FALTA MESMO\n", r.id, r.page, mark)
				fmt.Printf("   livro: %s\n", pyfmt.FirstRunes(a.text, 110))
			}
		}
	}
	fmt.Printf("\nDESLOCADOS (existem, na magia errada): %d\n", found)
	fmt.Printf("A TRANSCREVER (não existem em lugar nenhum): %d\n", ownerless)
}

func anySimilar(text string, phrases []string) bool {
	for _, p := range phrases {
		if similarity(text, p) >= similarityFloor {
			return true
		}
	}
	return false
}

// greaterCandidate é a ordem da tupla (parecença, id, índice, texto) do Python,
// de quem o maior vence.
func greaterCandidate(a, b candidate) bool {
	if a.score != b.score {
		return a.score > b.score
	}
	if a.id != b.id {
		return a.id > b.id
	}
	if a.index != b.index {
		return a.index > b.index
	}
	return a.desc > b.desc
}

// singleSpellReport confere UMA magia e mostra os dois lados.
func singleSpellReport(id string, results []checkResult) {
	i := sort.Search(len(results), func(i int) bool { return results[i].id >= id })
	if i == len(results) || results[i].id != id {
		fmt.Printf("%s: NÃO MEDIDA — o cabeçalho não foi achado no capítulo\n", id)
		return
	}
	r := results[i]
	fmt.Printf("=== %s (p%d) — %s\n", r.title, r.page, r.id)
	fmt.Printf("\nLIVRO (%d aprimoramentos):\n", len(r.book))
	for _, a := range r.book {
		mark := fmt.Sprintf("+%d PM", a.pmCost)
		if a.cantrip {
			mark = "Truque"
		}
		only := ""
		if a.classOnly != "" {
			only = " (Apenas " + a.classOnly + ")"
		}
		fmt.Printf("  %s%s: %s\n", mark, only, pyfmt.FirstRunes(a.text, 300))
	}
	fmt.Printf("\nCATÁLOGO (%d aprimoramentos):\n", len(r.catalog))
	for _, a := range r.catalog {
		only := ""
		if a.ClassOnly != "" {
			only = " (classOnly=" + a.ClassOnly + ")"
		}
		fmt.Printf("  +%d PM%s: %s\n", a.PMCost, only, pyfmt.FirstRunes(a.Description, 300))
	}
	if len(r.complaints) > 0 {
		fmt.Println("\nQUEIXAS:")
		for _, q := range r.complaints {
			fmt.Println("  -", q)
		}
	}
}
