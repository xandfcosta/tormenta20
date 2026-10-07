package character

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

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

// BumpCumulativeBonus sobe o bônus dos poderes deste personagem cujos gatilhos
// acabaram de acontecer.
//
// VÁRIOS GATILHOS NUMA CHAMADA porque um golpe confirmado pode ser dois ao
// mesmo tempo — um crítico que também é corpo a corpo —, e duas chamadas
// carregariam a mesma ficha duas vezes. Gatilho que o motor não conhece é
// descartado aqui, em silêncio: quem RECUSA o desconhecido é o guarda do
// catálogo, e recusar de novo em tempo de jogo derrubaria uma confirmação de
// ataque por causa de uma linha de catálogo.
//
// SILENCIOSA quando não há o que subir, e isso é desenho e não descuido: ela é
// chamada em TODA confirmação de ataque da mesa, e quase nenhuma envolve um
// personagem com poder cumulativo. Um erro aqui viraria um ataque recusado por
// causa de um bônus que não existe.
func (p Plays) BumpCumulativeBonus(
	ctx context.Context, characterID int64, triggers ...string,
) error {
	fired := make([]string, 0, len(triggers))
	for _, t := range triggers {
		if book.CumulativeTriggerIsKnown(t) {
			fired = append(fired, t)
		}
	}
	if len(fired) == 0 {
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
	for _, spec := range cumulativesInTheAir(dto, fired) {
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
func cumulativesInTheAir(dto sheet.CharacterDTO, triggers []string) []book.Activation {
	flags := map[string]bool{}
	for _, s := range dto.Stances {
		flags[s.Flag] = true
	}
	outside := []book.Activation{}
	for _, spec := range book.Activations() {
		if spec.Cumulative == nil || !slices.Contains(triggers, spec.Cumulative.On) {
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
	amount, gained, err := p.cumulativeNow(ctx, dto.ID, spec.ID)
	if err != nil {
		return err
	}
	// O TETO É DO QUE SE GANHOU NA CENA, não do que está na poça agora.
	//
	// A distinção só aparece quando o alvo é CONSUMÍVEL: *"você pode ganhar um
	// máximo de PM temporários por cena igual ao seu nível"* (p45) — o bardo
	// que ganhou 4 e gastou 4 não recomeça do zero. Com o teto lido da poça, ele
	// acumularia a cena inteira, dois PM por golpe, sem limite nenhum.
	//
	// Para ataque e dano as duas contagens andam iguais, porque nada as gasta.
	nextGained := book.CumulativeNext(*spec.Cumulative, gained, int(dto.Level))
	step := nextGained - gained
	if step <= 0 {
		return nil
	}
	mods := make([]map[string]any, 0, len(spec.Cumulative.Targets))
	for _, target := range spec.Cumulative.Targets {
		mods = append(mods, map[string]any{
			"target": target,
			"amount": amount + step,
			// GAINED viaja no mesmo mapa e NÃO é campo de modificador: o motor o
			// ignora, e o `withTempAmount` o preserva ao reescrever a poça — ele
			// guarda o mapa cru justamente para não perder campo que não conhece.
			"gained": nextGained,
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
		return fmt.Errorf("gravar o bônus cumulativo de %q em +%d: %w", spec.ID, amount+step, err)
	}
	return nil
}

// cumulativeNow é o que o efeito em curso guarda: quanto ele VALE agora e
// quanto já foi GANHO na cena.
//
// Os dois zerados quando não há linha — é o estado inicial, e não um erro. Os
// valores saem do PRIMEIRO modificador porque todos os alvos do mesmo poder
// sobem juntos: é um bônus só, escrito em dois lugares porque ele move dois
// números.
func (p Plays) cumulativeNow(
	ctx context.Context, characterID int64, catalogID string,
) (amount, gained int, err error) {
	effects, err := p.queries.ListActiveEffectsByCharacter(ctx, characterID)
	if err != nil {
		return 0, 0, fmt.Errorf("ler os efeitos da ficha %d: %w", characterID, err)
	}
	for _, e := range effects {
		if e.Catalogid != catalogID || e.Scope != "scene" {
			continue
		}
		var mods []struct {
			Amount int `json:"amount"`
			Gained int `json:"gained"`
		}
		if json.Unmarshal([]byte(e.Modifiers), &mods) != nil || len(mods) == 0 {
			return 0, 0, nil
		}
		return mods[0].Amount, mods[0].Gained, nil
	}
	return 0, 0, nil
}
