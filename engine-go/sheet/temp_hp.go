package sheet

import (
	"encoding/json"
	"sort"

	"t20engine/db/sqlcgen"
)

// O DANO CONTRA PV TEMPORÁRIO: qual poça esvazia primeiro, e o que sobra da
// poça vazia.
//
// O dano que chega drena a MAIOR poça de PV temporário antes de tocar o PV.
// As poças vivem como modificadores `tempHp` nas linhas de efeito ativo, e a
// distinção que importa é entre poça PURA e MISTA: a pura (só modificadores de
// tempHp) é APAGADA quando esvazia; a mista fica, com o tempHp zerado, porque
// os outros modificadores dela continuam valendo.
//
// Portada do `temp-hp.helpers` do backend Nest. A prosa estava em INGLÊS e foi
// traduzida ao mudar de pacote (ALE-278) — comentário é o que uma pessoa lê.
//
// Ela mora no `sheet` pela mesma razão do `equip.go`: é regra sobre a ficha,
// lida pela cena E pelo hospedeiro, e os imports que ela precisa já eram os
// permitidos deste pacote.

type TempHpPool struct {
	EffectID  int64
	CatalogID string
	Scope     string
	Amount    int
	Pure      bool
	Mods      []map[string]any // preserved verbatim so a rewrite drops no fields
}

type DisplacedPool struct {
	EffectID int64 `json:"effectId"`
	Removed  bool  `json:"removed"`
}

// PoolPlan é o VALE-O-MAIOR do livro (p256) virado plano.
//
// Os campos são exportados porque quem APLICA é o hospedeiro: a decisão de qual
// poça vence é regra e mora aqui; apagar linha e reescrever modificador é
// escrita, e é dele.
type PoolPlan struct {
	Superseded   bool
	KeptEffectID int64
	KeptAmount   int
	Displaced    []DisplacedPool
	ZeroWrites   []EffectModifierWrite
	DeleteIDs    []int64
}

// PlanPoolSupremacy decide se uma poça nova pode existir ao lado das outras.
//
// Uma poça existente MAIOR OU IGUAL vence, e a nova nem chega a existir; senão a
// nova vence e toda poça menor é deslocada. É o "vale o maior" do livro (p256):
// PV temporário não soma, ele substitui.
func PlanPoolSupremacy(pools []TempHpPool, ownCatalogID, ownScope string, newAmount int) PoolPlan {
	others := []TempHpPool{}
	for _, p := range pools {
		if !(p.CatalogID == ownCatalogID && p.Scope == ownScope) {
			others = append(others, p)
		}
	}
	var top *TempHpPool
	for i := range others {
		if top == nil || others[i].Amount > top.Amount {
			top = &others[i]
		}
	}
	if top != nil && top.Amount >= newAmount {
		return PoolPlan{Superseded: true, KeptEffectID: top.EffectID, KeptAmount: top.Amount}
	}
	plan := PoolPlan{Displaced: []DisplacedPool{}}
	for _, p := range others {
		plan.Displaced = append(plan.Displaced, DisplacedPool{EffectID: p.EffectID, Removed: p.Pure})
		if p.Pure {
			plan.DeleteIDs = append(plan.DeleteIDs, p.EffectID)
		} else {
			plan.ZeroWrites = append(plan.ZeroWrites, EffectModifierWrite{p.EffectID, withTempHpAmount(p.Mods, 0)})
		}
	}
	return plan
}

// DamageDrain é uma poça drenada pelo dano, e quanto saiu dela.
type DamageDrain struct {
	EffectID  int64 `json:"effectId"`
	NewAmount int   `json:"newAmount"`
	Removed   bool  `json:"removed"`
}

// DamagePlan é o que o dano FAZ: quanto cada poça perde e quanto sobra no PV.
//
// Ele é exportado porque quem APLICA é o hospedeiro — a conta é regra e mora
// aqui, a escrita no banco é dele.
type DamagePlan struct {
	Drained         []DamageDrain
	Updates         []EffectModifierWrite
	DeleteIDs       []int64
	HpCurrent       int
	TempHpRemaining int
}

// EffectModifierWrite é uma reescrita de modificador que o hospedeiro aplica.
type EffectModifierWrite struct {
	EffectID  int64
	Modifiers string
}

// ParseTempHpPools lê as poças das linhas de efeito ativo.
//
// Os modificadores viajam num JSON, e ela guarda o mapa CRU de cada um: uma
// reescrita depois precisa devolver todos os campos, inclusive os que este
// código não conhece.
func ParseTempHpPools(rows []sqlcgen.ListActiveEffectsByCharacterRow) []TempHpPool {
	pools := []TempHpPool{}
	for _, row := range rows {
		var Mods []map[string]any
		if json.Unmarshal([]byte(row.Modifiers), &Mods) != nil {
			continue
		}
		Amount, found, Pure := 0, false, true
		for _, m := range Mods {
			if IsTempHpModifier(m) {
				if !found {
					Amount = ToInt(m["amount"])
					found = true
				}
			} else {
				Pure = false
			}
		}
		if !found || Amount <= 0 {
			continue
		}
		pools = append(pools, TempHpPool{
			EffectID: row.ID, CatalogID: row.Catalogid, Scope: row.Scope, Amount: Amount, Pure: Pure, Mods: Mods,
		})
	}
	return pools
}

// PlanDamage drena as poças (a maior primeiro) e derrama o resto no PV.
func PlanDamage(pools []TempHpPool, HpCurrent, Amount int) DamagePlan {
	plan := DamagePlan{Drained: []DamageDrain{}, HpCurrent: HpCurrent}
	sort.SliceStable(pools, func(i, j int) bool { return pools[i].Amount > pools[j].Amount })
	left := Amount
	for _, pool := range pools {
		Drained := min(left, pool.Amount)
		left -= Drained
		newAmount := pool.Amount - Drained
		plan.TempHpRemaining += newAmount
		if newAmount == pool.Amount {
			continue // untouched — not part of the delta
		}
		removed := newAmount == 0 && pool.Pure
		plan.Drained = append(plan.Drained, DamageDrain{EffectID: pool.EffectID, NewAmount: newAmount, Removed: removed})
		if removed {
			plan.DeleteIDs = append(plan.DeleteIDs, pool.EffectID)
			continue
		}
		plan.Updates = append(plan.Updates, EffectModifierWrite{pool.EffectID, withTempHpAmount(pool.Mods, newAmount)})
	}
	plan.HpCurrent = max(0, HpCurrent-left)
	return plan
}

// withTempHpAmount reescreve o valor do modificador de tempHp e recodifica,
// PRESERVANDO todos os outros campos.
//
// A ida e volta é por MAPA e não por struct tipada, e isso é deliberado: uma
// struct só carrega os campos que ela declara, então um campo que este código
// não conhece sumiria na regravação — em silêncio.
func withTempHpAmount(Mods []map[string]any, Amount int) string {
	for _, m := range Mods {
		if IsTempHpModifier(m) {
			m["amount"] = Amount
		}
	}
	b, _ := json.Marshal(Mods)
	return string(b)
}

// IsTempHpModifier diz se um modificador cru é de PV temporário.
func IsTempHpModifier(m map[string]any) bool {
	t, ok := m["target"].(map[string]any)
	return ok && t["k"] == "tempHp"
}

// ToInt lê um número de um modificador cru, que vem do JSON como `any`.
func ToInt(v any) int {
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return 0
}
