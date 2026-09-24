package book

import (
	"encoding/json"
	"sync"

	"t20engine/domain/catalog"
	"t20engine/domain/engine"
)

// Os TEXTOS das habilidades de raça.
//
// Ele PARECE uma segunda leitura do mesmo catálogo e não é: o
// `engine.RaceDefinition` não serve aqui porque é um subconjunto DELIBERADO —
// não guarda `Name` nem `Description`, que é do que a TELA precisa. Ampliar a
// struct do motor para caber texto de tela faria o motor carregar dado que ele
// nunca lê.
//
// **A relação deixou de ser de subconjuntos disjuntos na ALE-384**, e vale
// dizer por quê: os dois lêem `Modifiers` agora. O motor os RESOLVE — funde
// tudo num conjunto de flags e perde a procedência no caminho —, e a tela
// precisa justamente da procedência, para nomear a habilidade que isentou o
// personagem de alguma coisa. É a mesma pergunta lida com dois propósitos, e
// não uma cópia por descuido.
//
// São dois olhares sobre o MESMO arquivo, cada um com o seu tipo, e não duas
// fontes de verdade: o `catalog/data/race-defs.json` continua autorado num lugar
// só.

// RaceAbility é o que o dossiê mostra: nome e uma linha.
type RaceAbility struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Variants    []AbilityVariant `json:"variants"`
	// Modifiers é o que a habilidade CONCEDE, e ele entrou para a tela poder
	// NOMEAR a habilidade responsável por uma isenção (ALE-384). O motor resolve
	// os modificadores num conjunto de flags e perde a procedência no caminho —
	// "quem me isentou" é pergunta de apresentação, e é aqui que ela se responde.
	Modifiers []engine.Modifier `json:"modifiers,omitempty"`
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

// Lido UMA vez: o conteúdo vem de `go:embed` e não muda enquanto o binário for o
// mesmo — o custo é de carga, não de pedido.
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
			// Por ID E por NOME, porque o personagem guarda a raça por um dos dois.
			// No `race-defs.json` de hoje eles coincidem ("Humano"), mas coincidir
			// não é o mesmo que ser garantido.
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
