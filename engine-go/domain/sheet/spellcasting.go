package sheet

import (
	"encoding/json"
	"strconv"

	"t20engine/domain/book"
)

// AS DUAS REGRAS DE CONJURAÇÃO que a ficha e a rota JSON leem juntas (ALE-278).
//
// Elas moravam junto da conjuração, no `serve/api`, e a cena da ficha as lia de
// lá enquanto tudo era um pacote só. Vieram para cá pela dependência: uma é uma
// tabela sem dependência nenhuma, a outra pergunta sobre `[]ClassDTO` — nenhuma
// das duas toca HTTP, catálogo ou banco. Quem as chama hoje é o
// `app/character/cast.go`.
//
// É a mesma família do `equip.go` e do `temp_hp.go`, que saíram na fatia
// anterior: regra da ficha hospedada no `api` por acidente de história.

// SpellBasePmCost é a Tabela 4-1, "Custo de Magias" (livro p170).
var SpellBasePmCost = map[int]int{0: 0, 1: 1, 2: 3, 3: 6, 4: 10, 5: 15}

// alwaysPrepareClasses são as classes que preparam magia sempre.
var alwaysPrepareClasses = map[string]bool{"Clérigo": true, "Druida": true}

// RequiresPreparation diz se esta ficha precisa PREPARAR a magia antes de
// conjurar: Clérigo e Druida sempre, e o Arcanista quando o caminho é "mago".
func RequiresPreparation(classes []ClassDTO, classChoicesRaw string) bool {
	hasArcanist := false
	for _, c := range classes {
		if alwaysPrepareClasses[c.ClassName] {
			return true
		}
		if c.ClassName == "Arcanista" {
			hasArcanist = true
		}
	}
	if !hasArcanist {
		return false
	}
	var choices map[string]struct {
		Path string `json:"caminho"`
	}
	_ = json.Unmarshal([]byte(classChoicesRaw), &choices)
	return choices["Arcanista"].Path == "mago"
}

// HighestCastableCircle é o maior círculo que o personagem alcança.
//
// O piso é o círculo da PRÓPRIA magia, e essa é a regra que não se adivinha: uma
// magia concedida por poder (Totem Espiritual, p42) é conjurável por quem não
// tem classe nenhuma de conjurador — no círculo dela, e só nele. Sem o piso, um
// bárbaro com Totem não conseguiria conjurar a magia que o poder lhe deu.
func HighestCastableCircle(classes []ClassDTO, spellCircle int) int {
	best := spellCircle
	progressions := book.SpellProgressions()
	for _, entry := range classes {
		prog, casts := progressions[entry.ClassName]
		if !casts {
			continue
		}
		for circle := 1; circle <= prog.MaxCircle; circle++ {
			level := prog.UnlockLevel[strconv.Itoa(circle)]
			if level == nil || entry.Level < int64(*level) {
				continue
			}
			if circle > best {
				best = circle
			}
		}
	}
	return best
}

// IsCasterClass diz se a classe conjura.
func IsCasterClass(name string) bool {
	_, casts := book.SpellProgressions()[name]
	return casts
}
