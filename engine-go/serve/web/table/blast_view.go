package table

import (
	"fmt"
	"strconv"
	"strings"

	"t20engine/domain/live"
)

// A FAIXA DO DISPARO — a bola de fogo da Flamejante (p335) — é irmã da do teste: ela INFORMA. O dano e a
// resistência do alvo são do mestre, aplicados com os gestos de PV que a fila já tem; desenhar ✓ e ✕ aqui
// daria à mesa dois controles que não fazem nada. As duas sessões a veem inteira: a bola de fogo cai à vista
// de todos, e a CD é de quem dispara, não da ficha do alvo.

// blastBand é o disparo do jeito que a mesa lê.
type blastBand struct {
	// Head é "Arwen · Bola de fogo → Goblin", e Weapon a arma do encanto.
	Head   string
	Weapon string
	Total  int
	// Tally é a conta: "6d6 (3, 5, 1, 6, 2, 4) de fogo", ou "6d6 de fogo · da mesa" no total rolado na mão.
	Tally string
	// Save é a resistência: "Reflexos CD 17 reduz à metade · alcance médio".
	Save string
}

// blastBandOf traduz o disparo do regime na faixa, ou nil quando não há. O alvo sai da FILA, pela razão do
// ataque: a linha pode ter sido renomeada entre disparar e desenhar.
func blastBandOf(st *live.SessionRuntimeState) *blastBand {
	if st == nil || st.LastBlast == nil {
		return nil
	}
	b := st.LastBlast
	dice := b.Dice
	if len(b.Rolls) > 0 {
		written := make([]string, 0, len(b.Rolls))
		for _, die := range b.Rolls {
			written = append(written, strconv.Itoa(die))
		}
		dice += " (" + strings.Join(written, ", ") + ")"
	}
	tally := dice + " de " + b.DamageType
	if b.ByHand {
		tally += " · da mesa"
	}
	return &blastBand{
		Head:   fmt.Sprintf("%s · %s → %s", b.Who, b.Name, entryLabel(st, b.TargetEntryID)),
		Weapon: b.Weapon, Total: b.Total, Tally: tally,
		Save: fmt.Sprintf("%s CD %d %s · alcance %s", b.Save, b.DC, b.OnSave, b.Range),
	}
}
