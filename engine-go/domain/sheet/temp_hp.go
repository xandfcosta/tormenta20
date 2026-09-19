package sheet

import (
	"encoding/json"
	"sort"

	"t20engine/infra/db/sqlcgen"
)

// O DANO CONTRA PV TEMPORÁRIO: qual poça esvazia primeiro, e o que sobra da
// poça vazia.
//
// AS POÇAS SOMAM. *"Certos efeitos fornecem PV ou PM temporários. Eles são
// somados a seus pontos atuais, mesmo que ultrapassem o máximo"* (p106). Duas
// fontes convivem, e o que empilha os valores é o motor — os modificadores
// `tempHp` saem com `bonusType: untyped`, que é o tipo que acumula.
//
// O dano que chega drena a MAIOR poça antes de tocar o PV. **A ORDEM não é do
// livro** — ele só diz que os pontos temporários "são sempre os primeiros a
// serem gastos" (p106), e cala sobre qual poça primeiro. A maior é escolha
// nossa, e o que ela protege é a poça pequena de cena, que expira sozinha.
//
// As poças vivem como modificadores `tempHp` nas linhas de efeito ativo, e a
// distinção que importa é entre poça PURA e MISTA: a pura (só modificadores de
// tempHp) é APAGADA quando esvazia; a mista fica, com o tempHp zerado, porque
// os outros modificadores dela continuam valendo.
//
// Mora no `sheet` pela mesma razão do `equip.go`: é regra sobre a ficha, lida
// pela cena E pelo hospedeiro.

type TempHpPool struct {
	EffectID  int64
	CatalogID string
	Scope     string
	Amount    int
	Pure      bool
	Mods      []map[string]any // guardado CRU, para uma reescrita não perder campo
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
			continue // intocada — não entra no delta
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
