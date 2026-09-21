package sheet

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"t20engine/domain/engine"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// A CONSTRUÇÃO da ficha.
//
// `Load` monta o agregado a partir das linhas do banco; `Compute` passa esse
// agregado pelo motor. As dependências vêm por PARÂMETRO — o que eles usam são
// as `queries` e os `catalogs`, e nada mais.
//
// `LoadAndCompute` é a soma dos dois, para quem só tem a linha do banco. Quem já
// tem o agregado na mão chama o `Compute` direto: a cena de personagens precisa
// da ficha de TODOS de uma vez, e passar pelo caminho completo faria cada herói
// ser lido do banco DUAS vezes.

// loadCharacter anexa as seis relações à linha do personagem, em ordem estável:
// raças, classes, itens e efeitos por id, perícias por nome, magias por
// `learnedAt`.
func Load(
	ctx context.Context, q *sqlcgen.Queries, cat *engine.Catalogs, c sqlcgen.Character,
) (CharacterDTO, error) {
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
			ID: it.ID, CatalogID: dbvalue.NullToPtr(it.Catalogid), Name: it.Name,
			Quantity: it.Quantity, Slots: it.Slots, Equipped: dbvalue.NullToPtr(it.Equipped),
			Improvements: it.Improvements, Material: dbvalue.NullToPtr(it.Material),
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

	// As regras opcionais da mesa entram na ficha AQUI, e num lugar só: tudo o
	// que calcula — a rota da ficha, os PV/PM do nível, o bônus de iniciativa, a
	// ficha inteira que as cenas desenham — passa por este carregamento. Falha de
	// leitura não derruba a ficha: o `IgnoredRules` fica zerado, que significa
	// TODAS as regras em vigor. É o lado seguro, e o único em que um banco mudo
	// não afrouxa regra sem ninguém ver.
	ignored, err := q.ListIgnoredRulesForCharacter(ctx, c.ID)
	if err == nil {
		dto.IgnoredRules = engine.IgnoredRulesFrom(ignored)
	}

	// O estado de JOGO vem junto e nao por rota propria: separado, a ficha abriria
	// com a Furia desligada e a ligaria um instante depois, piscando os numeros
	// que ela muda.
	//
	// Este DERRUBA a carga em caso de falha e o de cima nao, e a diferenca e
	// deliberada: sem o estado de jogo a ficha mente sobre o que esta ligado,
	// enquanto sem as regras opcionais ela cai no padrao do livro, que e o lado
	// seguro. Uma regra a mais nunca inventa numero; uma postura a menos sim.
	if err := LoadPlayState(ctx, q, c.ID, &dto); err != nil {
		return dto, err
	}

	// OS POÇOS SÃO DERIVADOS, e é o último passo de propósito: eles dependem do
	// agregado inteiro — classes, raça, poderes, e os atributos já somados pelos
	// itens que as linhas acima carregaram.
	if err := withDerivedPools(ctx, q, cat, &dto); err != nil {
		return dto, err
	}
	return dto, nil
}

// withDerivedPools troca os poços GRAVADOS pelos derivados, e o atual pelo que
// sobra do dano.
//
// # Por que o máximo não pode vir da coluna
//
// Ele vinha, e era recomputado só em gesto de ESCRITA — nascer, passo de
// atributo, degrau de nível. Um catálogo que mudasse o poço de uma classe não
// alcançava ninguém até o próximo desses gestos, e a própria semente tinha um
// bardo com 31 PM gravados contra 33 derivados: o Carisma total dele subiu de 3
// para 5, e o poder "Magias (1º círculo)" soma o Carisma no total (p44).
//
// Decisão do dono: se o catálogo mudou, as regras do mundo mudaram, logo os
// personagens acompanham (ALE-355).
//
// # E o ATUAL sai de uma subtração, e não de outra coluna
//
// `pvAtual = máximo derivado − dano`. Guardar o atual seria guardar um valor que
// depende de um número derivado; o que é ESTADO é quanto o personagem apanhou.
//
// O que isso compra vem de graça: subir de nível entrega os PV novos já
// preenchidos, e um máximo que ENCOLHE leva o atual junto — as duas regras que o
// `ShiftedByNewMax` e o `ClampedToNewMax` implementavam à mão, e que precisavam
// do máximo ANTIGO para diferenciar.
//
// # O catálogo é obrigatório, e isso é a decisão do arranque
//
// Sem ele não há poço para derivar, e as saídas seriam servir zero PV ou cair na
// coluna velha — a segunda verdade que esta mudança existe para apagar. O
// `cmd/api` já se recusa a subir sem catálogo; aqui a recusa é a mesma, dita
// para quem montar um agregado sem ele.
func withDerivedPools(
	ctx context.Context, q *sqlcgen.Queries, cat *engine.Catalogs, dto *CharacterDTO,
) error {
	if cat == nil {
		return fmt.Errorf("sem catálogo primado não há poço a derivar para a ficha %d", dto.ID)
	}
	ec, err := EngineCharacterFrom(*dto)
	if err != nil {
		return fmt.Errorf("montar o personagem do motor (%d): %w", dto.ID, err)
	}
	pools := cat.VitalsForCharacter(ec)

	// AUSÊNCIA de linha quer dizer INTACTO: só quem apanhou tem registro (00014).
	var damage sqlcgen.GetCharacterDamageRow
	if row, err := q.GetCharacterDamage(ctx, dto.ID); err == nil {
		damage = row
	} else if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("ler o dano da ficha %d: %w", dto.ID, err)
	}

	dto.HpMax, dto.MpMax = int64(pools.PvMax), int64(pools.PmMax)
	// O `WithinPool` e não uma conta própria: `máximo − gasto` preso entre zero e
	// o teto é a MESMA regra que o funil aplica ao gravar, e o piso existe pela
	// mesma razão nos dois — dano que sobreviveu a um máximo que ENCOLHEU. Um
	// personagem com 60 de dano num poço que virou 50 tem zero, não dez
	// negativos.
	dto.HpCurrent = WithinPool(dto.HpMax-damage.Hpdamage, dto.HpMax)
	dto.MpCurrent = WithinPool(dto.MpMax-damage.Mpspent, dto.MpMax)
	return nil
}

// LoadAndCompute monta o agregado a partir da linha e o passa pelo motor.
//
// A ficha sai COM os situacionais que o jogador ligou, porque o `Compute` os tira
// do agregado. Este bloco dizia "ficha base, sem condicional ligada" e deixou de
// ser verdade na ALE-357 — que é quando o crachá do topo da ficha parou de
// discordar da aba Combate. Os catálogos têm de estar primados.
func LoadAndCompute(ctx context.Context, q *sqlcgen.Queries, cat *engine.Catalogs, row sqlcgen.Character) (engine.ComputedSheet, error) {
	dto, err := Load(ctx, q, cat, row)
	if err != nil {
		return engine.ComputedSheet{}, err
	}
	return Compute(cat, dto)
}

// engineCharacterFrom leva o agregado até o `engine.Character` por JSON: os dois
// têm a mesma forma, então a ida e volta não perde nada.
func EngineCharacterFrom(dto CharacterDTO) (engine.Character, error) {
	var ec engine.Character
	b, err := json.Marshal(dto)
	if err != nil {
		return ec, err
	}
	return ec, json.Unmarshal(b, &ec)
}

// Compute computa a ficha de um agregado JÁ CARREGADO.
//
// Separado do `computeSheet` para a cena de personagens, que precisa da ficha de
// TODOS de uma vez: ela já tem os agregados na mão, e passar por `computeSheet`
// faria cada personagem ser lido do banco DUAS vezes — uma na lista e outra
// dentro dele.
// # OS CONDICIONAIS DO JOGADOR ENTRAM, e é o agregado que os traz
//
// Aqui passava-se `map[string]bool{}` — um conjunto vazio inventado por quem
// computa. O `dto.Conditionals` é o opt-in que o jogador LIGOU e que já vem
// carregado pelo `Load`; ignorá-lo fazia a mesma ficha ter dois números.
//
// Medido: o crachá do topo da ficha dizia Defesa 12 e a aba Combate, 17 — a aba
// passa os condicionais de verdade (`sheetui.sheetForPanels`) e o crachá vinha
// por aqui. Nos oráculos, a Lenda de nível 20 erra por DEZ pontos de Defesa
// (ALE-357).
//
// São sete chamadores pelo `LoadAndCompute`, e entre eles a Defesa do cartão do
// Grupo na Mesa — que é a tela que o cabeçalho do `sheetForPanels` citava como
// aquela de quem a ficha não podia discordar.
func Compute(cat *engine.Catalogs, dto CharacterDTO) (engine.ComputedSheet, error) {
	ec, err := EngineCharacterFrom(dto)
	if err != nil {
		return engine.ComputedSheet{}, err
	}
	return cat.ComputeSheet(ec, ToStringSet(dto.Conditionals)), nil
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
