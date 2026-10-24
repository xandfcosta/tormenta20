package main

import (
	"regexp"
	"strings"

	"t20engine/infra/bookpdf"
)

// As classes do Python: `\s`, `\d` e `\w` são UNICODE lá e ASCII no RE2.
const (
	sp    = `[\t\n\v\f\r \x{1c}-\x{1f}\x{85}\p{Z}]`
	digit = `\p{Nd}`
)

// O VOCABULÁRIO DA COLUNA DE PRÉ-REQUISITOS, levantado do próprio livro: são 45
// textos distintos nas duas tabelas, e todos caem numa destas formas.
var bookAttribute = map[string]string{"For": "strength", "Des": "dexterity", "Con": "constitution",
	"Int": "intelligence", "Sab": "wisdom", "Car": "charisma"}

var (
	attributeCell = regexp.MustCompile(`^(For|Des|Con|Int|Sab|Car)` + sp + `+(` + digit + `+)$`)
	levelCell     = regexp.MustCompile(`^(` + digit + `+)º nível de personagem$`)
	// `poderes?` exigiria "podere": o `?` cobre só o `s`. A coluna escreve "Um
	// poder da Tormenta" no singular, e o grupo é o que a torna opcional.
	tormentaCell = regexp.MustCompile(`^(Um|Dois|Três|Quatro|Cinco)` + sp + `+poder(?:es)?` + sp + `+da` + sp + `+Tormenta$`)
	trainedCell  = regexp.MustCompile(`(?i)^treinad[oa]` + sp + `+(?:em` + sp + `+|n[ao]` + sp + `+)?(.+)$`)
	// O ponto final pode ser o ÚLTIMO caractere do verbete — exigir ponto MAIS
	// espaço faz o último verbete de cada trecho sair sem pré-requisito, e o
	// relatório acusa o livro de discordar de si mesmo em 56 poderes. O Python
	// dizia `(?=\s|$)`; o RE2 não tem lookahead, e consumir o espaço não muda nem
	// onde o casamento começa nem o grupo 1, que são as duas coisas lidas.
	entryPrereq = regexp.MustCompile(`Pré-?` + sp + `*-?requisitos?:` + sp + `*(.+?)\.(?:` + sp + `|$)`)
)

var towerCount = map[string]int{"Um": 1, "Dois": 2, "Três": 3, "Quatro": 4, "Cinco": 5}

const noPrereq = "—"

// onePrereq é um pedaço da coluna na forma normalizada que o catálogo também
// produz.
func onePrereq(piece string, expertises, powers stringSet) string {
	piece = bookpdf.Strip(piece)
	if piece == "" || piece == noPrereq {
		return ""
	}
	if m := attributeCell.FindStringSubmatch(piece); m != nil {
		return "attr:" + bookAttribute[m[1]] + ":" + m[2]
	}
	if m := levelCell.FindStringSubmatch(piece); m != nil {
		return "level:" + m[1]
	}
	if m := tormentaCell.FindStringSubmatch(piece); m != nil {
		return "tormenta:" + itoa(towerCount[m[1]])
	}
	if m := trainedCell.FindStringSubmatch(piece); m != nil {
		piece = bookpdf.Strip(m[1])
	}
	if expertises[bookpdf.Key(piece)] {
		return "trained:" + bookpdf.Key(piece)
	}
	// A célula pode trazer o NOME de um poder em vez do travessão: a "Carga de
	// Cavalaria" está indentada sob "Ginete" e a célula dela diz "Ginete".
	if powers[bookpdf.Key(piece)] {
		return "power:" + bookpdf.Key(piece)
	}
	return "nota:" + bookpdf.NormalizePhrase(piece)
}

// readPrereqs lê a coluna inteira, que pode trazer vários separados por
// vírgula. A vírgula NÃO separa dentro de parênteses: "Ofício (alquimista)" é
// um requisito só. `parent` vazio é "sem pai".
func readPrereqs(text string, expertises, powers stringSet, parent string) stringSet {
	var pieces []string
	depth := 0
	var current strings.Builder
	for _, c := range text {
		switch c {
		case '(':
			depth++
		case ')':
			depth--
		}
		if c == ',' && depth == 0 {
			pieces = append(pieces, current.String())
			current.Reset()
			continue
		}
		current.WriteRune(c)
	}
	pieces = append(pieces, current.String())
	read := stringSet{}
	for _, p := range pieces {
		read[onePrereq(p, expertises, powers)] = true
	}
	// A INDENTAÇÃO implica o pai SEMPRE, e não só quando a célula traz `—`: o
	// travessão diz "nada além do pai"; um texto diz "isto E o pai" — o
	// "Fanático" está sob "Encouraçado" com "12º nível de personagem".
	if parent != "" {
		read["power:"+bookpdf.Key(parent)] = true
	}
	delete(read, "")
	return read
}

// shapeOnly são os pré-requisitos que têm FORMA, sem as notas em prosa. A
// tabela e o verbete dizem a mesma coisa com palavras diferentes ("Armaduras
// pesadas" × "Proficiência com armaduras pesadas"), e comparar a prosa daria
// uma discordância por redação a cada dois poderes. A nota sai para o OLHO.
func shapeOnly(pre stringSet) stringSet {
	out := stringSet{}
	for r := range pre {
		if !strings.HasPrefix(r, "nota:") {
			out[r] = true
		}
	}
	return out
}

var (
	number = regexp.MustCompile(`[+\-–−]?` + digit + `+(?:\.` + digit + `{3})*`)
	// A REMISSÃO DE PÁGINA não é número de regra, e aparece nas DUAS formas: o
	// livro escreve "veja a página 260" e o catálogo "ver p260". Filtrar só uma
	// deixa a assimetria acusando um verbete cuja regra está certa.
	pageReference = regexp.MustCompile(`p(?:[áa]g(?:ina|s?\.)?)?` + sp + `*` + digit + `+(?:` + sp + `+e` + sp + `+` + digit + `+)?`)
	// wordRun é o `\w+` do Python. As palavras por extenso são só letras, então
	// `\b(um|uma|…)\b` é exatamente "uma corrida de letras inteira que é uma
	// delas" — e é assim que se lê sem o `\b` Unicode que o RE2 não tem.
	wordRun = regexp.MustCompile(`[\p{L}\p{N}_]+`)
)

// spelledOut: o LIVRO ESCREVE O NUMERAL POR EXTENSO e o catálogo em dígito
// ("uma ação de movimento extra" × "1 ação"). Sem traduzir, cinco poderes
// certos aparecem como número inventado.
var spelledOut = map[string]string{"um": "1", "uma": "1", "dois": "2", "duas": "2", "três": "3",
	"quatro": "4", "cinco": "5", "seis": "6", "sete": "7", "oito": "8", "nove": "9", "dez": "10"}

var dashes = strings.NewReplacer("–", "-", "−", "-")

// numbers são os números que a frase imprime, sem as remissões de página.
func numbers(text string) stringSet {
	clean := pageReference.ReplaceAllString(text, " ")
	clean = wordRun.ReplaceAllStringFunc(clean, func(w string) string {
		if d, ok := spelledOut[strings.ToLower(w)]; ok {
			return d
		}
		return w
	})
	out := stringSet{}
	for _, n := range number.FindAllString(clean, -1) {
		out[strings.TrimLeft(dashes.Replace(n), "+")] = true
	}
	return out
}

// isWord é o `\w` do Python para um caractere.
func isWord(r rune) bool { return wordRun.MatchString(string(r)) }

// citedExpertises são as perícias nomeadas no texto, com a borda de palavra
// Unicode do `\b` do Python: há borda onde um lado é palavra e o outro não.
func citedExpertises(text string, names []string) stringSet {
	out := stringSet{}
	for _, n := range names {
		if n != "" && containsWholeWord(text, n) {
			out[n] = true
		}
	}
	return out
}

func containsWholeWord(text, needle string) bool {
	runes := []rune(text)
	want := []rune(needle)
	boundary := func(i int) bool {
		before := i > 0 && isWord(runes[i-1])
		after := i < len(runes) && isWord(runes[i])
		return before != after
	}
	for i := 0; i+len(want) <= len(runes); i++ {
		if string(runes[i:i+len(want)]) == needle && boundary(i) && boundary(i+len(want)) {
			return true
		}
	}
	return false
}

// entryRule é o texto do verbete sem o título e sem a linha de pré-requisito —
// só a regra, que é o que o catálogo descreve.
func entryRule(body, name string) string {
	at := strings.Index(body, name)
	if at < 0 {
		return ""
	}
	rest := body[at+len(name):]
	if rest == "" {
		return ""
	}
	if loc := entryPrereq.FindStringIndex(rest); loc != nil {
		rest = rest[:loc[0]]
	}
	return bookpdf.Strip(rest)
}
