package sheetui

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"t20engine/app/character"
	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// AS AÇÕES DA MAGIA NA ABA AÇÕES (ver GLOSSARY, **ação da magia**): enquanto a magia está em vigor, cada opção dela
// é uma linha no grupo da ação que custa — o Esmagar da Mão em movimento, a energia mista do Relâmpago em padrão,
// o dardo da Lança em livre —, como a bola de fogo da Flamejante. O Rolar abre o diálogo com o dado já aumentado
// pelos aprimoramentos da conjuração e o "ou rolei" de cada parcela; o gesto cobra a ação da vez.

// spellActionSheet é o diálogo de uma opção: o dado escrito, os campos da mão e o gesto.
type spellActionSheet struct {
	Dialog string
	Title  string
	Lines  []handRollLine
	Roll   actionRoll
	// Elements são os tipos do Raio Elemental, no seletor do diálogo do Raio Arcano; vazio no resto.
	Elements []boltElementOption
}

// handResets zera os totais da mão ao abrir o diálogo: o número digitado numa opção não vale na outra.
const handResets = "$spell_hand_1 = 0; $spell_hand_2 = 0; $spell_hand_3 = 0; $spell_hand_4 = 0; "

// spellActionRows são as linhas das ações das magias em vigor, pelo custo de cada uma.
func spellActionRows(dto sheet.CharacterDTO, computed engine.ComputedSheet) map[engine.ActionCost][]actionRow {
	rows := map[engine.ActionCost][]actionRow{}
	for _, spellEffect := range sheet.CasterOf(dto).SpellEffects() {
		onSheet, effect, actions := spellEffect.Spell, spellEffect.Effect, spellEffect.Actions()
		if actions == nil {
			continue
		}
		dc, _ := onSheet.DC(computed)
		for i := range actions.Options {
			action, parts, err := spellEffect.Action(i)
			if err != nil {
				continue
			}
			// A OPÇÃO QUE É MANOBRA não tem dado nem diálogo: o Rolar pede o alvo da barra da Mesa, como as manobras
			// com arma, e o detalhe diz o que a mão soma ao d20 (ver GLOSSARY, **ação da magia**).
			if action.Maneuver != "" {
				bonus, found := character.ManeuverBonusOf(computed, action)
				if !found {
					continue
				}
				name := action.Name + " · " + onSheet.Spell.Name
				detail := []string{fmt.Sprintf("manobra %s · %s %s contra o Luta do alvo", action.Maneuver, action.Skill, book.WithSign(bonus))}
				if action.Note != "" {
					detail = append(detail, action.Note)
				}
				rows[engine.ActionCost(action.Action)] = append(rows[engine.ActionCost(action.Action)], actionRow{
					Name: name, Detail: strings.Join(detail, " · "), Can: true,
					Rolls: []actionRoll{{Label: "Rolar", Path: fmt.Sprintf("/acoes/magia/%d/%d", effect.ID, i), NeedsTarget: true, Primary: true}},
				})
				continue
			}
			name := action.Name + " · " + onSheet.Spell.Name
			written := make([]string, 0, len(parts))
			for _, part := range parts {
				written = append(written, writtenSpellPart(part))
			}
			detail := []string{strings.Join(written, " + ")}
			if action.Note != "" {
				detail = append(detail, action.Note)
			}
			save := action.Save
			if save == "" {
				save = book.SpellResistanceWritten(onSheet.Spell)
			}
			if save != "" && dc > 0 {
				save += fmt.Sprintf(" · CD %d", dc)
			}
			if save != "" {
				detail = append(detail, save)
			}
			if actions.OncePerTurn {
				detail = append(detail, "uma vez por turno")
			}
			if left, counted := spellEffect.Charges(); counted {
				detail = append(detail, fmt.Sprintf("sobram %d", left))
			}
			dialog := fmt.Sprintf("spell-action-%d-%d", effect.ID, i)
			path := fmt.Sprintf("/acoes/magia/%d/%d", effect.ID, i)
			row := actionRow{
				Name: name, Detail: strings.Join(detail, " · "), Can: true,
				Rolls: []actionRoll{{Label: "Rolar", Dialog: dialog, Before: handResets, Path: path, Primary: true}},
				SpellSheet: &spellActionSheet{
					Dialog: dialog, Title: name, Lines: handRollLinesOf(parts, name),
					Roll: actionRoll{Label: "Rolar", Path: path},
				},
			}
			cost := engine.ActionCost(action.Action)
			rows[cost] = append(rows[cost], row)
		}
	}
	// O RAIO ARCANO conta como magia (p39), e a linha dele mora com as das magias, no grupo da ação padrão.
	if row, has := arcaneBoltRow(dto, computed); has {
		rows[engine.ActionStandard] = append(rows[engine.ActionStandard], row)
	}
	return rows
}

// actsOnSpell faz a ação da magia em vigor pedida: confere a vez (a ação e o "uma vez por turno") ANTES de rolar,
// rola e grava o que ela gasta da magia, cobra a ação, e põe o resultado na faixa da magia da mesa.
func actsOnSpell(s Scene, r *http.Request, row sqlcgen.Character, signals Signals) error {
	effectID, err := strconv.ParseInt(chi.URLParam(r, "efeito"), 10, 64)
	if err != nil {
		return fmt.Errorf("o efeito %q não é um número", chi.URLParam(r, "efeito"))
	}
	option, err := strconv.Atoi(chi.URLParam(r, "opcao"))
	if err != nil {
		return fmt.Errorf("a ação %q não é um número", chi.URLParam(r, "opcao"))
	}
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	asked, err := character.SpellActionOf(dto, effectID, option)
	if err != nil {
		return err
	}
	// A OPÇÃO QUE É MANOBRA vai pela manobra da mesa, contra o alvo da barra, com a mão no lugar da arma: a
	// proposta cobra a ação de movimento, e o veredito é do mestre (ver GLOSSARY, **ação da magia**).
	if asked.Action.Maneuver != "" {
		bonus, err := asked.ManeuverBonus(dto)
		if err != nil {
			return err
		}
		hand := &SpellHand{Name: asked.Spell.Spell.Name, Bonus: bonus}
		return s.proposeOnTheTable(r, row, ActionStrike{Maneuver: asked.Action.Maneuver, Hand: hand}, signals)
	}
	cost := engine.ActionCost(asked.Action.Action)
	refusal := fmt.Sprintf("%s é uma vez por turno (p%d), e já saiu nesta vez", asked.Spell.Spell.Name, asked.Spell.Spell.BookPage)
	if err := s.deps.SpellActionFitsOnTurn(r.Context(), row.ID, cost, asked.OnceKey(), refusal); err != nil {
		return err
	}
	cast, err := s.plays.ActOnSpell(r.Context(), dto, asked, signals.spellHandTotals())
	if err != nil {
		return err
	}
	if err := s.deps.SpendSpellActionOnTurn(r.Context(), row.ID, cost, asked.OnceKey(), refusal); err != nil {
		return err
	}
	return s.deps.PublishSpellCast(r.Context(), row.ID, cast)
}
