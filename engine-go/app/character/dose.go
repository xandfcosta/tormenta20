package character

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"t20engine/domain/catalog"
	"t20engine/domain/sheet"
	"t20engine/infra/db"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// ErrDailyPortion é a recusa da porção diária. Ela é RECONHECÍVEL em vez de
// virar texto porque a dose tem dois caminhos para ela — o efeito já na ficha e
// a colisão do UNIQUE no meio da transação — e quem chama precisa saber que os
// dois são a mesma recusa.
var ErrDailyPortion = errors.New("apenas uma porção por dia")

// Dose é o que uma dose consumida MUDOU: o item, o efeito que ela deixou e os
// poços depois dela.
type Dose struct {
	Name      string
	ItemID    int64
	Quantity  int64
	Removed   bool
	Effect    *sheet.EffectDTO
	HpCurrent int64
	MpCurrent int64
}

// Consume é a dose INTEIRA: a rolagem imediata presa no máximo, a linha de
// efeito de cena ou dia, e a baixa do item — tudo numa transação.
//
// Ela tem dois chamadores, e é por isso que mora aqui: a Mochila da ficha e o
// gerador da seed. Reescrita numa delas, daria DUAS respostas para "posso beber
// esta poção?". É a mesma razão do `Cast`.
//
// A TRANSAÇÃO é o caso inteiro desta fatia: três escritas — o efeito, o item e
// os poços — que ou acontecem juntas ou não acontecem. Meia dose gravada é uma
// poção que sumiu do inventário sem curar ninguém.
func (p Plays) Consume(
	ctx context.Context, row sqlcgen.Character, itemID int64, hpRolled, mpRolled *int64,
) (Dose, error) {
	dto, err := sheet.Load(ctx, p.queries, p.catalogs, row)
	if err != nil {
		return Dose{}, err
	}
	item := findItemDTO(dto.Items, itemID)
	if item == nil {
		return Dose{}, fmt.Errorf("o item %d não está nesta ficha", itemID)
	}
	if item.CatalogID == nil {
		return Dose{}, fmt.Errorf("%q é um item custom e não tem o que usar", item.Name)
	}
	cat, known := catalog.LookupItem(*item.CatalogID)
	if !known || cat.Consumable == nil {
		return Dose{}, fmt.Errorf("%q não é um consumível", item.Name)
	}
	if item.Quantity < 1 {
		return Dose{}, fmt.Errorf("não sobrou nenhum uso de %q", item.Name)
	}
	spec := cat.Consumable
	if spec.OncePerDay {
		for _, e := range dto.ActiveEffects {
			if e.CatalogID == cat.ID {
				return Dose{Name: cat.Name}, ErrDailyPortion
			}
		}
	}

	hpGain, hasHp := rollGain(hpRolled, spec.Instant, true)
	mpGain, hasMp := rollGain(mpRolled, spec.Instant, false)

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return Dose{}, fmt.Errorf("abrir a transação da dose de %q: %w", cat.Name, err)
	}
	defer func() { _ = tx.Rollback() }()
	q := p.queries.WithTx(tx)
	now := dbvalue.NowISO()

	var effect *sheet.EffectDTO
	if wantsEffectRow(spec) {
		eff, err := q.CreateActiveEffect(ctx, sqlcgen.CreateActiveEffectParams{
			Characterid: row.ID, Catalogid: cat.ID, Scope: spec.Scope,
			Modifiers: effectModifiers(spec.Modifiers), Createdat: now,
		})
		if db.IsUniqueViolation(err) {
			return Dose{Name: cat.Name}, ErrDailyPortion
		}
		if err != nil {
			return Dose{}, fmt.Errorf("gravar o efeito de %q: %w", cat.Name, err)
		}
		effect = &sheet.EffectDTO{
			ID: eff.ID, CatalogID: eff.Catalogid, Scope: eff.Scope,
			Modifiers: eff.Modifiers, CreatedAt: eff.Createdat,
		}
	}

	removed := false
	newQty := item.Quantity - 1
	if item.Quantity > 1 {
		if err := q.SetItemQuantity(ctx, sqlcgen.SetItemQuantityParams{Quantity: newQty, ID: itemID}); err != nil {
			return Dose{}, fmt.Errorf("dar baixa numa dose de %q: %w", cat.Name, err)
		}
	} else {
		if err := q.DeleteItem(ctx, itemID); err != nil {
			return Dose{}, fmt.Errorf("tirar %q da mochila: %w", cat.Name, err)
		}
		removed, newQty = true, 0
	}

	// O poço tem de sair do funil mesmo quando a dose não cura nada: ele é o que
	// a cena redesenha, e devolver a coluna crua aqui traria o número velho.
	pools, err := sheet.ApplyToPools(ctx, q, p.catalogs, row,
		func(poolMap sheet.Pools) (sheet.Pools, error) {
			if hasHp {
				poolMap.HpCurrent += int64(hpGain)
			}
			if hasMp {
				poolMap.MpCurrent += int64(mpGain)
			}
			return poolMap, nil
		})
	if err != nil {
		return Dose{}, fmt.Errorf("gravar os poços da ficha %d: %w", row.ID, err)
	}
	hpCurrent, mpCurrent := pools.HpCurrent, pools.MpCurrent
	if err := tx.Commit(); err != nil {
		return Dose{}, fmt.Errorf("fechar a transação da dose de %q: %w", cat.Name, err)
	}
	return Dose{
		Name: cat.Name, ItemID: itemID, Quantity: newQty, Removed: removed,
		Effect: effect, HpCurrent: hpCurrent, MpCurrent: mpCurrent,
	}, nil
}

func findItemDTO(items []sheet.ItemDTO, itemID int64) *sheet.ItemDTO {
	for i := range items {
		if items[i].ID == itemID {
			return &items[i]
		}
	}
	return nil
}

func rollGain(rolled *int64, instant *catalog.Instant, isHp bool) (int, bool) {
	if instant == nil {
		return 0, false
	}
	g := instant.Mp
	if isHp {
		g = instant.Hp
	}
	if g == nil {
		return 0, false
	}
	if rolled != nil {
		return int(*rolled), true
	}
	return rollAverage(g.Dice, g.Bonus), true
}

// wantsEffectRow decide se a dose deixa LINHA de efeito. A linha é duas coisas
// ao mesmo tempo: o que a ficha mostra em "Efeitos ativos" e o MARCADOR de que
// a porção do dia já foi consumida — é ela que o `OncePerDay` lá em cima
// procura, e é ela que o UNIQUE (characterId, catalogId, scope) protege.
//
// Exigir modificadores para criá-la mata a regra da porção diária inteira: os
// cinco pratos que o catálogo marca como `oncePerDay` só curam e nenhum tem
// modificador, então não haveria marcador para achar e a mesa comeria o mesmo
// prato a manhã inteira.
func wantsEffectRow(spec *catalog.Consumable) bool {
	return spec.Scope != "instant" && (spec.OncePerDay || hasModifiers(spec.Modifiers))
}

// effectModifiers normaliza o blob do catálogo. A coluna é NOT NULL e a ficha
// faz JSON.parse nela: o marcador sem modificador guarda "[]" e não "".
func effectModifiers(raw json.RawMessage) string {
	if !hasModifiers(raw) {
		return "[]"
	}
	return string(raw)
}

func hasModifiers(raw json.RawMessage) bool {
	s := strings.TrimSpace(string(raw))
	return s != "" && s != "null" && s != "[]"
}

var diceRe = regexp.MustCompile(`^(\d+)d(\d+)$`)

// rollAverage é a média de uma rolagem: NdF vira `floor(N*(F+1)/2)`, um
// inteiro pelado vale como bônus fixo, e "" ou "0" devolvem só o bônus.
func rollAverage(dice string, bonus int) int {
	t := strings.TrimSpace(dice)
	if t == "" || t == "0" {
		return bonus
	}
	if flat, err := strconv.Atoi(t); err == nil {
		return flat + bonus
	}
	m := diceRe.FindStringSubmatch(strings.ToLower(t))
	if m == nil {
		return bonus
	}
	n, _ := strconv.Atoi(m[1])
	f, _ := strconv.Atoi(m[2])
	return (n*(f+1))/2 + bonus
}
