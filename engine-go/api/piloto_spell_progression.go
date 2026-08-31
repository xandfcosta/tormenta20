package api

import (
	"encoding/json"
	"strconv"
	"sync"

	"t20engine/catalog"
)

// A PROGRESSÃO DE CÍRCULO por classe (ALE-272, fatia 6).
//
// # Ela vivia SÓ no TypeScript, e por isso a regra era só de interface
//
// O `SPELL_PROGRESSION` do front dizia em que nível cada classe destrava cada
// círculo, e era com ele que a SPA TRANCAVA um aprimoramento de círculo alto. O
// servidor nunca soube dessa tabela: o `validateAugments` conferia índice,
// duplicata, `stacks ≥ 1` e "muda não empilha", e aceitava qualquer
// `requiresCircle` — um pedido montado à mão conjurava o que a regra não
// permite. São 126 dos 486 aprimoramentos do catálogo, um quarto deles.
//
// Travar na UI é UX; a fronteira é o servidor. A tabela veio para o
// `classes.json`, ao lado da página de cada classe, e agora as DUAS pontas leem
// a mesma coisa.
//
// # Ela foi MOVIDA, e não retranscrita
//
// Nenhum número aqui foi lido do livro nesta fatia: o que se garante é que a
// mudança é FIEL, e quem garante é `spell-progression-agree.test.ts`, que
// compara as duas cópias enquanto a SPA viver. Uma auditoria contra o livro é
// outro trabalho, e fingir que ela aconteceu seria pior que não fazê-la.

// spellProgression é o que uma classe conjuradora destrava, e quando.
type spellProgression struct {
	List      string `json:"list"`
	Attribute string `json:"attribute"`
	MaxCircle int    `json:"maxCircle"`
	// UnlockLevel é o nível em que cada círculo abre. NULO significa "esta
	// classe nunca chega lá" — o Bardo e o Druida param no 4º, o Paladino no 1º.
	// Ponteiro e não zero: nível 0 seria "abre de saída", que é outra coisa.
	UnlockLevel map[string]*int `json:"unlockLevel"`
}

type classOfBook struct {
	Name         string            `json:"name"`
	Spellcasting *spellProgression `json:"spellcasting"`
}

var (
	progressionOnce    sync.Once
	progressionByClass map[string]spellProgression
)

// spellProgressions lê a tabela do catálogo, uma vez.
func spellProgressions() map[string]spellProgression {
	progressionOnce.Do(func() {
		progressionByClass = map[string]spellProgression{}
		bruto, ok := catalog.Resource("classes")
		if !ok {
			return
		}
		var classes []classOfBook
		if err := json.Unmarshal(bruto, &classes); err != nil {
			return
		}
		for _, c := range classes {
			if c.Spellcasting != nil {
				progressionByClass[c.Name] = *c.Spellcasting
			}
		}
	})
	return progressionByClass
}

// highestCastableCircle é o maior círculo que o personagem alcança.
//
// O piso é o círculo da PRÓPRIA magia, e essa é a regra que não se adivinha: uma
// magia concedida por poder (Totem Espiritual, p42) é conjurável por quem não
// tem classe nenhuma de conjurador — no círculo dela, e só nele. Sem o piso, um
// bárbaro com Totem não conseguiria conjurar a magia que o poder lhe deu.
func highestCastableCircle(classes []ClassDTO, spellCircle int) int {
	melhor := spellCircle
	for _, entrada := range classes {
		prog, conjura := spellProgressions()[entrada.ClassName]
		if !conjura {
			continue
		}
		for circulo := 1; circulo <= prog.MaxCircle; circulo++ {
			nivel := prog.UnlockLevel[strconv.Itoa(circulo)]
			if nivel == nil || entrada.Level < int64(*nivel) {
				continue
			}
			if circulo > melhor {
				melhor = circulo
			}
		}
	}
	return melhor
}
