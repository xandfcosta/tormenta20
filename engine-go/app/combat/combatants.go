package combat

import (
	"context"
	"encoding/json"
	"fmt"

	"t20engine/domain/book"
	"t20engine/domain/creature"
	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// Roster traduz uma linha da queueDouble no combatente que a regra precisa.
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

func (r Roster) Of(ctx context.Context, e live.InitiativeEntry) (Combatant, error) {
	switch {
	case e.CharacterID != nil:
		return r.fromSheet(ctx, e)
	case e.CreatureID != nil:
		return r.fromBlock(ctx, e)
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
	computada, err := sheet.Compute(r.catalogs, dto)
	if err != nil {
		return Combatant{}, fmt.Errorf("computar a ficha de %s: %w", e.Label, err)
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		return Combatant{}, err
	}
	return Combatant{
		EntryID: e.ID, Label: e.Label,
		Weapons:         r.catalogs.ComputeWeaponCards(ec, sheet.ToStringSet(dto.Conditionals)),
		Defense:         computada.Defense.Total,
		DamageReduction: computada.DamageReduction.Total,
	}, nil
}

// fromBlock: o NPC que o mestre escreveu. Ele não empunha arma do catálogo — o
// ataque dele é texto no bloco —, então serve de ALVO e não de atacante.
func (r Roster) fromBlock(ctx context.Context, e live.InitiativeEntry) (Combatant, error) {
	row, err := r.queries.GetCampaignCreature(ctx, *e.CreatureID)
	if err != nil {
		return Combatant{}, fmt.Errorf("carregar o bloco de %s: %w", e.Label, err)
	}
	var bloco creature.Block
	if err := json.Unmarshal([]byte(row.Block), &bloco); err != nil {
		return Combatant{}, fmt.Errorf("o bloco de %s está ilegível: %w", e.Label, err)
	}
	return Combatant{EntryID: e.ID, Label: e.Label, Defense: bloco.Defense}, nil
}

// fromBestiary: o bestiário do livro.
func (r Roster) fromBestiary(e live.InitiativeEntry) (Combatant, error) {
	for _, verbete := range book.Creatures() {
		if verbete.ID == *e.MonsterID {
			return Combatant{EntryID: e.ID, Label: e.Label, Defense: verbete.Defense}, nil
		}
	}
	return Combatant{}, fmt.Errorf("o verbete %q não está no bestiário", *e.MonsterID)
}
