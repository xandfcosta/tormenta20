package table

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"t20engine/domain/book"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/app/character"
	"t20engine/domain/live"
	"t20engine/serve/web/routes"
)

// O ROLAR DE NOVO NA FAIXA DO TESTE (p226, ALE-507): só quem rolou, enquanto o
// teste é o último da mesa, e vale o segundo dado. A janela do livro — "antes
// de o mestre declarar se você passou" — fica com a mesa, que decide em voz.

// rerollOffer é um botão da faixa: "Rolar de novo · Etiqueta · 1 PM".
type rerollOffer struct {
	Label string
	// Path é o endereço do gesto; o `@post` sai do `offerCommand`.
	Path string
}

// offerCommand é o `@post` de uma oferta.
func offerCommand(offer rerollOffer) string {
	return "@post('" + offer.Path + "')"
}

// rerollsSkillTest rola de novo o último teste com o gasto do caminho.
func rerollsSkillTest(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	ctx := c.R.Context()
	state, err := st.deps.Sessions().State(ctx, c.SessionID)
	if err != nil {
		return nil, err
	}
	test := state.LastSkillTest
	if test == nil {
		return nil, errors.New("não há teste na mesa para rolar de novo")
	}
	if test.ByUserID != c.User {
		return nil, fmt.Errorf("o teste é de %s, e só quem rolou rola de novo", test.Who)
	}
	row, err := st.deps.Queries().GetCharacter(ctx, test.CharacterID)
	if err != nil {
		return nil, fmt.Errorf("a ficha de %s não existe mais", test.Who)
	}
	// O "Eu rolei" só existe na faixa do teste rolado na mesa: no que o app rolou, o
	// número no sinal é sobra de outro teste, e o servidor rola.
	var d20 *int
	if test.ByHand {
		if d20, err = rerollDie(c); err != nil {
			return nil, err
		}
	}
	rolled, err := st.plays.RerollSkillTest(ctx, row, *test, chi.URLParam(c.R, "gasto"), d20)
	// O GASTO grava na ficha (a sorte, o PM), e a ficha aberta só se redesenha no evento.
	st.deps.CharacterChanged(c.R.Context(), row.ID)
	if err != nil {
		return nil, err
	}
	recorded, err := st.deps.Sessions().RecordRoll(ctx, c.SessionID, rolled)
	if err == nil {
		// O "Eu rolei" ZERA depois de valer: o próximo teste rolado na mão abria com
		// este número já escrito. Na recusa ele fica, para quem digitou não o perder.
		c.Signals["reroll_d20"] = ""
	}
	return recorded, err
}

// rerollDie lê o d20 que a mesa digitou no "Eu rolei" do teste. Vazio é o servidor
// rolar.
func rerollDie(c commandCtx) (*int, error) {
	var signals struct {
		D20 json.RawMessage `json:"reroll_d20"`
	}
	if err := datastar.ReadSignals(c.R, &signals); err != nil {
		return nil, nil
	}
	raw := strings.Trim(string(signals.D20), `" `)
	if raw == "" || raw == "null" {
		return nil, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return nil, fmt.Errorf("o d20 da mesa tem de ser um número de 1 a 20, e veio %q", raw)
	}
	return &n, nil
}

// skillTestBandFor é a faixa do teste com os botões de rolar de novo de QUEM
// OLHA, quando é ele quem rolou.
func (s Scene) skillTestBandFor(
	st *live.SessionRuntimeState, userID, campaignID, sessionID int64, sheets *tableSheets,
) *rollBand {
	band := skillTestBandOf(st)
	if band == nil || st.LastSkillTest.ByUserID != userID || st.LastSkillTest.CharacterID == 0 {
		return band
	}
	test := st.LastSkillTest
	dto, ok := sheets.of(test.CharacterID)
	if !ok {
		return band
	}
	skill := test.Skill
	if test.UsedSkill != "" {
		skill = test.UsedSkill
	}
	band.Offers = rerollOffersAt(character.RerollOffersFor(dto, skill), routes.Session(campaignID, sessionID)+"/teste/rolar-de-novo/")
	return band
}

// rerollOffersAt são os botões de rolar de novo de UM gesto, o ataque ou o teste:
// o rótulo é o mesmo nos dois, e o endereço muda só no prefixo de quem rola.
func rerollOffersAt(offers []book.SkillSpendOffer, prefix string) []rerollOffer {
	out := make([]rerollOffer, 0, len(offers))
	for _, offer := range offers {
		out = append(out, rerollOffer{
			Label: fmt.Sprintf("Rolar de novo · %s · %d PM", offer.Name, offer.PM),
			Path:  prefix + offer.Key,
		})
	}
	return out
}

// writtenRerolls diz os "rolar de novo" pagos: "rolou de novo (Etiqueta · 1
// PM): o d20 4 saiu". Vale o último dado (p226). O dado que cada um ROLOU é o
// que o seguinte descartou, e o último é o `roll` que ficou.
func writtenRerolls(rerolls []live.DieReroll, roll int) string {
	parts := make([]string, 0, len(rerolls))
	for i, r := range rerolls {
		rolled := roll
		if i+1 < len(rerolls) {
			rolled = rerolls[i+1].Discarded
		}
		paid := r.Name
		if r.Bonus != 0 {
			paid += fmt.Sprintf(" %s%d", signOf(r.Bonus), abs(r.Bonus))
		}
		part := fmt.Sprintf("rolou de novo (%s · %d PM): o d20 %d saiu", paid, r.PM, r.Discarded)
		if r.Natural {
			part += fmt.Sprintf("; %d conta como 20 natural", rolled)
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, ", ")
}
