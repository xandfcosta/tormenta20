package combat

import (
	"context"
	"encoding/json"
	"fmt"

	"t20engine/app"
	"t20engine/domain/book"
	"t20engine/domain/creature"
	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// Roster traduz uma linha da fila no combatente que a regra precisa.
//
// São TRÊS procedências e a linha diz qual: a FICHA de um personagem, o
// VERBETE do livro, ou o BLOCO que o mestre escreveu. As três respondem as
// mesmas quatro perguntas, e é por isso que o gesto de atacar conhece só o
// `Combatant` — juntar as três aqui é o que impede o `if` de procedência de
// aparecer espalhado por quem usa.
type Roster struct {
	queries  *sqlcgen.Queries
	catalogs *engine.Catalogs
}

func NewRoster(q *sqlcgen.Queries, catalogs *engine.Catalogs) Roster {
	return Roster{queries: q, catalogs: catalogs}
}

// A CAMPANHA entra porque o bloco de criatura é DELA, e o id do bloco veio do
// CLIENTE: o `initiative.npcEntry` grava o que mandarem, de propósito — um id
// desconhecido vira "sem bloco" na tela em vez de derrubar a adição no meio do
// combate —, e diz por escrito que "quem confere o dono do bloco é a rota que o
// serve". Esta é uma rota que serve, e ela não conferia (ALE-377).
func (r Roster) Of(ctx context.Context, campaignID int64, e live.InitiativeEntry) (Combatant, error) {
	switch {
	case e.CharacterID != nil:
		return r.fromSheet(ctx, e)
	case e.CreatureID != nil:
		return r.fromBlock(ctx, campaignID, e)
	case e.MonsterID != nil:
		return r.fromBestiary(e)
	}
	// LINHA DIGITADA À MÃO não tem Defesa em lugar nenhum: o mestre escreveu um
	// nome e uma iniciativa, e nada mais. Recusar com o nome dele é melhor que
	// medir contra Defesa 0, que daria acerto em tudo.
	return Combatant{}, fmt.Errorf(
		"%s é uma linha sem ficha nem bloco: não há Defesa para medir o ataque", e.Label)
}

// fromSheet: a ficha COMPUTADA manda, com os condicionais ligados juntos —
// a Fúria muda o ataque e a Defesa, e medir sem ela seria medir outro herói.
func (r Roster) fromSheet(ctx context.Context, e live.InitiativeEntry) (Combatant, error) {
	if r.catalogs == nil {
		return Combatant{}, fmt.Errorf("o catálogo de regras não está carregado")
	}
	row, err := r.queries.GetCharacter(ctx, *e.CharacterID)
	if err != nil {
		return Combatant{}, fmt.Errorf("carregar o personagem %d: %w", *e.CharacterID, err)
	}
	dto, err := sheet.Load(ctx, r.queries, r.catalogs, row)
	if err != nil {
		return Combatant{}, fmt.Errorf("montar a ficha de %s: %w", e.Label, err)
	}
	computed, err := sheet.Compute(dto)
	if err != nil {
		return Combatant{}, fmt.Errorf("computar a ficha de %s: %w", e.Label, err)
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		return Combatant{}, err
	}
	return Combatant{
		EntryID: e.ID, Label: e.Label,
		Weapons:         dto.Ruleset.ComputeWeaponCards(ec, sheet.ToStringSet(dto.Conditionals)),
		Defense:         computed.Defense.Total,
		DamageReduction: computed.DamageReduction.Total,
	}, nil
}

// fromBlock: o NPC que o mestre escreveu. Ele não empunha arma do catálogo — o
// ataque dele é texto no bloco —, então serve de ALVO e não de atacante.
func (r Roster) fromBlock(
	ctx context.Context, campaignID int64, e live.InitiativeEntry,
) (Combatant, error) {
	row, err := r.queries.GetCampaignCreature(ctx, *e.CreatureID)
	if err != nil {
		return Combatant{}, fmt.Errorf("carregar o bloco de %s: %w", e.Label, err)
	}
	// O BLOCO É DA CAMPANHA, e MEDIDO: sem esta linha, uma linha da fila com um
	// `CreatureID` de outra campanha devolvia a Defesa dela. Um inteiro só, mas
	// atravessando a fronteira entre duas mesas que não se conhecem.
	if row.Campaignid != campaignID {
		return Combatant{}, fmt.Errorf(
			"o bloco de %s não é desta campanha: %w", e.Label, app.ErrForbidden)
	}
	var block creature.Block
	if err := json.Unmarshal([]byte(row.Block), &block); err != nil {
		return Combatant{}, fmt.Errorf("o bloco de %s está ilegível: %w", e.Label, err)
	}
	return Combatant{EntryID: e.ID, Label: e.Label, Defense: block.Defense}, nil
}

// fromBestiary: o bestiário do livro.
func (r Roster) fromBestiary(e live.InitiativeEntry) (Combatant, error) {
	for _, entry := range book.Creatures() {
		if entry.ID == *e.MonsterID {
			return Combatant{EntryID: e.ID, Label: e.Label, Defense: entry.Defense}, nil
		}
	}
	return Combatant{}, fmt.Errorf("o verbete %q não está no bestiário", *e.MonsterID)
}
