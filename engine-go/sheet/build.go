package sheet

import (
	"context"
	"encoding/json"

	"t20engine/db/sqlcgen"
	"t20engine/engine"
	"t20engine/plataforma"
)

// A CONSTRUÇÃO da ficha (ALE-278, terceira camada compartilhada).
//
// `Load` monta o agregado a partir das linhas do banco; `Compute` passa esse
// agregado pelo motor. Os dois eram métodos do `api.Server` e viraram funções
// com as dependências por PARÂMETRO — o que elas usavam dele eram as `queries` e
// os `catalogs`, e nada mais.
//
// `LoadAndCompute` é a soma dos dois e existe porque a maioria dos chamadores só
// tem a linha do banco. Quem já tem o agregado na mão chama o `Compute` direto:
// a cena de personagens precisa da ficha de TODOS de uma vez, e passar pelo
// caminho completo faria cada herói ser lido do banco DUAS vezes (ALE-239).

// loadCharacter attaches the six relations to a character row in the Prisma
// include order (races/classes/items/effects by id, expertises by name, spells by
// learnedAt).
func Load(ctx context.Context, q *sqlcgen.Queries, c sqlcgen.Character) (CharacterDTO, error) {
	dto := CharacterScalarsFrom(c)

	races, err := q.ListRacesByCharacter(ctx, c.ID)
	if err != nil {
		return dto, err
	}
	for _, race := range races {
		dto.Races = append(dto.Races, RaceDTO{Race: race})
	}

	classes, err := q.ListClassesByCharacter(ctx, c.ID)
	if err != nil {
		return dto, err
	}
	for _, cl := range classes {
		dto.Classes = append(dto.Classes, ClassDTO{ClassName: cl.Classname, Level: cl.Level})
	}

	exps, err := q.ListExpertisesByCharacter(ctx, c.ID)
	if err != nil {
		return dto, err
	}
	for _, e := range exps {
		dto.Expertises = append(dto.Expertises, ExpertiseDTO{
			Name: e.Name, Attribute: e.Attribute, Trained: e.Trained != 0, Custom: e.Custom != 0,
		})
	}

	items, err := q.ListItemsByCharacter(ctx, c.ID)
	if err != nil {
		return dto, err
	}
	for _, it := range items {
		dto.Items = append(dto.Items, ItemDTO{
			ID: it.ID, CatalogID: plataforma.NullToPtr(it.Catalogid), Name: it.Name,
			Quantity: it.Quantity, Slots: it.Slots, Equipped: plataforma.NullToPtr(it.Equipped),
			Improvements: it.Improvements, Material: plataforma.NullToPtr(it.Material),
		})
	}

	effects, err := q.ListActiveEffectsByCharacter(ctx, c.ID)
	if err != nil {
		return dto, err
	}
	for _, ef := range effects {
		dto.ActiveEffects = append(dto.ActiveEffects, EffectDTO{
			ID: ef.ID, CatalogID: ef.Catalogid, Scope: ef.Scope,
			Modifiers: ef.Modifiers, CreatedAt: ef.Createdat,
		})
	}

	spells, err := q.ListSpellsByCharacter(ctx, c.ID)
	if err != nil {
		return dto, err
	}
	for _, sp := range spells {
		dto.Spells = append(dto.Spells, SpellDTO{
			ID: sp.ID, CatalogSpellID: sp.Catalogspellid, Prepared: sp.Prepared != 0, LearnedAt: sp.Learnedat,
		})
	}

	// As regras opcionais da mesa entram na ficha AQUI, e num lugar só (ALE-221):
	// tudo o que calcula — o `GET /sheet`, os PV/PM do nível, o bônus de
	// iniciativa, a ficha que o navegador recalcula no WASM — passa por este
	// carregamento. Falha de leitura não derruba a ficha: o `IgnoredRules` fica
	// zerado, que significa TODAS as regras em vigor. É o lado seguro, e o único
	// em que um banco mudo não afrouxa regra sem ninguém ver.
	ignored, err := q.ListIgnoredRulesForCharacter(ctx, c.ID)
	if err == nil {
		dto.IgnoredRules = engine.IgnoredRulesFrom(ignored)
	}

	// O estado de JOGO (ALE-222). Vem junto e nao por endpoint proprio: separado,
	// a ficha abriria com a Furia desligada e a ligaria um instante depois,
	// piscando os numeros que ela muda.
	//
	// Este DERRUBA a carga em caso de falha e o de cima nao, e a diferenca e
	// deliberada: sem o estado de jogo a ficha mente sobre o que esta ligado,
	// enquanto sem as regras opcionais ela cai no padrao do livro, que e o lado
	// seguro. Uma regra a mais nunca inventa numero; uma postura a menos sim.
	if err := LoadPlayState(ctx, q, c.ID, &dto); err != nil {
		return dto, err
	}
	return dto, nil
}

// computeSheet builds the engine input from an already-loaded character row and returns the
// server-computed ComputedSheetV2 (base sheet, no active conditionals). Shared by GET /sheet
// and the power-grant temp-HP amount so the Load→engine→compute wiring lives in one place.
// Caller must ensure s.catalogs is primed.
func LoadAndCompute(ctx context.Context, q *sqlcgen.Queries, cat *engine.Catalogs, row sqlcgen.Character) (engine.ComputedSheetV2, error) {
	dto, err := Load(ctx, q, row)
	if err != nil {
		return engine.ComputedSheetV2{}, err
	}
	return Compute(cat, dto)
}

// engineCharacterFrom bridges the API aggregate to engine.Character via JSON —
// both mirror the frontend Character contract, so the round-trip is lossless.
func EngineCharacterFrom(dto CharacterDTO) (engine.Character, error) {
	var ec engine.Character
	b, err := json.Marshal(dto)
	if err != nil {
		return ec, err
	}
	return ec, json.Unmarshal(b, &ec)
}

// sheetFromDTO computa a ficha de um agregado JÁ CARREGADO.
//
// Separado do `computeSheet` para a cena de personagens (ALE-239), que precisa
// da ficha de TODOS de uma vez: ela já tem os agregados na mão, e passar por
// `computeSheet` faria cada personagem ser lido do banco DUAS vezes — uma na
// lista e outra dentro dele. Com uma dúzia de heróis isso é o dobro das
// consultas para o mesmo resultado.
func Compute(cat *engine.Catalogs, dto CharacterDTO) (engine.ComputedSheetV2, error) {
	ec, err := EngineCharacterFrom(dto)
	if err != nil {
		return engine.ComputedSheetV2{}, err
	}
	return cat.ComputeSheetV2(ec, map[string]bool{}), nil
}

// loadPlayState anexa os três ao DTO da ficha.
//
// Vai JUNTO com a ficha em vez de num endpoint próprio porque a tela precisa dos
// três para desenhar o primeiro quadro: separados, a ficha abriria com a Fúria
// desligada e a ligaria um instante depois, piscando os números que ela muda.
func LoadPlayState(ctx context.Context, q *sqlcgen.Queries, id int64, dto *CharacterDTO) error {
	conditionals, err := q.ListCharacterConditionals(ctx, id)
	if err != nil {
		return err
	}
	dto.Conditionals = conditionals

	uses, err := q.ListCharacterPowerUses(ctx, id)
	if err != nil {
		return err
	}
	for _, u := range uses {
		dto.PowerUses = append(dto.PowerUses, PowerUseDTO{PowerID: u.Powerid, Scope: u.Scope, Used: u.Used})
	}

	stances, err := q.ListCharacterStances(ctx, id)
	if err != nil {
		return err
	}
	for _, st := range stances {
		dto.Stances = append(dto.Stances, StanceDTO{Flag: st.Flag, Steps: st.Steps, PmPaid: st.Pmpaid})
	}
	return nil
}
