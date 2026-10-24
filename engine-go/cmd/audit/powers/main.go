// Command powers confere os poderes GERAIS e os da TORMENTA contra a seção
// "Poderes" (p124-137). Seção 6 da ALE-391.
//
// São 90 verbetes, e desde a ALE-401 eles são a FONTE ÚNICA: os benefícios de
// origem e de classe APONTAM para cá em vez de copiar a regra. A faixa p124-137
// guarda quatro famílias e só duas são daqui; os Poderes Concedidos têm issue
// própria, a ALE-408.
//
// O livro declara os PRÉ-REQUISITOS duas vezes — na tabela de cada grupo e na
// linha `Pré-requisito:` do verbete —, e os dois instrumentos se conferem um ao
// outro ANTES de encostar no catálogo. Por que a segunda âncora existe, e por
// que a tabela se lê primeiro retirando as células dela da busca por verbete, é
// do `engine-go/CLAUDE.md`, na seção dos auditores.
//
// O denominador: por poder, se ancorou nas duas declarações, numa só, ou em
// nenhuma; e o total de PRÉ-REQUISITOS comparados, porque um poder que ancorou
// e cujo pré-requisito não foi lido sai verde sobre nada.
//
// Ele PROPÕE: nada é escrito, e cada correção se revisa contra a página citada.
//
//	go run ./cmd/audit/powers
package main

import (
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"t20engine/cmd/internal/pyfmt"

	"t20engine/domain/catalog/transcript"
	"t20engine/infra/bookpdf"
)

// minimumCoverage mede AO CONTRÁRIO da que nasceu nas origens: aqui o catálogo
// RESUME um parágrafo inteiro, e medir livro→catálogo dá 20% para a Torcida,
// cuja regra está perfeita. A pergunta para um resumo é a inversa — tudo o que
// o catálogo AFIRMA está no livro? O piso é frouxo de propósito: ele ORDENA a
// lista para a revisão humana em vez de decidir sozinho.
const minimumCoverage = 0.5

type declared struct{ name, reason string }

// accepted é o que fica ACEITO, e cada linha diz por quê. Declarar faz um caso
// NOVO reprovar com o NOME dele; uma linha que PARAR de valer também reprova.
var accepted = []declared{
	{"Foco em Arma", "o catálogo diz \"pego várias vezes\" onde o livro diz \"escolher este poder outras vezes\" (p128) — sinônimo, e o +2 e o escopo batem"},
	{"Foco em Perícia", "o catálogo escreve \"2d20\" e a p130 escreve \"dois dados\". Em T20 teste de perícia É d20, então o catálogo especifica o que a página deixa implícito"},
	{"Inventário Organizado", "o catálogo escreve \"1/2\" e a p130 escreve \"meio espaço\" — o mesmo número por extenso, que a tradução de numeral não alcança por ser fração"},
	{"Ao Sabor do Destino", "a progressão por patamar está na tabela da página SEGUINTE (p130), fora do verbete — o catálogo a transcreve e o auditor não a lê. NÃO MEDIDO, não aprovado"},
	{"Disparo Preciso", "o catálogo aceita \"Estilo de Disparo OU Estilo de Arremesso\" e o VERBETE da p125 diz exatamente isso; quem simplifica é a TABELA da p126, que a indentação obriga a um pai só"},
}

// selfDisagreement é ONDE O LIVRO DISCORDA DE SI MESMO. Não é defeito nosso, e
// o catálogo segue o VERBETE — que é onde a regra é enunciada por extenso.
var selfDisagreement = []declared{
	{"Estilo de Arma e Escudo", "a tabela da p126 não lista \"treinado em Luta\" e o verbete da p125 exige"},
	{"Disparo Preciso", "a tabela da p126 lista só \"Estilo de Disparo\" e o verbete da p125 aceita \"ou Estilo de Arremesso\""},
}

func isDeclared(list []declared, name string) bool {
	return slices.ContainsFunc(list, func(d declared) bool { return d.name == name })
}

// power é o que o auditor lê de um poder geral ou da Tormenta.
type power struct {
	name, description string
	bookPage          int
	tormenta          bool
	catalogPrereqs    stringSet
}

// catalogPrereqs são os pré-requisitos do catálogo na MESMA forma normalizada.
func catalogPrereqs(minLevel, otherPowers int, requiresPower string, rules []transcript.Prerequisite,
	nameByID map[string]string) stringSet {
	out := stringSet{}
	if minLevel != 0 {
		out["level:"+itoa(minLevel)] = true
	}
	if otherPowers != 0 {
		out["tormenta:"+itoa(otherPowers)] = true
	}
	if requiresPower != "" {
		out["power:"+bookpdf.Key(nameByID[requiresPower])] = true
	}
	for _, r := range rules {
		switch r.Kind {
		case "attribute":
			out["attr:"+r.Attr+":"+itoa(r.Min)] = true
		case "trained":
			out["trained:"+bookpdf.Key(r.Expertise)] = true
		case "power":
			out["power:"+bookpdf.Key(nameByID[r.ID])] = true
		case "anyPower":
			keys := make([]string, len(r.IDs))
			for i, id := range r.IDs {
				keys[i] = bookpdf.Key(nameByID[id])
			}
			sort.Strings(keys)
			out["power:"+strings.Join(keys, "|")] = true
		default:
			out["nota:"+bookpdf.NormalizePhrase(r.Description)] = true
		}
	}
	return out
}

func loadPowers() []power {
	nameByID := map[string]string{}
	for _, p := range transcript.GeneralPowers {
		nameByID[p.ID] = p.Name
	}
	for _, p := range transcript.TormentaPowers {
		nameByID[p.ID] = p.Name
	}
	var out []power
	for _, p := range transcript.GeneralPowers {
		out = append(out, power{p.Name, p.Description, p.BookPage, false,
			catalogPrereqs(p.MinLevel, 0, "", p.Prerequisites, nameByID)})
	}
	for _, p := range transcript.TormentaPowers {
		out = append(out, power{p.Name, p.Description, p.BookPage, true,
			catalogPrereqs(0, p.RequiresOtherPowers, p.RequiresPower, nil, nameByID)})
	}
	return out
}

func main() {
	book, err := bookpdf.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	all := loadPowers()
	var expertiseNames []string
	expertises := stringSet{}
	for _, e := range transcript.Expertises {
		expertiseNames = append(expertiseNames, e.Name)
		expertises[bookpdf.Key(e.Name)] = true
	}
	powerKeys := stringSet{}
	var names []string
	for _, p := range all {
		powerKeys[bookpdf.Key(p.name)] = true
		names = append(names, p.name)
	}

	table := readTable(book)
	entries, seen := readEntries(book, names)

	m := &measured{}
	diverge, inBoth, inOne := 0, 0, 0
	var inNone, bookSelf, complaints, notes []string
	acceptedSeen := map[string]bool{}
	for _, p := range all {
		k := bookpdf.Key(p.name)
		var fromTable stringSet
		if row, ok := table[k]; ok {
			fromTable = readPrereqs(row.prereq, expertises, powerKeys, row.parent)
		}
		entry, haveEntry := entries[k]
		body := entry.body
		var fromEntry stringSet
		switch {
		case p.tormenta:
			// O VERBETE DA TORMENTA NÃO DECLARA PRÉ-REQUISITO: a seção os imprime
			// só na tabela da p128. Ler a ausência como "nenhum" faria os 22
			// discordarem da tabela, e a discordância seria do instrumento.
		case body != "" && entryPrereq.MatchString(body):
			fromEntry = readPrereqs(entryPrereq.FindStringSubmatch(body)[1], expertises, powerKeys, "")
		case body != "":
			fromEntry = stringSet{}
		}
		if fromTable == nil && fromEntry == nil {
			inNone = append(inNone, p.name+": não ancorou em nenhuma das duas declarações — NADA dele foi medido")
			continue
		}
		if fromTable == nil || fromEntry == nil {
			inOne++
			where := "no verbete"
			if fromTable == nil {
				where = "na tabela"
			}
			complaints = append(complaints, p.name+": não ancorou "+where)
		} else {
			inBoth++
		}
		var page *int
		if haveEntry {
			page = &entry.page
		}
		found, disagree, prose := check(p, fromTable, fromEntry, page, entryRule(body, p.name), expertiseNames, m)
		bookSelf = append(bookSelf, disagree...)
		notes = append(notes, prose...)
		if isDeclared(accepted, p.name) {
			acceptedSeen[p.name] = true
			continue
		}
		for _, c := range found {
			if strings.HasPrefix(c, "  ") {
				fmt.Printf("  %s\n", c)
				continue
			}
			fmt.Printf("  %s: %s\n", p.name, c)
			diverge++
		}
	}

	if len(bookSelf) > 0 {
		fmt.Println("\nO LIVRO DISCORDA DE SI MESMO (achado sobre o LIVRO):")
		for _, l := range bookSelf {
			fmt.Printf("  %s\n", l)
		}
	}
	if len(notes) > 0 {
		fmt.Printf("\nPRÉ-REQUISITO QUE SÓ DIFERE NA REDAÇÃO (%d, para o OLHO — a forma bate):\n", len(notes))
		for _, l := range notes {
			fmt.Printf("  %s\n", l)
		}
	}
	for _, l := range append(inNone, complaints...) {
		fmt.Printf("  NÃO MEDIDO %s\n", l)
	}
	var repeated []countEntry
	for _, s := range seen {
		if s.n > 1 {
			repeated = append(repeated, s)
		}
	}
	if len(repeated) > 0 {
		fmt.Printf("  NÃO MEDIDO títulos achados mais de uma vez: %s\n", pyCountDict(repeated))
	}

	for _, a := range accepted {
		fmt.Printf("  ACEITO %s: %s\n", a.name, a.reason)
		if !acceptedSeen[a.name] {
			fmt.Printf("  PROBLEMA %s está em ACEITO e não diverge mais — tire a linha\n", a.name)
			diverge++
		}
	}
	for _, d := range selfDisagreement {
		fmt.Printf("  DISCORDÂNCIA DECLARADA %s: %s\n", d.name, d.reason)
	}

	fmt.Printf("\npoderes: %d | ancoraram nas DUAS declarações: %d | em UMA só: %d | em NENHUMA (≠ corretos): %d\n",
		len(all), inBoth, inOne, len(inNone))
	fmt.Printf("lidos na tabela: %d/%d | verbetes achados: %d/%d\n", len(table), len(all), len(entries), len(all))
	fmt.Printf("COMPARADOS: pre %d, pagina %d, regra %d\n", m.prereqs, m.pages, m.rules)
	fmt.Printf("cobertura da regra, da pior para a melhor (%d medidas, piso %s):\n",
		len(m.coverage), pyfmt.Percent(minimumCoverage))
	sort.Slice(m.coverage, func(i, j int) bool {
		if m.coverage[i].ratio != m.coverage[j].ratio {
			return m.coverage[i].ratio < m.coverage[j].ratio
		}
		return m.coverage[i].name < m.coverage[j].name
	})
	for i, c := range m.coverage {
		if i == 14 {
			break
		}
		fmt.Printf("  %5s %s\n", pyfmt.Percent(c.ratio), c.name)
	}
	fmt.Printf("divergem do livro: %d | o livro discorda de si: %d\n", diverge, len(bookSelf))
	if diverge > 0 || len(bookSelf) > 0 || len(inNone) > 0 {
		os.Exit(1)
	}
}
