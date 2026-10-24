package main

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// pageSpan é a faixa de páginas impressas de um capítulo, inclusiva.
type pageSpan struct{ first, last int }

// String imprime a faixa como a tupla do Python: "(178, 211)".
func (s *pageSpan) String() string {
	if s == nil {
		return "None"
	}
	return fmt.Sprintf("(%d, %d)", s.first, s.last)
}

// lookup é o que a escada precisa de uma entrada: o nome (ou o id, quando ela
// não tem nome) e a página que já estava (0 é nenhuma).
type lookup struct {
	name, id string
	had      int
}

func (l lookup) label() string {
	if l.name != "" {
		return l.name
	}
	return l.id
}

// bySignature é a página em que o livro ABRE este verbete, pela assinatura do
// catálogo. Só serve quando a resposta é ÚNICA: duas páginas casando significam
// que a assinatura não distingue, e escolher uma seria dar a uma dúvida a cara
// de uma prova.
func (b *book) bySignature(name, signature string, span pageSpan) int {
	if signature == "" {
		return 0
	}
	pattern := regexp.MustCompile(strings.ReplaceAll(signature, "{nome}", regexp.QuoteMeta(fold(name))))
	var found []int
	for printed := span.first; printed <= span.last; printed++ {
		if printed < b.indexStart && pattern.MatchString(b.page(printed)) {
			found = append(found, printed)
		}
	}
	if len(found) == 1 {
		return found[0]
	}
	return 0
}

// resolve é a escada, da fonte mais forte para a mais fraca: a página (0 é
// nenhuma) e a fonte, ou o motivo de não haver página.
//
// Página que não confere não é usada, venha de onde vier — inclusive a que já
// estava no catálogo, que é como as cinco erradas do bestiário foram achadas.
func (b *book) resolve(entry lookup, qualifiers []string, span *pageSpan, signature string) (int, string) {
	name := entry.label()

	// A ASSINATURA vem antes de tudo, inclusive do que já estava: ela é a única
	// fonte que sabe a diferença entre a página que CITA o verbete e a que o
	// ABRE. Foi o que corrigiu o Lobo (289→290), o Troll (309→308) e o Trog
	// (292→291).
	if span != nil {
		if page := b.bySignature(name, signature, *span); page != 0 {
			return page, "assinatura do livro"
		}
	}

	if entry.had != 0 && b.confirms(name, entry.had) {
		return entry.had, "já estava"
	}

	for _, page := range b.byIndex(name, qualifiers...) {
		if b.confirms(name, page) {
			return page, "índice"
		}
		// JANELA em volta do que o índice diz, medida e não folga arbitrária: o
		// índice às vezes aponta a ABERTURA da seção e não o verbete — "Raio
		// Arcano" está indexado em 38 e escrito em 40.
		if near := b.nearestThatConfirms(name, page, 4); near != 0 {
			return near, "índice (janela)"
		}
	}

	if span != nil {
		var candidates []int
		for _, p := range b.byTitle(name, *span) {
			if b.confirms(name, p) {
				candidates = append(candidates, p)
			}
		}
		if len(candidates) == 1 {
			return candidates[0], "título na faixa"
		}
		if len(candidates) > 1 {
			// A ÚLTIMA do trecho, CALIBRADA contra o próprio livro: nos 52 itens
			// que o índice resolve E que têm mais de uma candidata, a página do
			// índice é a última em 48. O nome aparece antes na TABELA e depois
			// na descrição, e é a descrição que o índice aponta. Erra em 4 de 52,
			// e a página escolhida CONTÉM a entrada.
			//
			// Trecho curto porque candidatas espalhadas pelo livro são
			// ocorrências diferentes do mesmo nome, e aí não há o que escolher.
			lo, hi := candidates[0], candidates[0]
			for _, c := range candidates {
				lo, hi = min(lo, c), max(hi, c)
			}
			if hi-lo <= 8 {
				return hi, "última do trecho (calibrado)"
			}
			return 0, fmt.Sprintf("ambígua (%d páginas na faixa)", len(candidates))
		}
	}

	// O nome sem o DEGRAU: "Fúria +3" e "Magias (2° círculo)" não são verbetes —
	// o livro escreve o poder uma vez e a tabela da classe mostra a progressão.
	// A volta sem o degrau não leva a assinatura, como no Python.
	if base := withoutStep(name); base != name {
		if page, source := b.resolve(lookup{name: base, id: entry.id}, qualifiers, span, ""); page != 0 {
			return page, source + " (sem o degrau)"
		}
	}

	if entry.had != 0 {
		return 0, fmt.Sprintf("a que estava (p%d) não confere", entry.had)
	}
	return 0, "sem fonte"
}

var (
	stepSuffix  = regexp.MustCompile(pySpace + `*(?:\+` + pySpace + `*[0-9]+(?:d[0-9]+)?|\(` + pySpace + `*[0-9]+.*\))` + pySpace + `*$`)
	digitSuffix = regexp.MustCompile(`[0-9]+` + pySpace + `*$`)
)

// withoutStep tira o degrau de progressão e o prefixo de família do nome do
// poder. A família ("Música:", "Caminho:") é rótulo nosso para agrupar.
//
//	withoutStep("Fúria +3")                 == "Fúria"
//	withoutStep("Magias (2° círculo)")      == "Magias"
//	withoutStep("Música: Melodia Curativa") == "Melodia Curativa"
//
// O Python era um regex só com `\b[0-9]+` na terceira alternativa; o `\b` do
// RE2 é ASCII, então essa alternativa é casada à parte e vence a de começo mais
// à esquerda, que é o que o `re.sub` faria.
func withoutStep(name string) string {
	cut := len(name) + 1
	if loc := stepSuffix.FindStringIndex(name); loc != nil {
		cut = loc[0]
	}
	if loc := digitSuffix.FindStringIndex(name); loc != nil && atBoundary(name, loc[0]) {
		start := loc[0]
		for start > 0 {
			r, size := utf8.DecodeLastRuneInString(name[:start])
			if !isSpace(r) {
				break
			}
			start -= size
		}
		cut = min(cut, start)
	}
	bare := name
	if cut <= len(name) {
		bare = name[:cut]
	}
	bare = strings.TrimFunc(bare, isSpace)
	if _, after, ok := strings.Cut(bare, ": "); ok {
		return after
	}
	return bare
}

// nearestThatConfirms é a página mais próxima de `target` em que o nome
// aparece, ou 0.
func (b *book) nearestThatConfirms(name string, target, window int) int {
	for distance := 1; distance <= window; distance++ {
		for _, page := range []int{target + distance, target - distance} {
			if page > 0 && b.confirms(name, page) {
				return page
			}
		}
	}
	return 0
}

// spanOfConfirmed é o envelope do que já se resolveu na primeira volta, que
// vira a faixa de busca dos que sobraram: o capítulo se declara pelo que o
// índice já apontou, e não por um número escrito à mão que envelhece.
func spanOfConfirmed(pages []int) *pageSpan {
	if len(pages) < 3 {
		return nil
	}
	s := pageSpan{pages[0], pages[0]}
	for _, p := range pages {
		s.first, s.last = min(s.first, p), max(s.last, p)
	}
	return &s
}
