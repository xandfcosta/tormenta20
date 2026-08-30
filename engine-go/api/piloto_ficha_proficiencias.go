package api

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
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

// asCategoriasDeProficiencia são as sete, na ordem em que a tela as mostra.
//
// A ORDEM É A DO LIVRO dentro de cada grupo (p142 para as armas, p148 para as
// armaduras), e não alfabética: "simples, marciais, exóticas, de fogo" é uma
// escala de dificuldade, e ordenar por nome a embaralharia.
var asCategoriasDeProficiencia = []categoriaDeProficiencia{
	{Chave: "armas-simples", Rotulo: "Armas simples", Grupo: grupoDasArmas},
	{Chave: "armas-marciais", Rotulo: "Armas marciais", Grupo: grupoDasArmas},
	{Chave: "armas-exoticas", Rotulo: "Armas exóticas", Grupo: grupoDasArmas},
	{Chave: "armas-de-fogo", Rotulo: "Armas de fogo", Grupo: grupoDasArmas},
	{Chave: "armaduras-leves", Rotulo: "Armaduras leves", Grupo: grupoDasArmaduras},
	{Chave: "armaduras-pesadas", Rotulo: "Armaduras pesadas", Grupo: grupoDasArmaduras},
	{Chave: "escudos", Rotulo: "Escudos", Grupo: grupoDasArmaduras},
}

const (
	grupoDasArmas     = "Armas"
	grupoDasArmaduras = "Armaduras & Escudos"
)

// aBaseDeTodoMundo é a proficiência que ninguém precisa ganhar.
//
// p142: *"Armas Simples. […] Todos os personagens sabem usar armas simples."*
// Ela aparece na lista mesmo assim, e marcada, porque a lista é a resposta à
// pergunta "com o que eu sei lutar?" — esconder a resposta mais comum obrigaria
// o jogador a saber a regra de cor.
const aBaseDeTodoMundo = "armas-simples"

// aFonteDeTodoMundo é o que a etiqueta diz quando a proficiência não vem de
// classe nenhuma.
const aFonteDeTodoMundo = "Todas as classes"

type categoriaDeProficiencia struct {
	Chave  string
	Rotulo string
	Grupo  string
}

// proficienciaDaFicha é uma linha do painel.
type proficienciaDaFicha struct {
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

// grupoDeProficiencias é um dos dois blocos do painel.
type grupoDeProficiencias struct {
	Titulo string
	Linhas []proficienciaDaFicha
}

// oPainelDeProficiencias monta as sete linhas nos dois grupos.
func oPainelDeProficiencias(dto CharacterDTO) []grupoDeProficiencias {
	tem := asProficienciasGuardadas(dto.Proficiencies)
	fontes := asFontesDeProficiencia(dto)

	porGrupo := map[string]*grupoDeProficiencias{}
	ordem := []string{grupoDasArmas, grupoDasArmaduras}
	for _, titulo := range ordem {
		porGrupo[titulo] = &grupoDeProficiencias{Titulo: titulo}
	}
	for _, cat := range asCategoriasDeProficiencia {
		g := porGrupo[cat.Grupo]
		g.Linhas = append(g.Linhas, proficienciaDaFicha{
			Chave:    cat.Chave,
			Rotulo:   cat.Rotulo,
			Tem:      tem[cat.Chave],
			DeClasse: len(fontes[cat.Chave]) > 0,
			Fontes:   fontes[cat.Chave],
		})
	}
	return []grupoDeProficiencias{*porGrupo[grupoDasArmas], *porGrupo[grupoDasArmaduras]}
}

// asProficienciasGuardadas lê o blob da coluna `proficiencies`.
//
// Blob CORROMPIDO vira conjunto VAZIO, e não erro: a coluna é um `string[]` cru
// e a ficha inteira não pode deixar de abrir porque uma linha do banco está
// torta. Sem proficiência nenhuma é um estado legítimo (um arcanista de nível 1
// chega perto disso), então a degradação é para um estado que a tela sabe
// desenhar.
func asProficienciasGuardadas(blob string) map[string]bool {
	var lista []string
	if json.Unmarshal([]byte(blob), &lista) != nil {
		return map[string]bool{}
	}
	tem := make(map[string]bool, len(lista))
	for _, chave := range lista {
		tem[chave] = true
	}
	return tem
}

// asFontesDeProficiencia diz, por categoria, QUAIS classes a concedem.
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
func asFontesDeProficiencia(dto CharacterDTO) map[string][]string {
	daClasse := asProficienciasPorClasse()
	fontes := map[string][]string{aBaseDeTodoMundo: {aFonteDeTodoMundo}}
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
		fontes[chave] = semRepetir(fontes[chave])
	}
	return fontes
}

// semRepetir tira o nome duplicado de quem tem a mesma classe duas vezes —
// impossível hoje, e a etiqueta "Padrão: Guerreiro, Guerreiro" seria o sintoma.
func semRepetir(nomes []string) []string {
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

// asProficienciasPorClasse é a tabela do livro, lida do catálogo de classes.
//
// Ela sai de `catalog/data/classes.json` e não de um `map` escrito em Go pelo
// mesmo motivo das perícias de classe: é DADO TRANSCRITO — a linha
// "Proficiências." de cada classe, p36–83 — e dado transcrito mora no catálogo,
// onde a validação de schema o alcança.
func asProficienciasPorClasse() map[string][]string {
	_, classes, _ := catalogosDoPersonagem()
	tabela := make(map[string][]string, len(classes))
	for _, c := range classes {
		tabela[c.Name] = c.Proficiencias
	}
	return tabela
}

// oPadraoDaClasse é o alvo do "Restaurar padrão de classe": tudo o que as
// classes concedem, e nada do que foi acrescentado na mão.
func oPadraoDaClasse(dto CharacterDTO) []string {
	fontes := asFontesDeProficiencia(dto)
	padrao := make([]string, 0, len(fontes))
	for _, cat := range asCategoriasDeProficiencia {
		if len(fontes[cat.Chave]) > 0 {
			padrao = append(padrao, cat.Chave)
		}
	}
	return padrao
}

// aTrocaDaProficiencia devolve o conjunto DEPOIS de ligar ou desligar uma.
//
// A saída sai na ordem do catálogo e não na de chegada: o blob é lido por
// pessoa numa revisão de banco, e uma ordem estável faz o diff de duas gravações
// significar alguma coisa.
func aTrocaDaProficiencia(dto CharacterDTO, chave string) ([]string, error) {
	if !proficiencyCategories[chave] {
		return nil, fmt.Errorf("proficiência %q não existe: são %s",
			chave, strings.Join(asChavesDeProficiencia(), ", "))
	}
	tem := asProficienciasGuardadas(dto.Proficiencies)
	tem[chave] = !tem[chave]
	depois := make([]string, 0, len(tem))
	for _, cat := range asCategoriasDeProficiencia {
		if tem[cat.Chave] {
			depois = append(depois, cat.Chave)
		}
	}
	return depois, nil
}

// asChavesDeProficiencia é a lista para a mensagem de erro dizer o formato
// esperado, ordenada para a frase não mudar entre duas execuções.
func asChavesDeProficiencia() []string {
	chaves := make([]string, 0, len(proficiencyCategories))
	for chave := range proficiencyCategories {
		chaves = append(chaves, chave)
	}
	sort.Strings(chaves)
	return chaves
}

// aEtiquetaDaFonte é o `title` da etiqueta "classe": "Padrão: Guerreiro, Nobre".
func aEtiquetaDaFonte(linha proficienciaDaFicha) string {
	return "Padrão: " + strings.Join(linha.Fontes, ", ")
}

// oRotuloDaTroca é o nome acessível do botão de cada linha.
//
// O VERBO diz o que o clique FAZ, e não o estado atual: um leitor de tela lê o
// botão para decidir se aperta, e "Armas marciais" sozinho não diz se apertar
// dá ou tira.
func oRotuloDaTroca(linha proficienciaDaFicha) string {
	if linha.Tem {
		return "Remover proficiência: " + linha.Rotulo
	}
	return "Adicionar proficiência: " + linha.Rotulo
}
