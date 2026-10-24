package main

import (
	"strings"

	"t20engine/domain/catalog/transcript"
	"t20engine/infra/bookpdf"
)

// columnWidth é a largura de uma coluna de texto, medida (o corpo começa em
// 70,9 e a linha mais longa acaba perto de 290): o itálico começa onde a FRASE
// está, e não onde a coluna começa.
const columnWidth = 220.0

// neighbourLine separa "linha vizinha" de "outro parágrafo"; a linha mede 12,2
// pontos, e o teto é maior de propósito.
const neighbourLine = 14.0

// piece é uma peça que o catálogo conhece: o id e a categoria.
type piece struct{ id, category string }

type candidate struct {
	piece
	appliesTo []string
}

// overlays são as categorias que são SOBREPOSIÇÃO; o resto é item, e item é o
// que pode ser base.
var overlays = map[string]bool{"improvement": true, "material": true, "weapon-enchant": true, "armor-enchant": true}

// connectives LIGAM peças sem serem peça. Também aparecem DENTRO de um nome
// ("machado de guerra"), e por isso o descarte vem depois de todas as
// tentativas de casar, nunca antes.
var connectives = map[string]bool{"de": true, "do": true, "da": true, "e": true}

// composition é o que a frase em itálico diz: as peças achadas, as palavras
// que não casaram com nada, e a família do item.
type composition struct {
	pieces   []piece
	leftover []string
	family   string
}

// knownPieces é nome normalizado → candidatos, com as variantes de GÊNERO: o
// livro concorda a peça com o item ("defensora" na cota, "defensor" no escudo)
// e o catálogo guarda UMA grafia. É LISTA porque a mesma palavra é duas peças —
// o "defensor" de arma (p335) e o de armadura (p338) —, e a família desempata.
func knownPieces(items []transcript.Item) map[string][]candidate {
	table := map[string][]candidate{}
	for _, it := range items {
		if it.ComposedOf != nil {
			continue
		}
		for _, variant := range genderVariants(it.Name) {
			k := bookpdf.Key(variant)
			table[k] = append(table[k], candidate{piece{it.ID, it.Category}, it.AppliesTo})
		}
	}
	return table
}

// genderVariants são as grafias possíveis de um nome. A flexão cai na CABEÇA do
// nome, que nem sempre é a última palavra ("banhada a ouro", p340): as duas
// pontas são tentadas.
func genderVariants(name string) []string {
	out := []string{name}
	seen := map[string]bool{name: true}
	words := strings.Fields(name)
	for _, i := range []int{0, len(words) - 1} {
		for _, inflected := range inflections(words[i]) {
			swapped := append([]string(nil), words...)
			swapped[i] = inflected
			v := strings.Join(swapped, " ")
			if !seen[v] {
				seen[v] = true
				out = append(out, v)
			}
		}
	}
	return out
}

// inflections são três pares, que são os do português e os que o catálogo
// usa. Gerar MAIS traria grafia que não existe e daria a ela a chance de casar
// com outra peça.
func inflections(word string) []string {
	lower := strings.ToLower(word)
	for _, pair := range [][2]string{{"ão", "ã"}, {"or", "ora"}, {"o", "a"}} {
		for _, swap := range [][2]string{{pair[0], pair[1]}, {pair[1], pair[0]}} {
			if strings.HasSuffix(lower, swap[0]) {
				return []string{word[:len(word)-len(swap[0])] + swap[1]}
			}
		}
	}
	return nil
}

func familyOf(category string) string {
	switch {
	case strings.HasPrefix(category, "weapon"):
		return "weapon"
	case strings.HasPrefix(category, "armor"):
		return "armor"
	}
	return category
}

// compose quebra a frase nas peças conhecidas, pelo TRECHO MAIS LONGO primeiro
// ("machado de guerra" antes de "machado"). A FAMÍLIA sai da BASE e desempata o
// resto; antes de a base aparecer, o único candidato aceito é um item.
func compose(phrase string, pieces map[string][]candidate) composition {
	words := strings.FieldsFunc(phrase, bookpdf.IsPySpace)
	var c composition
	for i := 0; i < len(words); {
		matched := false
		for size := min(5, len(words)-i); size > 0; size-- {
			p, ok := choose(pieces[bookpdf.Key(strings.Join(words[i:i+size], " "))], c.family)
			if !ok {
				continue
			}
			c.pieces = append(c.pieces, p)
			if c.family == "" && !overlays[p.category] {
				c.family = familyOf(p.category)
			}
			i += size
			matched = true
			break
		}
		if !matched {
			if !connectives[strings.ToLower(words[i])] {
				c.leftover = append(c.leftover, words[i])
			}
			i++
		}
	}
	return c
}

// choose é o candidato que CABE: a sobreposição pela família, o resto por ser
// item.
func choose(candidates []candidate, family string) (piece, bool) {
	for _, c := range candidates {
		if !overlays[c.category] {
			if family == "" {
				return c.piece, true
			}
			continue
		}
		if family == "" || len(c.appliesTo) == 0 || contains(c.appliesTo, family) {
			return c.piece, true
		}
	}
	return piece{}, false
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

// entryComposition é (frase em itálico, composição) — do que o item é feito,
// dito pelo livro. O itálico também marca o próprio item mais adiante, o nome
// de uma MAGIA ou um encanto só CONCEDIDO; duas perguntas separam a composição
// dos outros: a frase é CONTÍGUA no verbete, e nomeia peça conhecida? Vazio é
// estado legítimo: a azagaia dos relâmpagos não tem encanto nenhum.
func entryComposition(book *bookpdf.Book, entry *bookpdf.Entry, pieces map[string][]candidate) (string, composition) {
	bodyX := entry.Column.Body
	text := bookpdf.NormalizePhrase(bookpdf.Join(entry.Body))
	var inColumn []bookpdf.Positioned
	for _, it := range book.Italics(entry.PDF) {
		if entry.Y0-2 <= it.Y && it.Y <= entry.Y1+2 && bodyX-slack <= it.X && it.X <= bodyX+columnWidth {
			inColumn = append(inColumn, it)
		}
	}
	for _, run := range sequences(inColumn) {
		for start := range run {
			for end := len(run); end > start; end-- {
				phrase := joinHyphenated(run[start:end])
				if !strings.Contains(text, bookpdf.NormalizePhrase(phrase)) {
					continue
				}
				if found := compose(phrase, pieces); len(found.pieces) > 0 {
					return phrase, found
				}
			}
		}
	}
	return "", composition{}
}

// sequences agrupa os trechos em linhas vizinhas, cada grupo em ordem.
func sequences(runs []bookpdf.Positioned) [][]string {
	var out [][]string
	var group []string
	lastY := 0.0
	for _, r := range runs {
		if len(group) > 0 && r.Y-lastY > neighbourLine {
			out = append(out, group)
			group = nil
		}
		group = append(group, r.Text)
		lastY = r.Y
	}
	if len(group) > 0 {
		out = append(out, group)
	}
	return out
}

// joinHyphenated remonta a frase que a quebra partiu. O HÍFEN às vezes vem
// SOZINHO num trecho — "refor" + "-" + "çada" (Baluarte anão, p340) —, e é
// gancho dos dois jeitos: entra sem espaço, e o trecho seguinte cola nele.
func joinHyphenated(runs []string) string {
	phrase := ""
	for _, r := range runs {
		switch {
		case strings.HasSuffix(phrase, "-"):
			phrase = phrase[:len(phrase)-1] + r
		case r == "-":
			phrase += "-"
		case phrase != "":
			phrase += " " + r
		default:
			phrase = r
		}
	}
	return bookpdf.Collapse(phrase)
}
