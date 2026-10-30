package table

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/app/character"
	"t20engine/domain/live"
	"t20engine/domain/sheet"
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
	d20, err := rerollDie(c)
	if err != nil {
		return nil, err
	}
	rolled, err := st.plays.RerollSkillTest(ctx, row, *test, chi.URLParam(c.R, "gasto"), d20)
	if err != nil {
		return nil, err
	}
	return st.deps.Sessions().RecordSkillTest(ctx, c.SessionID, rolled)
}

// rerollDie lê o d20 da mesa, quando o primeiro dado veio de lá. Vazio é o
// servidor rolar.
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
	ctx context.Context, st *live.SessionRuntimeState, userID, campaignID, sessionID int64,
) *skillTestBand {
	band := skillTestBandOf(st)
	if band == nil || st.LastSkillTest.ByUserID != userID || st.LastSkillTest.CharacterID == 0 {
		return band
	}
	test := st.LastSkillTest
	row, err := s.deps.Queries().GetCharacter(ctx, test.CharacterID)
	if err != nil {
		return band
	}
	dto, err := sheet.Load(ctx, s.deps.Queries(), s.deps.Catalogs(), row)
	if err != nil {
		return band
	}
	skill := test.Skill
	if test.UsedSkill != "" {
		skill = test.UsedSkill
	}
	for _, offer := range character.RerollOffersFor(dto, skill) {
		label := fmt.Sprintf("Rolar de novo · %s · %d PM", offer.Name, offer.PM)
		band.Offers = append(band.Offers, rerollOffer{
			Label: label,
			Path:  fmt.Sprintf("%s/teste/rolar-de-novo/%s", routes.Session(campaignID, sessionID), offer.Key),
		})
	}
	return band
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
