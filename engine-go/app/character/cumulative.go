package character

import (
	"context"
	"encoding/json"
	"fmt"

	"t20engine/domain/book"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// O BÔNUS CUMULATIVO DE CENA (p42).
//
// O catálogo declara a REGRA de crescimento (`book.CumulativeBonus`); o valor
// de agora é estado de jogo e mora num EFEITO ATIVO de escopo `scene`, como
// toda outra coisa que mexe num número por um tempo.
//
// POR QUE UM EFEITO ATIVO e não uma coluna nova: o caminho inteiro já existe e
// já funciona. O efeito carrega os próprios modificadores em JSON, então o +N
// chega ao ataque e ao dano pelo mesmo cano que uma poção; o `UNIQUE
// (characterId, catalogId, scope)` faz do incremento um upsert; a aba Efeitos
// já desenha o que está em curso; e o `DeleteEffectsByScope('scene')` do
// encerrar cena já o varre. Uma coluna nova teria de reaprender as quatro
// coisas.

// BumpCumulativeBonus sobe o bônus dos poderes deste personagem cujo gatilho
// acabou de acontecer.
//
// SILENCIOSA quando não há o que subir, e isso é desenho e não descuido: ela é
// chamada em TODA confirmação de ataque da mesa, e quase nenhuma envolve um
// personagem com poder cumulativo. Um erro aqui viraria um ataque recusado por
// causa de um bônus que não existe.
func (p Plays) BumpCumulativeBonus(ctx context.Context, characterID int64, trigger string) error {
	if !book.CumulativeTriggerIsKnown(trigger) {
		return nil
	}
	row, err := p.queries.GetCharacter(ctx, characterID)
	if err != nil {
		return fmt.Errorf("carregar o personagem %d para subir o bônus cumulativo: %w", characterID, err)
	}
	dto, err := sheet.Load(ctx, p.queries, p.catalogs, row)
	if err != nil {
		return fmt.Errorf("montar a ficha %d para subir o bônus cumulativo: %w", characterID, err)
	}
	for _, spec := range cumulativesInTheAir(dto, trigger) {
		if err := p.raiseOne(ctx, dto, spec); err != nil {
			return err
		}
	}
	return nil
}

// cumulativesInTheAir são os poderes cumulativos que ESTE personagem tem, com a
// postura deles acesa e o gatilho casando.
//
// AS TRÊS CONDIÇÕES JUNTAS, e nenhuma é dispensável: sem a posse, todo bárbaro
// acumularia; sem a flag, a Sangue dos Inimigos valeria fora da fúria, que é
// metade do poder; sem o gatilho, um crítico subiria o bônus de um poder que
// só acumula ao derrubar.
func cumulativesInTheAir(dto sheet.CharacterDTO, trigger string) []book.Activation {
	flags := map[string]bool{}
	for _, s := range dto.Stances {
		flags[s.Flag] = true
	}
	outside := []book.Activation{}
	for _, spec := range book.Activations() {
		if spec.Cumulative == nil || spec.Cumulative.On != trigger {
			continue
		}
		if spec.RequiresFlag != "" && !flags[spec.RequiresFlag] {
			continue
		}
		if !sheet.OwnsClassPower(dto, spec.ID) {
			continue
		}
		outside = append(outside, spec)
	}
	return outside
}

// raiseOne sobe UM poder, e não escreve nada quando já está no teto.
//
// NÃO ESCREVER NO TETO é o que mantém a aba Efeitos honesta: um upsert por
// crítico com o mesmo valor carimbaria o `createdAt` de novo, e a linha
// pareceria recém-nascida a cada golpe de um bárbaro que já parou de crescer.
func (p Plays) raiseOne(ctx context.Context, dto sheet.CharacterDTO, spec book.Activation) error {
	current, err := p.cumulativeNow(ctx, dto.ID, spec.ID)
	if err != nil {
		return err
	}
	next := book.CumulativeNext(*spec.Cumulative, current, int(dto.Level))
	if next <= current {
		return nil
	}
	mods := make([]map[string]any, 0, len(spec.Cumulative.Targets))
	for _, target := range spec.Cumulative.Targets {
		mods = append(mods, map[string]any{
			"target": target,
			"amount": next,
			// SEM TIPO, e o livro é quem decide: a p42 não dá tipo a este bônus,
			// e bônus sem tipo SOMA com os outros (p105). Marcá-lo `morale` o
			// faria disputar com o +3 da própria Fúria, e o maior venceria —
			// o bárbaro acumularia até +3 sem ver número nenhum se mexer.
			"bonusType": "untyped",
			// SEM NOTA: a linha da aba Efeitos já se chama "Sangue dos
			// Inimigos" (o `effectDisplayName` a resolve pela ativação), e cada
			// modificador já diz o alvo dele — "Ataque +2", "Dano +2". Repetir
			// o nome do poder em cada sub-linha seria a terceira vez que a
			// mesma palavra aparece na mesma caixa.
			"note": "",
		})
	}
	blob, err := json.Marshal(mods)
	if err != nil {
		return fmt.Errorf("escrever o bônus cumulativo de %q: %w", spec.ID, err)
	}
	if _, err := p.queries.UpsertActiveEffect(ctx, sqlcgen.UpsertActiveEffectParams{
		Characterid: dto.ID, Source: "power", Catalogid: spec.ID, Scope: "scene",
		Modifiers: string(blob), Createdat: dbvalue.NowISO(),
	}); err != nil {
		return fmt.Errorf("gravar o bônus cumulativo de %q em +%d: %w", spec.ID, next, err)
	}
	return nil
}

// cumulativeNow é quanto o bônus vale AGORA, lido do efeito em curso.
//
// Zero quando não há linha — é o estado inicial, e não um erro. O valor sai do
// PRIMEIRO modificador porque todos os alvos do mesmo poder sobem juntos: é um
// bônus só, escrito em dois lugares porque ele move dois números.
func (p Plays) cumulativeNow(ctx context.Context, characterID int64, catalogID string) (int, error) {
	effects, err := p.queries.ListActiveEffectsByCharacter(ctx, characterID)
	if err != nil {
		return 0, fmt.Errorf("ler os efeitos da ficha %d: %w", characterID, err)
	}
	for _, e := range effects {
		if e.Catalogid != catalogID || e.Scope != "scene" {
			continue
		}
		var mods []struct {
			Amount int `json:"amount"`
		}
		if json.Unmarshal([]byte(e.Modifiers), &mods) != nil || len(mods) == 0 {
			return 0, nil
		}
		return mods[0].Amount, nil
	}
	return 0, nil
}
