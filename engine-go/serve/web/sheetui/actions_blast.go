package sheetui

import (
	"fmt"
	"strconv"
	"strings"

	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// A BOLA DE FOGO DA FLAMEJANTE NA AÇÃO PADRÃO (p335). Ela é "em vez do ataque normal", então mora no grupo da
// ação do ataque, numa linha própria e não na tira da linha de ataque: o que sai dela não é golpe, não tem d20
// nem decisão de acerto (decisão do dono). A linha diz o que a mesa precisa para aplicar — o dano, o alcance,
// a resistência com a CD desta ficha — e o que ela custa.

// blastRows são as linhas do que as armas em punho disparam em vez do golpe, uma por encanto.
func blastRows(dto sheet.CharacterDTO, computed engine.ComputedSheet) []actionRow {
	if dto.Ruleset == nil {
		return nil
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		return nil
	}
	mana := int(dto.ManaAvailable())
	rows := []actionRow{}
	for _, blast := range dto.Ruleset.EnchantBlasts(ec) {
		path := "/acoes/bola-de-fogo/" + strconv.FormatInt(blast.ItemID, 10)
		row := actionRow{
			Name: blast.Name + " · " + blast.Weapon,
			// O "UMA VEZ POR RODADA" vai no detalhe, e o número fica fora, e isto foi o OLHO a 390px: na coluna
			// do custo, junto do "6d6" que o botão já diz, os dois espremiam o nome numa coluna de uma palavra.
			Detail: writtenBlast(blast, sheet.BlastSaveDC(dto.Level, computed.Attributes, blast.SaveAttributes)) +
				" · uma vez por rodada",
			Cost: fmt.Sprintf("%d PM", blast.PM),
			Can:  blast.PM <= mana,
			Rolls: []actionRoll{{
				Label: "Rolar " + blast.Damage, Path: path, NeedsTarget: true, Primary: true, Before: "$blast_total = 0; ",
			}},
			Hand: &handTotal{Path: path, Dice: blast.Damage},
		}
		if !row.Can {
			row.Why = fmt.Sprintf("falta PM: a %s pede %d", strings.ToLower(blast.Name), blast.PM)
		}
		rows = append(rows, row)
	}
	return rows
}

// writtenBlast é o que o disparo faz, como a mesa lê: "6d6 de fogo · alcance médio · Reflexos CD 17 reduz à
// metade". Sem a CD (a ficha sem os atributos computados), a resistência fica sem número em vez de inventar um.
func writtenBlast(blast engine.EnchantBlast, dc int) string {
	save := blast.Save + " " + blast.OnSave
	if dc > 0 {
		save = fmt.Sprintf("%s CD %d %s", blast.Save, dc, blast.OnSave)
	}
	return fmt.Sprintf("%s de %s · alcance %s · %s", blast.Damage, blast.DamageType, blast.Range, save)
}

// handTotal é o "ou rolei" de uma linha: o total que a mesa rolou na mão, mandado pelo mesmo gesto do Rolar
// (decisão do dono: o dado da mesa é de primeira classe). Dice é a notação, que dá o intervalo do campo.
type handTotal struct {
	Path string
	Dice string
}

// handMin e handMax são o intervalo do campo, pela notação: 6d6 vai de 6 a 36. Quem recusa o total fora dele
// é o servidor (`character.RollsTheBlast`); o campo só não oferece o que os dados não tiram.
func handMin(h handTotal) string {
	lowest, _, err := engine.DiceRange(h.Dice)
	if err != nil {
		return ""
	}
	return strconv.Itoa(lowest)
}

func handMax(h handTotal) string {
	_, highest, err := engine.DiceRange(h.Dice)
	if err != nil {
		return ""
	}
	return strconv.Itoa(highest)
}
