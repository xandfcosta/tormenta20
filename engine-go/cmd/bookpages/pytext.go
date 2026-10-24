package main

import (
	"regexp"
	"strings"
	"t20engine/cmd/internal/pyfmt"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	"t20engine/infra/bookpdf"
)

// O script nasceu em Python, onde `\s`, `\w` e `\b` são UNICODE; no RE2 do Go
// são ASCII. O livro tem espaço não separável e letra acentuada, e trocar a
// classe trocaria o que casa sem erro nenhum — por isso as classes moram aqui,
// escritas por extenso.
const (
	spaceChars = `\t\n\v\f\r \x{1c}-\x{1f}\x{85}\p{Z}`
	pySpace    = `[` + spaceChars + `]`
	pyNonSpace = `[^` + spaceChars + `]`
)

var (
	blanksAndTabs = regexp.MustCompile(`[ \t]+`)
	anySpaces     = regexp.MustCompile(pySpace + `+`)
	hyphenBreak   = regexp.MustCompile(`-\n` + pySpace + `*`)
)

// fold é o `dobra` do Python: minúsculas e sem acento — ninguém digita til, e o
// índice usa caixa baixa.
//
//	fold("Índice  Remissivo") == "indice remissivo"
func fold(text string) string {
	var out strings.Builder
	for _, r := range norm.NFD.String(text) {
		if !unicode.Is(unicode.Mn, r) {
			out.WriteRune(r)
		}
	}
	return bookpdf.Strip(blanksAndTabs.ReplaceAllString(strings.ToLower(out.String()), " "))
}

// flatten é o `achata`: uma linha só, hífen de quebra emendado e todo espaço
// virando um.
func flatten(text string) string {
	return anySpaces.ReplaceAllString(fold(hyphenBreak.ReplaceAllString(text, "")), " ")
}

func isSpace(r rune) bool {
	return unicode.IsSpace(r) || unicode.Is(unicode.Z, r) || (r >= 0x1c && r <= 0x1f)
}

func isWord(r rune) bool { return unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' }

// atBoundary é o `\b` do Python em `pos` (índice de byte): de um lado um
// caractere de palavra e do outro não, com o fora do texto contando como "não".
func atBoundary(text string, pos int) bool {
	before, after := false, false
	if pos > 0 {
		r, _ := utf8.DecodeLastRuneInString(text[:pos])
		before = isWord(r)
	}
	if pos < len(text) {
		r, _ := utf8.DecodeRuneInString(text[pos:])
		after = isWord(r)
	}
	return before != after
}

// withSealInMiddle é o nome com o SELO da diagramação enfiado no meio dele.
//
// Conserto de um falso negativo que o guarda do bestiário denunciou: a ficha
// imprime o ND à direita do nome, e o texto extraído sai "cavaleiro do leopardo
// nd 9 sangrento". A comparação por substring reprovava a página CERTA.
//
// A folga é apertada de propósito: até dois pedaços de no máximo três letras
// entre uma palavra e a seguinte. Cabe "nd 9" e "t$ 45"; não cabe uma frase, o
// que impede o nome de casar espalhado por uma página de prosa.
//
// O Python era `\b p1 (?:\s+\S{1,3}){0,2}\s+ p2 … \b`; o casamento é à mão
// porque o `\b` do RE2 não enxerga letra acentuada, e a pergunta é só de
// EXISTÊNCIA — então basta percorrer todos os caminhos.
func withSealInMiddle(target, text string) bool {
	var parts []string
	for _, p := range strings.Split(target, " ") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) < 2 {
		return false
	}
	for from := 0; ; {
		at := strings.Index(text[from:], parts[0])
		if at < 0 {
			return false
		}
		start := from + at
		if atBoundary(text, start) && restMatches(text, start+len(parts[0]), parts[1:]) {
			return true
		}
		_, size := utf8.DecodeRuneInString(text[start:])
		from = start + size
	}
}

// restMatches casa, a partir de `pos`, o separador e as partes que faltam.
func restMatches(text string, pos int, parts []string) bool {
	if len(parts) == 0 {
		return atBoundary(text, pos)
	}
	for _, end := range separatorEnds(text, pos, 0) {
		if strings.HasPrefix(text[end:], parts[0]) && restMatches(text, end+len(parts[0]), parts[1:]) {
			return true
		}
	}
	return false
}

// separatorEnds são todas as posições em que `(?:\s+\S{1,3}){0,2}\s+` pode
// terminar quando começa em `pos`, com `seals` repetições já gastas.
func separatorEnds(text string, pos, seals int) []int {
	var out []int
	for _, afterSpace := range runEnds(text, pos, isSpace, 0) {
		out = append(out, afterSpace)
		if seals == 2 {
			continue
		}
		for _, afterSeal := range runEnds(text, afterSpace, func(r rune) bool { return !isSpace(r) }, 3) {
			out = append(out, separatorEnds(text, afterSeal, seals+1)...)
		}
	}
	return out
}

// runEnds são os fins possíveis de uma corrida de 1 a `most` caracteres que
// satisfazem `keep` (`most` 0 é sem teto).
func runEnds(text string, pos int, keep func(rune) bool, most int) []int {
	var out []int
	for n := 0; pos < len(text) && (most == 0 || n < most); n++ {
		r, size := utf8.DecodeRuneInString(text[pos:])
		if !keep(r) {
			break
		}
		pos += size
		out = append(out, pos)
	}
	return out
}

// pyStrList é o `repr` de uma lista de strings do Python.
func pyStrList(items []string) string {
	quoted := make([]string, len(items))
	for i, s := range items {
		quoted[i] = pyfmt.Repr(s)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
