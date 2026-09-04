package sheetui

import (
	"fmt"
	"strings"
	"t20engine/book"
	"t20engine/sheet"
)

// AS PROFICIÊNCIAS como dado (ALE-272, fatia 2).
//
// A menor das sete abas, e a primeira que traz REGRA para o servidor: até aqui a
// tabela de "que classe sabe usar o quê" só existia em TypeScript
// (`frontend/src/shared/rules/proficiencies.ts`), fora do motor. Ela é dado
// transcrito do livro, então foi para onde o dado transcrito mora — o campo
// `proficiencies` de `catalog/data/classes.json`, ao lado da página — e o que
// sobra aqui é só a RESOLUÇÃO: juntar as classes do personagem, aplicar as duas
// linhas de base e dizer quais categorias ele tem.
//
// Ser proficiente ou não é o que a Mochila lê para avisar "sem proficiência"
// (p142: −5 no ataque; a penalidade da armadura nas perícias de Força e
// Destreza) — esta aba é onde esse aviso se resolve.

// As sete categorias moram no `book` desde a ALE-278 (`ProficiencyCategories`).
// A lista estava aqui, e mais duas vezes no `api` — ver o comentário de lá, que
// conta como as três transcrições conviveram sem ninguém notar.

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
	Chave  string
	Rotulo string
	// Tem é o estado GUARDADO — o que o personagem de fato tem, depois de
	// qualquer ajuste manual.
	Tem bool
	// DeClasse é o que as classes concedem por si. Ela existe separada de `Tem`
	// para o jogador distinguir um ajuste deliberado do que veio de fábrica: sem
	// isso, uma proficiência tirada na mão parece defeito da conta.
	DeClasse bool
	// Fontes são as classes que concedem, para a etiqueta explicar de onde vem.
	Fontes []string
}

// proficiencyGroup é um dos dois blocos do painel.
type proficiencyGroup struct {
	Titulo string
	Linhas []sheetProficiency
}

// proficiencyGroupsOf monta as sete linhas nos dois grupos.
func proficiencyGroupsOf(dto sheet.CharacterDTO) []proficiencyGroup {
	tem := savedProficiencies(dto.Proficiencies)
	fontes := proficiencySources(dto)

	porGrupo := map[string]*proficiencyGroup{}
	ordem := []string{book.WeaponGroup, book.ArmorGroup}
	for _, titulo := range ordem {
		porGrupo[titulo] = &proficiencyGroup{Titulo: titulo}
	}
	for _, cat := range book.ProficiencyCategories {
		g := porGrupo[cat.Group]
		g.Linhas = append(g.Linhas, sheetProficiency{
			Chave:    cat.Key,
			Rotulo:   cat.Label,
			Tem:      tem[cat.Key],
			DeClasse: len(fontes[cat.Key]) > 0,
			Fontes:   fontes[cat.Key],
		})
	}
	return []proficiencyGroup{*porGrupo[book.WeaponGroup], *porGrupo[book.ArmorGroup]}
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
//     escreve** — conferi a p148, que define as duas categorias e a penalidade
//     por não proficiência, e não há linha dizendo que uma implica a outra. É
//     decisão de produto herdada da SPA, e ela é a que NÃO machuca: sem ela,
//     "restaurar o padrão de classe" tiraria a armadura leve de um guerreiro, e o
//     motor passaria a aplicar a penalidade da p148 num personagem treinado em
//     algo mais pesado. Está aqui explícita para poder ser revista de propósito,
//     e não redescoberta como defeito.
func proficiencySources(dto sheet.CharacterDTO) map[string][]string {
	daClasse := book.ProficienciesByClass()
	fontes := map[string][]string{everyoneStartsWith: {everyoneSourceLabel}}
	for _, cl := range dto.Classes {
		concede := daClasse[cl.ClassName]
		for _, chave := range concede {
			fontes[chave] = append(fontes[chave], cl.ClassName)
			if chave == "armaduras-pesadas" {
				fontes["armaduras-leves"] = append(fontes["armaduras-leves"], cl.ClassName)
			}
		}
	}
	for chave := range fontes {
		fontes[chave] = repetirSem(fontes[chave])
	}
	return fontes
}

// repetirSem tira o nome duplicado de quem tem a mesma classe duas vezes —
// impossível hoje, e a etiqueta "Padrão: Guerreiro, Guerreiro" seria o sintoma.
func repetirSem(nomes []string) []string {
	visto := map[string]bool{}
	unicos := nomes[:0]
	for _, nome := range nomes {
		if visto[nome] {
			continue
		}
		visto[nome] = true
		unicos = append(unicos, nome)
	}
	return unicos
}

// classDefault é o alvo do "Restaurar padrão de classe": tudo o que as
// classes concedem, e nada do que foi acrescentado na mão.
func classDefault(dto sheet.CharacterDTO) []string {
	fontes := proficiencySources(dto)
	padrao := make([]string, 0, len(fontes))
	for _, cat := range book.ProficiencyCategories {
		if len(fontes[cat.Key]) > 0 {
			padrao = append(padrao, cat.Key)
		}
	}
	return padrao
}

// proficiencySwap devolve o conjunto DEPOIS de ligar ou desligar uma.
//
// A saída sai na ordem do catálogo e não na de chegada: o blob é lido por
// pessoa numa revisão de banco, e uma ordem estável faz o diff de duas gravações
// significar alguma coisa.
func proficiencySwap(dto sheet.CharacterDTO, chave string) ([]string, error) {
	if !book.IsProficiencyCategory(chave) {
		return nil, fmt.Errorf("proficiência %q não existe: são %s",
			chave, strings.Join(book.ProficiencyKeys(), ", "))
	}
	tem := savedProficiencies(dto.Proficiencies)
	tem[chave] = !tem[chave]
	depois := make([]string, 0, len(tem))
	for _, cat := range book.ProficiencyCategories {
		if tem[cat.Key] {
			depois = append(depois, cat.Key)
		}
	}
	return depois, nil
}

// A lista que a mensagem de erro cita é a do livro (`book.ProficiencyKeys`).
// Aqui ela era ordenada em ordem ALFABÉTICA "para a frase não mudar entre duas
// execuções" — o que resolvia a instabilidade de percorrer um `map` e trocava a
// escala de dificuldade da p142 por uma ordem que não diz nada.

// sourceTag é o `title` da etiqueta "classe": "Padrão: Guerreiro, Nobre".
func sourceTag(linha sheetProficiency) string {
	return "Padrão: " + strings.Join(linha.Fontes, ", ")
}

// swapLabel é o nome acessível do botão de cada linha.
//
// O VERBO diz o que o clique FAZ, e não o estado atual: um leitor de tela lê o
// botão para decidir se aperta, e "Armas marciais" sozinho não diz se apertar
// dá ou tira.
func swapLabel(linha sheetProficiency) string {
	if linha.Tem {
		return "Remover proficiência: " + linha.Rotulo
	}
	return "Adicionar proficiência: " + linha.Rotulo
}
