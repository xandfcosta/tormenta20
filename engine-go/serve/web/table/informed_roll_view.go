package table

import (
	"strconv"
	"strings"
)

// AS FAIXAS QUE INFORMAM UMA ROLAGEM (ver GLOSSARY, **rolagem informada**): o teste, o disparo do encanto e a
// magia conjurada. Elas são irmãs da faixa do ataque MENOS os verbos — não há o que confirmar nem cancelar, e ✓ e
// ✕ aqui seriam dois controles que não fazem nada. O que muda de uma para outra é o que se escreve nela, e por
// isso a forma é uma só: `rollBand`, desenhada pelo `informedRoll`.
//
// Faixa nova: o tipo em `informedRollKind`, a região em `informedRollRegions` e o tradutor no `LoadView`. A
// lista alimenta as duas sessões E o stream, e é por isso que ela é uma só: uma faixa desenhada na página e
// esquecida no stream nasce e nunca mais muda.

// informedRollKind é o índice de uma faixa em `View.Rolls`, na ordem da tela.
type informedRollKind int

const (
	skillTestRoll informedRollKind = iota
	blastRoll
	spellCastRoll
	informedRollKindCount
)

// informedRollRegions são o id e o nome acessível de cada faixa. O id existe SEMPRE, com a faixa vazia
// escondida: o stream remenda POR ID, e uma região que some leva o id junto.
var informedRollRegions = [informedRollKindCount]struct{ id, label string }{
	skillTestRoll: {"table-skill-test", "Teste rolado"},
	blastRoll:     {"table-blast", "Disparo do encanto"},
	spellCastRoll: {"table-spell-cast", "Magia conjurada"},
}

// rollBand é uma rolagem informada do jeito que a mesa lê.
type rollBand struct {
	// Badge é a etiqueta da cabeça ("Teste", a arma do encanto), vazia na faixa sem ela.
	Badge string
	// Seal e Tone são o selo que só o dado decide — os naturais da p221 — e o tom dele no kit; sem tom, sem selo.
	Seal string
	Tone string
	// Who e What são a cabeça: Who em texto corrido ("Arwen · "), What em negrito ("Atletismo").
	Who  string
	What string
	// Lines são os totais com a conta de cada um, e a condição do texto quando há.
	Lines []rollLine
	// Help são as frases do que o mestre aplica: a resistência com a CD, o alcance.
	Help []string
	// TargetCharacterID é a ficha atrás do alvo, zero no NPC e na faixa sem alvo, e Taken as frases de como o
	// dano a atinge — só na faixa do MESTRE, que o `LoadView` preenche no ramo dele.
	TargetCharacterID int64
	Taken             []string
	// Offers são os "rolar de novo" de QUEM ROLOU, e só ele os vê. HandReroll: o dado veio da mesa, e o rolar de
	// novo pede o "Eu rolei".
	Offers     []rerollOffer
	HandReroll bool
}

// rollLine é um total e a conta dele: "6d6 (3, 5, 1, 6, 2, 4) de fogo".
type rollLine struct {
	Total int
	Tally string
	Note  string
}

// writtenRoll é a conta de uma rolagem de dano ou cura: os dados que o app rolou entre parênteses, ou "da mesa"
// quando o total veio da mão, e o tipo — "6d6 (3, 5, 1, 6, 2, 4) de fogo", "6d6 de fogo · da mesa".
func writtenRoll(dice string, rolls []int, kind string, byHand bool) string {
	tally := dice
	if len(rolls) > 0 {
		written := make([]string, 0, len(rolls))
		for _, die := range rolls {
			written = append(written, strconv.Itoa(die))
		}
		tally += " (" + strings.Join(written, ", ") + ")"
	}
	if kind != "" {
		tally += " de " + kind
	}
	if byHand {
		tally += " · da mesa"
	}
	return tally
}
