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
// O `RefreshCharacterMaxes` roda a cada desenho da Mesa, inclusive a cada tique
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
	parciais, err := partialSheetsForPools(ctx, q, ids)
	if err != nil {
		return nil, err
	}
	danos, err := q.ListCharacterDamage(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("ler o dano de %d fichas: %w", len(ids), err)
	}
	dano := make(map[int64]sqlcgen.CharacterDamage, len(danos))
	for _, d := range danos {
		dano[d.Characterid] = d
	}

	pocos := make(map[int64]Pools, len(parciais))
	for id, dto := range parciais {
		ec, err := EngineCharacterFrom(dto)
		if err != nil {
			return nil, fmt.Errorf("montar o personagem do motor (%d): %w", id, err)
		}
		derivado := cat.VitalsForCharacter(ec)
		p := Pools{HpMax: int64(derivado.PvMax), MpMax: int64(derivado.PmMax)}
		p.HpCurrent = WithinPool(p.HpMax-dano[id].Hpdamage, p.HpMax)
		p.MpCurrent = WithinPool(p.MpMax-dano[id].Mpspent, p.MpMax)
		pocos[id] = p
	}
	return pocos, nil
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
	linhas, err := q.ListCharactersByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("ler %d fichas: %w", len(ids), err)
	}
	parciais := make(map[int64]CharacterDTO, len(linhas))
	for _, c := range linhas {
		parciais[c.ID] = CharacterScalarsFrom(c)
	}

	racas, err := q.ListRacesByCharacters(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("ler as raças de %d fichas: %w", len(ids), err)
	}
	for _, r := range racas {
		dto := parciais[r.Characterid]
		dto.Races = append(dto.Races, RaceDTO{Race: r.Race})
		parciais[r.Characterid] = dto
	}

	classes, err := q.ListClassesByCharacters(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("ler as classes de %d fichas: %w", len(ids), err)
	}
	for _, cl := range classes {
		dto := parciais[cl.Characterid]
		dto.Classes = append(dto.Classes, ClassDTO{ClassName: cl.Classname, Level: cl.Level})
		parciais[cl.Characterid] = dto
	}

	itens, err := q.ListItemsByCharacters(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("ler os itens de %d fichas: %w", len(ids), err)
	}
	for _, it := range itens {
		dto := parciais[it.Characterid]
		dto.Items = append(dto.Items, ItemDTO{
			ID: it.ID, CatalogID: dbvalue.NullToPtr(it.Catalogid), Name: it.Name,
			Quantity: it.Quantity, Slots: it.Slots, Equipped: dbvalue.NullToPtr(it.Equipped),
			Improvements: it.Improvements, Material: dbvalue.NullToPtr(it.Material),
		})
		parciais[it.Characterid] = dto
	}

	efeitos, err := q.ListActiveEffectsByCharacters(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("ler os efeitos de %d fichas: %w", len(ids), err)
	}
	for _, ef := range efeitos {
		dto := parciais[ef.Characterid]
		dto.ActiveEffects = append(dto.ActiveEffects, EffectDTO{
			ID: ef.ID, CatalogID: ef.Catalogid, Scope: ef.Scope,
			Modifiers: ef.Modifiers, CreatedAt: ef.Createdat,
		})
		parciais[ef.Characterid] = dto
	}
	return parciais, nil
}
