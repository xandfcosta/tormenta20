package book

import (
	"encoding/json"
	"sync"

	"t20engine/catalog"
)

// Os TEXTOS das habilidades de raça (ALE-239).
//
// Este arquivo existe por uma razão específica e vale explicá-la, porque à
// primeira vista ele parece uma segunda leitura do mesmo catálogo: o
// `engine.RaceDefinition` NÃO serve aqui. Ele é um subconjunto DELIBERADO —
// guarda `Modifiers` e `Variants`, que é do que o motor de regras precisa, e
// não guarda `Name` nem `Description`, que é do que a TELA precisa. O
// `engine-go/CLAUDE.md` registra que os catálogos tipados são subconjuntos de
// propósito; ampliar a struct do motor para caber texto de tela faria o motor
// carregar dado que ele nunca lê.
//
// Então são dois olhares sobre o MESMO arquivo, cada um com o seu tipo, e não
// duas fontes de verdade. O arquivo continua sendo um só
// (`catalog/data/race-defs.json`), autorado num lugar só.
//
// E é aqui que o dividendo da ALE-107 aparece pela primeira vez na migração: a
// SPA BAIXA este catálogo para mostrar quatro linhas de texto no dossiê. No
// servidor ele já está em memória, embutido no binário — a tela nova não pede
// nada, e some junto o `settledQuery` que existia porque ler o catálogo ainda
// pendente suspendia o route match e reanimava a cena inteira (ALE-95).

// RaceAbility é o que o dossiê mostra: nome e uma linha.
type RaceAbility struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Variants    []AbilityVariant `json:"variants"`
}

type RaceForScreen struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Abilities []RaceAbility `json:"abilities"`
}

// AbilityVariant é uma opção de uma habilidade que se escolhe — a Resistência
// Elemental do qareen tem seis, a Herança Divina do suraggel tem duas.
type AbilityVariant struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Lido UMA vez: o conteúdo vem de `go:embed` e não muda enquanto o binário for
// o mesmo. É a mesma decisão do `writeCatalogJSON`, que comprime uma vez em vez
// de por requisição.
var (
	raceTraitsOnce sync.Once
	raceTraitsByID map[string]RaceForScreen
)

func RaceTraitsByKey() map[string]RaceForScreen {
	raceTraitsOnce.Do(func() {
		raceTraitsByID = map[string]RaceForScreen{}
		raw, ok := catalog.Resource("race-defs")
		if !ok {
			// Catálogo ausente é caso NORMAL de degradação: o dossiê fica sem a
			// lista de habilidades e o resto da cena continua de pé. Derrubar a
			// tela inteira por causa de quatro linhas de texto seria pior.
			return
		}
		var list []RaceForScreen
		if err := json.Unmarshal(raw, &list); err != nil {
			return
		}
		for _, r := range list {
			// Por ID E por NOME, porque o personagem guarda a raça por um dos
			// dois — o `raceAbilityBlurbs` da SPA procura pelos dois pelo mesmo
			// motivo, e no `race-defs.json` de hoje eles coincidem ("Humano"),
			// mas coincidir não é o mesmo que ser garantido.
			raceTraitsByID[r.ID] = r
			raceTraitsByID[r.Name] = r
		}
	})
	return raceTraitsByID
}

// RaceAbilities são as primeiras `limite` habilidades da raça, para o
// dossiê.
//
// Raça desconhecida devolve lista vazia, não erro: um personagem com raça
// que saiu do catálogo continua abrindo, só sem as linhas de sabor.
func RaceAbilities(raceKey string, limit int) []RaceAbility {
	race, ok := RaceTraitsByKey()[raceKey]
	if !ok {
		return nil
	}
	if len(race.Abilities) > limit {
		return race.Abilities[:limit]
	}
	return race.Abilities
}
