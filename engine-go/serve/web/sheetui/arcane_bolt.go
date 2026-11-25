package sheetui

import (
	"fmt"
	"net/http"
	"strings"

	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// O RAIO ARCANO NA ABA AÇÕES (ver GLOSSARY, **raio arcano**): uma linha no grupo da ação padrão, e o Rolar abre o
// diálogo da ação da magia — o dado, o "ou rolei" e, com o Raio Elemental, o tipo. O gesto cobra a ação da vez e põe
// o raio na faixa da magia, porque ele conta como magia (p39).

// arcaneBoltPath é o endereço do gesto, depois de `/personagens/{id}`.
const arcaneBoltPath = "/acoes/raio-arcano"

// boltElementOption é um tipo do Raio Elemental no seletor do diálogo, com o que ele custa e o que deixa.
type boltElementOption struct {
	Value string
	Label string
}

// arcaneBoltRow é a linha do Raio Arcano, ou falso sem o poder.
func arcaneBoltRow(dto sheet.CharacterDTO, computed engine.ComputedSheet) (actionRow, bool) {
	bolt, has := sheet.ArcaneBoltOf(dto, computed)
	if !has {
		return actionRow{}, false
	}
	parts, err := bolt.Roll("")
	if err != nil {
		return actionRow{}, false
	}
	const name = "Raio Arcano"
	reach := "curto"
	if bolt.Range == "medio" {
		reach = "médio"
	}
	detail := []string{writtenSpellPart(parts[0]), fmt.Sprintf("Reflexos reduz à metade · CD %d", bolt.DC), "alcance " + reach}
	dialog := "arcane-bolt"
	row := actionRow{
		Name: name, Detail: strings.Join(detail, " · "), Can: true,
		Rolls: []actionRoll{{Label: "Rolar", Dialog: dialog, Before: handResets, Path: arcaneBoltPath, Primary: true}},
		SpellSheet: &spellActionSheet{
			Dialog: dialog, Title: name, Lines: handRollLinesOf(parts, name),
			Roll: actionRoll{Label: "Rolar", Path: arcaneBoltPath},
		},
	}
	if bolt.Elemental {
		row.SpellSheet.Elements = []boltElementOption{{Value: "", Label: "Essência"}}
		for _, e := range sheet.ArcaneBoltElements {
			row.SpellSheet.Elements = append(row.SpellSheet.Elements, boltElementOption{
				Value: e.Kind, Label: fmt.Sprintf("%s (+%d PM) · quem falha fica %s", capitalized(e.Kind), sheet.ArcaneBoltElementPM, e.Condition),
			})
		}
	}
	return row, true
}

// firesArcaneBolt dispara o Raio Arcano: confere a ação padrão da vez ANTES, dispara e cobra o PM do Elemental,
// cobra a ação, e põe o raio na faixa da magia da mesa.
func firesArcaneBolt(s Scene, r *http.Request, row sqlcgen.Character, signals Signals) error {
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	const refusal = "o Raio Arcano gasta a ação padrão, e ela já saiu nesta vez"
	if err := s.deps.SpellActionFitsOnTurn(r.Context(), row.ID, engine.ActionStandard, "", refusal); err != nil {
		return err
	}
	cast, err := s.plays.FireArcaneBolt(r.Context(), dto, signals.ArcaneBoltElement, signals.spellHandTotals())
	if err != nil {
		return err
	}
	if err := s.deps.SpendSpellActionOnTurn(r.Context(), row.ID, engine.ActionStandard, "", refusal); err != nil {
		return err
	}
	return s.deps.PublishSpellCast(r.Context(), row.ID, cast)
}
