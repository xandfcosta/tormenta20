package sheetui

import (
	"fmt"
	"strings"
	"t20engine/domain/book"
	"t20engine/domain/sheet"
)

// AS PROFICIÊNCIAS como dado.
//
// A tabela de "que classe sabe usar o quê" é dado transcrito do livro, então
// mora onde o dado transcrito mora: o campo `proficiencies` de
// `transcript.Classes`, ao lado da página. O que sobra aqui é só a
// RESOLUÇÃO — juntar as classes e as raças do personagem (o Vanguardista do
// kliren dá armas de fogo, ALE-493), aplicar as duas linhas de base e dizer
// quais categorias ele tem.
//
// Ser proficiente ou não é o que a Mochila lê para avisar "sem proficiência"
// (p142: −5 no ataque; a penalidade da armadura nas perícias de Força e
// Destreza) — esta aba é onde esse aviso se resolve.

// As sete categorias moram no `book` (`ProficiencyCategories`), e não aqui: três
// transcrições da mesma lista já conviveram sem ninguém notar.

// everyoneStartsWith é a proficiência que ninguém precisa ganhar.
//
// p142: *"Armas Simples. […] Todos os personagens sabem usar armas simples."*
// Ela aparece na lista mesmo assim, e marcada, porque a lista é a resposta à
// pergunta "com o que eu sei lutar?" — esconder a resposta mais comum obrigaria
// o jogador a saber a regra de cor.
const everyoneStartsWith = "armas-simples"

// everyoneSourceLabel é o que a etiqueta diz quando a proficiência não vem de
// classe nenhuma.
const everyoneSourceLabel = "Todas as classes"

// sheetProficiency é uma linha do painel.
type sheetProficiency struct {
	Key   string
	Label string
	// Has é o estado GUARDADO — o que o personagem de fato tem, depois de
	// qualquer ajuste manual.
	Has bool
	// FromClass é o que as classes concedem por si. Ela existe separada de `Has`
	// para o jogador distinguir um ajuste deliberado do que veio de fábrica: sem
	// isso, uma proficiência tirada na mão parece defeito da conta.
	FromClass bool
	// FromRace marca a concessão que é SÓ da raça — o Vanguardista do kliren
	// (ALE-493) —, para o selo dizer "raça" e não "classe".
	FromRace bool
	// FromPower marca a concessão que é SÓ do poder Proficiência (p129,
	// ALE-510), para o selo dizer "poder".
	FromPower bool
	// Sources são as classes e raças que concedem, para a etiqueta explicar de
	// onde vem.
	Sources []string
}

// Badge é o selo de fábrica: "classe", ou "raça" quando só a raça concede.
func (p sheetProficiency) Badge() string {
	if p.FromRace {
		return "raça"
	}
	if p.FromPower {
		return "poder"
	}
	return "classe"
}

// proficiencyGroup é um dos dois blocos do painel.
type proficiencyGroup struct {
	Title string
	Rows  []sheetProficiency
}

// proficiencyGroupsOf monta as sete linhas nos dois grupos.
func proficiencyGroupsOf(dto sheet.CharacterDTO) []proficiencyGroup {
	found := savedProficiencies(dto.Proficiencies)
	sources := proficiencySources(dto)
	byRace := raceProficiencySources(dto)
	byPower := powerProficiencySources(dto)

	byGroup := map[string]*proficiencyGroup{}
	order := []string{book.WeaponGroup, book.ArmorGroup}
	for _, title := range order {
		byGroup[title] = &proficiencyGroup{Title: title}
	}
	for _, cat := range book.ProficiencyCategories {
		g := byGroup[cat.Group]
		g.Rows = append(g.Rows, sheetProficiency{
			Key:       cat.Key,
			Label:     cat.Label,
			Has:       found[cat.Key],
			FromClass: len(sources[cat.Key]) > 0,
			FromRace:  len(sources[cat.Key]) == 0 && len(byRace[cat.Key]) > 0,
			FromPower: len(sources[cat.Key]) == 0 && len(byRace[cat.Key]) == 0 && len(byPower[cat.Key]) > 0,
			Sources:   grantersOf(dto, cat.Key),
		})
	}
	return []proficiencyGroup{*byGroup[book.WeaponGroup], *byGroup[book.ArmorGroup]}
}

// savedProficiencies lê o blob da coluna `proficiencies`.
//
// Blob CORROMPIDO vira conjunto VAZIO, e não erro: a coluna é um `string[]` cru
// e a ficha inteira não pode deixar de abrir porque uma linha do banco está
// torta. Sem proficiência nenhuma é um estado legítimo (um arcanista de nível 1
// chega perto disso), então a degradação é para um estado que a tela sabe
// desenhar.
func savedProficiencies(blob string) map[string]bool {
	return sheet.ToStringSet(sheet.UnmarshalStrings(blob))
}

// proficiencySources diz, por categoria, QUAIS classes a concedem.
//
// # As duas linhas de base
//
//  1. `armas-simples` é de todo mundo (p142), e por isso a fonte dela é a frase
//     "Todas as classes" em vez de um nome de classe.
//
//  2. Quem sabe usar armadura PESADA sabe usar a LEVE. **Isso o livro não
//     escreve** — a p148 define as duas categorias e a penalidade por não
//     proficiência, e não há linha dizendo que uma implica a outra. É decisão de
//     produto, e é a que NÃO machuca: sem ela, "restaurar o padrão de classe"
//     tiraria a armadura leve de um guerreiro, e o motor passaria a aplicar a
//     penalidade da p148 num personagem treinado em algo mais pesado. Está aqui
//     explícita para poder ser revista de propósito, e não redescoberta como
//     defeito.
func proficiencySources(dto sheet.CharacterDTO) map[string][]string {
	ofClass := book.ProficienciesByClass()
	sources := map[string][]string{everyoneStartsWith: {everyoneSourceLabel}}
	for _, cl := range dto.Classes {
		concede := ofClass[cl.ClassName]
		for _, key := range concede {
			sources[key] = append(sources[key], cl.ClassName)
			if key == "armaduras-pesadas" {
				sources["armaduras-leves"] = append(sources["armaduras-leves"], cl.ClassName)
			}
		}
	}
	for key := range sources {
		sources[key] = repetirSem(sources[key])
	}
	return sources
}

// repetirSem tira o nome duplicado de quem tem a mesma classe duas vezes —
// impossível hoje, e a etiqueta "Padrão: Guerreiro, Guerreiro" seria o sintoma.
func repetirSem(names []string) []string {
	seen := map[string]bool{}
	unique := names[:0]
	for _, name := range names {
		if seen[name] {
			continue
		}
		seen[name] = true
		unique = append(unique, name)
	}
	return unique
}

// raceProficiencySources diz, por categoria, QUAIS raças a concedem.
func raceProficiencySources(dto sheet.CharacterDTO) map[string][]string {
	out := map[string][]string{}
	for _, r := range dto.Races {
		for _, key := range book.RaceProficiencies(r.Race) {
			out[key] = append(out[key], book.RaceLabel(r.Race))
		}
	}
	return out
}

// powerProficiencySources diz, por categoria, se o poder Proficiência a
// concede (p129, ALE-510).
func powerProficiencySources(dto sheet.CharacterDTO) map[string][]string {
	out := map[string][]string{}
	for _, key := range sheet.PowerProficiencies(dto.PowerChoices) {
		out[key] = []string{"Proficiência"}
	}
	return out
}

// grantersOf são as ORIGENS da categoria — classes, raças e o poder —, na
// ordem em que a etiqueta as diz.
func grantersOf(dto sheet.CharacterDTO, key string) []string {
	out := append([]string{}, proficiencySources(dto)[key]...)
	out = append(out, raceProficiencySources(dto)[key]...)
	return append(out, powerProficiencySources(dto)[key]...)
}

// classDefault é o alvo do "Restaurar padrão": tudo o que as classes, as
// raças e o poder Proficiência concedem, e nada do que foi acrescentado na mão.
func classDefault(dto sheet.CharacterDTO) []string {
	standard := []string{}
	for _, cat := range book.ProficiencyCategories {
		if len(grantersOf(dto, cat.Key)) > 0 {
			standard = append(standard, cat.Key)
		}
	}
	return standard
}

// proficiencySwap devolve o conjunto DEPOIS de ligar ou desligar uma.
//
// A saída sai na ordem do catálogo e não na de chegada: o blob é lido por
// pessoa numa revisão de banco, e uma ordem estável faz o diff de duas gravações
// significar alguma coisa.
func proficiencySwap(dto sheet.CharacterDTO, key string) ([]string, error) {
	if !book.IsProficiencyCategory(key) {
		return nil, fmt.Errorf("proficiência %q não existe: são %s",
			key, strings.Join(book.ProficiencyKeys(), ", "))
	}
	found := savedProficiencies(dto.Proficiencies)
	found[key] = !found[key]
	after := make([]string, 0, len(found))
	for _, cat := range book.ProficiencyCategories {
		if found[cat.Key] {
			after = append(after, cat.Key)
		}
	}
	return after, nil
}

// A lista que a mensagem de erro cita é a do livro (`book.ProficiencyKeys`), na
// ordem DELE e não em ordem alfabética: alfabética também seria estável, e
// trocaria a escala de dificuldade da p142 por uma ordem que não diz nada.

// sourceTag é o `title` da etiqueta "classe": "Padrão: Guerreiro, Nobre".
func sourceTag(row sheetProficiency) string {
	return "Padrão: " + strings.Join(row.Sources, ", ")
}

// swapLabel é o nome acessível do botão de cada linha.
//
// O VERBO diz o que o clique FAZ, e não o estado atual: um leitor de tela lê o
// botão para decidir se aperta, e "Armas marciais" sozinho não diz se apertar
// dá ou tira.
func swapLabel(row sheetProficiency) string {
	if row.Has {
		return "Remover proficiência: " + row.Label
	}
	return "Adicionar proficiência: " + row.Label
}
