package sheet

import (
	"context"
	"fmt"

	"t20engine/domain/engine"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// OS POÇOS DE UM GRUPO INTEIRO, em seis consultas.
//
// # Por que existe um caminho em lote
//
// O `RefreshCharacterVitals` roda a cada desenho da Mesa, inclusive a cada tique
// do fluxo. Enquanto o máximo era coluna, ele era UMA consulta — `SELECT id,
// hpMax, mpMax FROM characters WHERE id IN (…)`. Com o máximo derivado, chamar
// o `Load` por personagem daria nove consultas cada um: numa mesa de seis, 54
// por desenho. O cálculo em si não custa nada (uma ficha de nível 20 com tudo
// ligado computa em 17 µs); o que custa é ir ao banco.
//
// # O que o poço precisa, e o que ele NÃO precisa
//
// O `engine.VitalContextFor` lê os escalares da linha e QUATRO relações: as
// classes (o poço é por nível de classe), a raça, os itens e os efeitos ativos —
// as duas últimas porque o `ActiveItemsFor` dobra efeito com modificador em
// item, e um bônus de Constituição de qualquer origem mexe no PV. Perícias,
// magias, regras ignoradas e estado de jogo não entram em poço nenhum, e é o que
// deixa este caminho ser curto: a própria `engine.Character` só tem cinco
// relações, e a quinta (perícias) não chega ao contexto vital.
//
// Isso é uma SEGUNDA maneira de chegar ao mesmo número, que é a família de
// defeito que a ALE-355 inteira existe para apagar. O que impede as duas de
// divergirem é o `TestTheBatchPoolsMatchTheOneByOnePools`: para cada personagem
// da semente, o lote tem de dar o mesmo que o `Load`. **Ele não é opcional** —
// foi ele que apontou o quarto carregamento acima, que este cabeçalho já dizia
// ser desnecessário.

// PoolsForCharacters deriva o poço de cada id pedido.
//
// Personagem que não existe simplesmente não aparece no mapa — quem chama já
// tem de lidar com isso, porque a lista de ids vem de uma mesa onde uma ficha
// pode ter sido apagada.
func PoolsForCharacters(
	ctx context.Context, q *sqlcgen.Queries, cat *engine.Catalogs, ids []int64,
) (map[int64]Pools, error) {
	if len(ids) == 0 {
		return map[int64]Pools{}, nil
	}
	if cat == nil {
		return nil, fmt.Errorf("sem catálogo primado não há poço a derivar para %d fichas", len(ids))
	}
	partials, err := partialSheetsForPools(ctx, q, ids)
	if err != nil {
		return nil, err
	}
	damages, err := q.ListCharacterDamage(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("ler o dano de %d fichas: %w", len(ids), err)
	}
	damage := make(map[int64]sqlcgen.CharacterDamage, len(damages))
	for _, d := range damages {
		damage[d.Characterid] = d
	}

	pools := make(map[int64]Pools, len(partials))
	for id, dto := range partials {
		ec, err := EngineCharacterFrom(dto)
		if err != nil {
			return nil, fmt.Errorf("montar o personagem do motor (%d): %w", id, err)
		}
		derived := cat.VitalsForCharacter(ec)
		p := Pools{HpMax: int64(derived.PvMax), MpMax: int64(derived.PmMax)}
		p.HpCurrent = WithinHitPoints(p.HpMax-damage[id].Hpdamage, p.HpMax)
		p.MpCurrent = WithinPool(p.MpMax-damage[id].Mpspent, p.MpMax)
		pools[id] = p
	}
	return pools, nil
}

// partialSheetsForPools monta o agregado MÍNIMO de cada ficha em cinco
// consultas: a linha, as raças, as classes, os itens e os efeitos ativos.
//
// Ele devolve um `CharacterDTO` e não uma estrutura própria de propósito: o
// `EngineCharacterFrom` passa por JSON, e usar o mesmo tipo faz o lote entrar no
// motor pela MESMA porta que a ficha inteira. Uma estrutura paralela seria a
// segunda tradução, e ela divergiria no dia em que um campo novo entrasse.
func partialSheetsForPools(
	ctx context.Context, q *sqlcgen.Queries, ids []int64,
) (map[int64]CharacterDTO, error) {
	rows, err := q.ListCharactersByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("ler %d fichas: %w", len(ids), err)
	}
	partials := make(map[int64]CharacterDTO, len(rows))
	for _, c := range rows {
		partials[c.ID] = CharacterScalarsFrom(c)
	}

	races, err := q.ListRacesByCharacters(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("ler as raças de %d fichas: %w", len(ids), err)
	}
	for _, r := range races {
		dto := partials[r.Characterid]
		dto.Races = append(dto.Races, RaceDTO{Race: r.Race})
		partials[r.Characterid] = dto
	}

	classes, err := q.ListClassesByCharacters(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("ler as classes de %d fichas: %w", len(ids), err)
	}
	for _, cl := range classes {
		dto := partials[cl.Characterid]
		dto.Classes = append(dto.Classes, ClassDTO{ClassName: cl.Classname, Level: cl.Level})
		partials[cl.Characterid] = dto
	}

	items, err := q.ListItemsByCharacters(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("ler os itens de %d fichas: %w", len(ids), err)
	}
	for _, it := range items {
		dto := partials[it.Characterid]
		dto.Items = append(dto.Items, ItemDTO{
			ID: it.ID, CatalogID: dbvalue.NullToPtr(it.Catalogid), Name: it.Name,
			Quantity: it.Quantity, Slots: it.Slots, Equipped: dbvalue.NullToPtr(it.Equipped),
			Improvements: it.Improvements, Material: dbvalue.NullToPtr(it.Material),
		})
		partials[it.Characterid] = dto
	}

	effects, err := q.ListActiveEffectsByCharacters(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("ler os efeitos de %d fichas: %w", len(ids), err)
	}
	for _, ef := range effects {
		dto := partials[ef.Characterid]
		dto.ActiveEffects = append(dto.ActiveEffects, EffectDTO{
			ID: ef.ID, CatalogID: ef.Catalogid, Scope: ef.Scope,
			Modifiers: ef.Modifiers, CreatedAt: ef.Createdat,
		})
		partials[ef.Characterid] = dto
	}
	return partials, nil
}
