package campaigns

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"slices"

	"t20engine/app"
	"t20engine/domain/sheet"
)

// O PODER QUE A REGRA EMPRESTAVA SAI DA FICHA quando ela deixa de emprestar (decisão do dono). Só esse:
// o poder de outro deus que a ficha já guardava fica, porque não foi a regra que o deu. Por isso a cena
// fotografa o que cada ficha podia escolher ANTES do gesto, e compara com o depois.

// grantedOffers são, por ficha da mesa, os poderes concedidos que ela escolhe hoje sob o próprio deus.
func (s Scene) grantedOffers(ctx context.Context, campaignID int64) (map[int64][]string, error) {
	members, err := s.deps.Queries().ListMembers(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	out := map[int64][]string{}
	for _, member := range members {
		row, err := s.deps.Queries().GetCharacter(ctx, member.Characterid)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}
		dto, err := sheet.Load(ctx, s.deps.Queries(), s.deps.Catalogs(), row)
		if err != nil {
			return nil, err
		}
		if dto.God != nil && *dto.God != "" {
			out[member.Characterid] = sheet.GrantedPowerNames(dto, *dto.God)
		}
	}
	return out, nil
}

// dropWhatIsNoLongerLent tira o que a ficha podia escolher antes e não pode mais. A regra já mudou, então
// a falha aqui não desfaz o gesto: ela vai para o log, e a ficha mostra o poder como de outro deus.
func (s Scene) dropWhatIsNoLongerLent(ctx context.Context, who app.Caller, campaignID int64, before map[int64][]string) {
	after, err := s.grantedOffers(ctx, campaignID)
	if err != nil {
		slog.Error("ler as fichas depois da regra da mesa", "campaign", campaignID, "error", err)
		return
	}
	lost := map[int64][]string{}
	for characterID, offered := range before {
		gone := slices.DeleteFunc(slices.Clone(offered), func(power string) bool { return slices.Contains(after[characterID], power) })
		if len(gone) > 0 {
			lost[characterID] = gone
		}
	}
	if len(lost) == 0 {
		return
	}
	if err := s.life.DropGrantedPowers(ctx, who, campaignID, lost); err != nil {
		slog.Error("tirar os poderes que a regra deixou de emprestar", "campaign", campaignID, "error", err)
	}
}

// grantedOffersOfTheOwner fotografa o antes, só para o dono da mesa: quem não é dono é recusado pelo
// próprio gesto logo depois, e não deve custar a leitura de todas as fichas.
func (s Scene) grantedOffersOfTheOwner(ctx context.Context, who app.Caller, campaignID int64) (map[int64][]string, error) {
	if _, err := s.access.OwnedCampaign(ctx, who, campaignID); err != nil {
		return nil, nil
	}
	return s.grantedOffers(ctx, campaignID)
}
