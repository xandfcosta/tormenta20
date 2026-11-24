package table

import (
	"fmt"

	"t20engine/domain/live"
)

// A FAIXA DO DISPARO — a bola de fogo da Flamejante (p335). O dano e a resistência do alvo são do mestre,
// aplicados com os gestos de PV que a fila já tem. As duas sessões a veem inteira: a bola de fogo cai à vista de
// todos, e a CD é de quem dispara, não da ficha do alvo.

// blastBandOf traduz o disparo do regime na faixa, ou nil quando não há. O alvo sai da FILA, pela razão do
// ataque: a linha pode ter sido renomeada entre disparar e desenhar.
func blastBandOf(st *live.SessionRuntimeState) *rollBand {
	if st == nil || st.LastBlast == nil {
		return nil
	}
	b := st.LastBlast
	band := &rollBand{
		Badge: b.Weapon,
		What:  fmt.Sprintf("%s · %s → %s", b.Who, b.Name, entryLabel(st, b.TargetEntryID)),
		Lines: []rollLine{{Total: b.Total, Tally: writtenRoll(b.Dice, b.Rolls, b.DamageType, b.ByHand)}},
		Help:  []string{fmt.Sprintf("%s CD %d %s · alcance %s", b.Save, b.DC, b.OnSave, b.Range)},
	}
	if i := live.FindEntryIndex(st, b.TargetEntryID); i >= 0 && st.Initiative[i].CharacterID != nil {
		band.TargetCharacterID = *st.Initiative[i].CharacterID
	}
	return band
}
