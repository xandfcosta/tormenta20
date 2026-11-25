package character

import (
	"context"
	"encoding/json"
	"fmt"

	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// spellEffectWrite é o que um gesto pede ao efeito da magia na ficha (ver GLOSSARY, **efeito da magia**).
type spellEffectWrite struct {
	// Scope é a duração que o gesto dá, na grafia da coluna: a da conjuração no Conjurar, a escolhida no Aplicar.
	Scope string
	// Modifiers é o bônus que o Aplicar traz; vazio mantém o que a linha já tem, e a linha nova abre sem nenhum.
	Modifiers string
	// Cast é o Conjurar desta ficha, que grava junto os aprimoramentos escolhidos (Augments): é a marca de que a
	// ficha sustenta a linha (`EffectDTO.Sustaining`), e o que o dado da vez e a ação da magia leem.
	Cast     bool
	Augments []sheet.AugmentPick
}

// writeSpellEffect é a ÚNICA escrita do efeito de uma magia numa ficha, e devolve o id da linha.
//
// A ficha tem UMA linha por magia. O gesto mais recente diz a duração, e os modificadores gravados ficam quando ele
// não traz nenhum: o Conjurar não apaga o bônus que o Aplicar gravou, em qualquer ordem. A linha que já tem a
// duração pedida é a escolhida; senão, a mais antiga da magia. Isso também evita a chave única (personagem,
// magia, duração) da tabela quando uma ficha gravada antes desta regra ainda tem duas.
//
// Quem a chama pela transação passa o `q` dela: a conjuração cobra o PM na mesma.
func writeSpellEffect(
	ctx context.Context, q *sqlcgen.Queries, characterID int64, spellID string, w spellEffectWrite,
) (int64, error) {
	effects, err := q.ListActiveEffectsByCharacter(ctx, characterID)
	if err != nil {
		return 0, fmt.Errorf("ler os efeitos da ficha %d: %w", characterID, err)
	}
	var found *sqlcgen.ListActiveEffectsByCharacterRow
	for i, e := range effects {
		if e.Catalogid != spellID {
			continue
		}
		if found == nil || e.Scope == w.Scope {
			found = &effects[i]
		}
		if e.Scope == w.Scope {
			break
		}
	}
	var effectID int64
	if found == nil {
		modifiers := w.Modifiers
		if modifiers == "" {
			modifiers = "[]"
		}
		opened, err := q.UpsertActiveEffect(ctx, sqlcgen.UpsertActiveEffectParams{
			Characterid: characterID, Source: "spell", Catalogid: spellID, Scope: w.Scope,
			Modifiers: modifiers, Createdat: dbvalue.NowISO(),
		})
		if err != nil {
			return 0, fmt.Errorf("abrir %q (%s) na ficha %d: %w", spellID, w.Scope, characterID, err)
		}
		effectID = opened.ID
	} else {
		modifiers := w.Modifiers
		if modifiers == "" {
			modifiers = found.Modifiers
		}
		if err := q.UpdateActiveEffect(ctx, sqlcgen.UpdateActiveEffectParams{
			Scope: w.Scope, Modifiers: modifiers, ID: found.ID,
		}); err != nil {
			return 0, fmt.Errorf("gravar %q (%s) na ficha %d: %w", spellID, w.Scope, characterID, err)
		}
		effectID = found.ID
	}
	if !w.Cast {
		return effectID, nil
	}
	augments := w.Augments
	if augments == nil {
		augments = []sheet.AugmentPick{}
	}
	written, err := json.Marshal(augments)
	if err != nil {
		return 0, err
	}
	if err := q.SaveEffectCastAugments(ctx, sqlcgen.SaveEffectCastAugmentsParams{Effectid: effectID, Augments: string(written)}); err != nil {
		return 0, fmt.Errorf("gravar a conjuração do efeito %d: %w", effectID, err)
	}
	return effectID, nil
}
