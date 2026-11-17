package table

import (
	"fmt"
	"strconv"

	"github.com/go-chi/chi/v5"

	"t20engine/app"
	"t20engine/domain/book"
	"t20engine/domain/live"
	"t20engine/domain/sheet"
	"t20engine/serve/web/routes"
)

// O ESTANDARTE (p54) NA MESA DO MESTRE. O cavaleiro recebe sozinho no início da cena; os
// aliados que "possam ver seu estandarte" recebem quando o mestre diz que veem, um
// toque por aliado (decisão do dono). A faixa só existe com a cena aberta e só para o
// mestre, e some para cada aliado que já recebeu nesta cena.

// bannerBand é o Estandarte de UM cavaleiro do grupo.
type bannerBand struct {
	// Head é "Estandarte de Sir Aldo · 3 PM temporários a quem vê o estandarte".
	Head   string
	Offers []bannerOffer
}

// bannerOffer é o toque que entrega o Estandarte a um aliado.
type bannerOffer struct {
	Name string
	Path string
}

// bannerBandsOf são as faixas dos cavaleiros do grupo, com a cena aberta. A conta do
// número é a do verbete (`PowerGrant.AmountFor`), a mesma que o gesto aplica.
func bannerBandsOf(st *live.SessionRuntimeState, group []Member, sheets *tableSheets, campaignID, sessionID int64) []bannerBand {
	spec := book.ActivationOf(book.BannerPowerID, "")
	if st == nil || !st.InScene() || spec == nil || spec.Grant == nil {
		return nil
	}
	var bands []bannerBand
	for _, knight := range group {
		dto, ok := sheets.of(knight.CharacterID)
		if !ok || !sheet.OwnsActivation(dto, *spec) {
			continue
		}
		computed, err := sheet.Compute(dto)
		if err != nil {
			continue
		}
		amount := spec.Grant.AmountFor(int(dto.Level), computed.Attributes[spec.Grant.Attribute].Total)
		band := bannerBand{Head: fmt.Sprintf("Estandarte de %s · %d PM temporários a quem vê o estandarte", knight.Name, amount)}
		for _, ally := range group {
			if ally.CharacterID == knight.CharacterID {
				continue
			}
			if allyDTO, ok := sheets.of(ally.CharacterID); !ok || sheet.ReceivedFrom(allyDTO, book.BannerPowerID) {
				continue
			}
			band.Offers = append(band.Offers, bannerOffer{
				Name: ally.Name,
				Path: fmt.Sprintf("%s/estandarte/%d/%d", routes.Session(campaignID, sessionID), knight.CharacterID, ally.CharacterID),
			})
		}
		if len(band.Offers) > 0 {
			bands = append(bands, band)
		}
	}
	return bands
}

// givesBanner é o toque do mestre num aliado da faixa.
func givesBanner(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	knight, err := strconv.ParseInt(chi.URLParam(c.R, "cavaleiro"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("o cavaleiro %q não é um id: %w", chi.URLParam(c.R, "cavaleiro"), app.ErrRefused)
	}
	ally, err := strconv.ParseInt(chi.URLParam(c.R, "aliado"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("o aliado %q não é um id: %w", chi.URLParam(c.R, "aliado"), app.ErrRefused)
	}
	return st.party.GiveBanner(c.R.Context(), app.Caller{ID: c.User}, c.CampaignID, c.SessionID, knight, ally)
}
