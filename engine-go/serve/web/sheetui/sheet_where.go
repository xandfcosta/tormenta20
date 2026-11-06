package sheetui

import (
	"context"
	"fmt"

	"t20engine/infra/db/sqlcgen"
)

// A FAIXA SOB O CABEÇALHO diz qual ficha é esta: a cópia de uma mesa, ou o rascunho. Ver `sheetWhere`.

// sheetTable é a mesa da cópia: o nome, o endereço e se ela está em sessão agora.
type sheetTable struct {
	ID   int64
	Name string
	Live bool
}

func (s Scene) tableOf(ctx context.Context, row sqlcgen.Character) (*sheetTable, error) {
	if !row.CampaignId.Valid {
		return nil, nil
	}
	campaign, err := s.deps.Queries().GetCampaign(ctx, row.CampaignId.Int64)
	if err != nil {
		return nil, fmt.Errorf("a mesa %d da ficha %d: %w", row.CampaignId.Int64, row.ID, err)
	}
	live, err := s.deps.Queries().HasLiveSessionForCharacter(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	return &sheetTable{ID: campaign.ID, Name: campaign.Name, Live: live}, nil
}
