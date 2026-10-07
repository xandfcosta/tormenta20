package sheet

import (
	"encoding/json"
	"sort"

	"t20engine/infra/db/sqlcgen"
)

// AS POÇAS TEMPORÁRIAS, de PV e de PM: qual esvazia primeiro, e o que sobra da
// poça vazia.
//
// UMA REGRA SÓ PARA OS DOIS VITAIS, e é o livro que os junta na mesma frase:
// *"Certos efeitos fornecem PV ou PM temporários. Eles são somados a seus
// pontos atuais, mesmo que ultrapassem o máximo. Pontos temporários são sempre
// os primeiros a serem gastos. Caso não seja especificado o contrário, pontos
// temporários desaparecem no fim do dia"* (p106). O que muda entre o PV e o PM
// é o ALVO do modificador e o que acontece com o que SOBRA do gasto — e é só
// isso que os dois invólucros no fim do arquivo carregam.
//
// AS POÇAS SOMAM: duas fontes convivem, e o que empilha os valores é o motor —
// os modificadores saem com `bonusType: untyped`, que é o tipo que acumula.
//
// O gasto drena a MAIOR poça primeiro. **A ORDEM não é do livro** — ele só diz
// que os temporários vêm antes, e cala sobre qual poça antes. A maior é
// escolha nossa, e o que ela protege é a poça pequena de cena, que expira
// sozinha.
//
// As poças vivem como modificadores nas linhas de efeito ativo, e a distinção
// que importa é entre poça PURA e MISTA: a pura (só modificadores daquele
// alvo) é APAGADA quando esvazia; a mista fica, zerada, porque os outros
// modificadores dela continuam valendo.
//
// Mora no `sheet` pela mesma razão do `equip.go`: é regra sobre a ficha, lida
// pela cena E pelo hospedeiro.

// Os dois alvos de modificador que fazem poça. Constantes e não literais
// espalhados: o casamento é por TEXTO, e um `"tempMP"` digitado assim não
// estoura — ele só não acha poça nenhuma.
const (
	TempHpTarget = "tempHp"
	TempMpTarget = "tempMp"
)

// TempPool é uma poça de um dos dois vitais.
type TempPool struct {
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

// ParseTempPools lê as poças de UM alvo nas linhas de efeito ativo.
//
// Os modificadores viajam num JSON, e ela guarda o mapa CRU de cada um: uma
// reescrita depois precisa devolver todos os campos, inclusive os que este
// código não conhece.
func ParseTempPools(target string, rows []sqlcgen.ListActiveEffectsByCharacterRow) []TempPool {
	pools := []TempPool{}
	for _, row := range rows {
		amount, pure, mods, ok := readTempPool(target, row.Modifiers)
		if !ok {
			continue
		}
		pools = append(pools, TempPool{
			EffectID: row.ID, CatalogID: row.Catalogid, Scope: row.Scope,
			Amount: amount, Pure: pure, Mods: mods,
		})
	}
	return pools
}

// readTempPool lê UMA linha de efeito: quanto daquele vital ela carrega, se
// ela é PURA, e os modificadores crus.
//
// `ok` falso quer dizer "esta linha não é uma poça": JSON ilegível, nenhum
// modificador de tempHp, ou um valor que não é positivo. Os três são o mesmo
// caso para quem chama — não há poça —, e distingui-los daria três ramos para
// uma decisão só.
//
// O PRIMEIRO modificador do alvo é o que vale. Uma linha com dois seria o
// catálogo dizendo duas coisas sobre a mesma poça, e somá-los inventaria um
// número que nenhuma fonte prometeu.
func readTempPool(target, modifiers string) (int, bool, []map[string]any, bool) {
	var mods []map[string]any
	if json.Unmarshal([]byte(modifiers), &mods) != nil {
		return 0, false, nil, false
	}
	amount, found, pure := 0, false, true
	for _, m := range mods {
		if IsTempModifier(target, m) {
			if !found {
				amount, found = ToInt(m["amount"]), true
			}
			continue
		}
		pure = false
	}
	if !found || amount <= 0 {
		return 0, false, nil, false
	}
	return amount, pure, mods, true
}

// TempTotal é quanto de um vital temporário um personagem tem, somando as
// linhas de efeito dele.
//
// SOMAR é a regra do livro: *"Certos efeitos fornecem PV ou PM temporários.
// Eles são somados a seus pontos atuais, mesmo que ultrapassem o máximo"*
// (**p106**). Duas fontes convivem — foi a ALE-347 que tirou daqui um
// "vale-o-maior" que o livro não tem.
//
// Ela recebe os BLOBS e não as linhas porque os dois chamadores têm formas
// diferentes na mão: a ficha lê o agregado já montado (`[]EffectDTO`) e a Mesa
// lê uma consulta de várias fichas de uma vez. O que os dois têm em comum é o
// texto do modificador.
func TempTotal(target string, modifiers []string) int {
	total := 0
	for _, blob := range modifiers {
		if amount, _, _, ok := readTempPool(target, blob); ok {
			total += amount
		}
	}
	return total
}

// DrainPlan é o que um gasto faz com as POÇAS, e nada sobre o vital.
//
// Ele para antes do vital de propósito: o que sobra do gasto significa coisas
// diferentes no PV e no PM — lá ele desce abaixo de zero até o limiar da morte,
// aqui ele é recusado antes de chegar. Um plano só que decidisse os dois teria
// de conhecer as duas regras.
type DrainPlan struct {
	Drained   []DamageDrain
	Updates   []EffectModifierWrite
	DeleteIDs []int64
	// Remaining é o que ficou nas poças depois do gasto.
	Remaining int
	// Left é o que o gasto NÃO conseguiu tirar das poças, e vai para o vital.
	Left int
}

// PlanDrain tira `amount` das poças daquele alvo, a maior primeiro.
func PlanDrain(target string, pools []TempPool, amount int) DrainPlan {
	plan := DrainPlan{Drained: []DamageDrain{}, Left: amount}
	sort.SliceStable(pools, func(i, j int) bool { return pools[i].Amount > pools[j].Amount })
	for _, pool := range pools {
		drained := min(plan.Left, pool.Amount)
		plan.Left -= drained
		newAmount := pool.Amount - drained
		plan.Remaining += newAmount
		if newAmount == pool.Amount {
			continue // intocada — não entra no delta
		}
		removed := newAmount == 0 && pool.Pure
		plan.Drained = append(plan.Drained, DamageDrain{EffectID: pool.EffectID, NewAmount: newAmount, Removed: removed})
		if removed {
			plan.DeleteIDs = append(plan.DeleteIDs, pool.EffectID)
			continue
		}
		plan.Updates = append(plan.Updates,
			EffectModifierWrite{pool.EffectID, withTempAmount(target, pool.Mods, newAmount)})
	}
	return plan
}

// ── os dois invólucros: o que muda é o que o SOBRANTE faz ───────────────────

// ParseTempHpPools lê as poças de PV das linhas de efeito ativo.
func ParseTempHpPools(rows []sqlcgen.ListActiveEffectsByCharacterRow) []TempPool {
	return ParseTempPools(TempHpTarget, rows)
}

// ParseTempMpPools lê as poças de PM das linhas de efeito ativo.
func ParseTempMpPools(rows []sqlcgen.ListActiveEffectsByCharacterRow) []TempPool {
	return ParseTempPools(TempMpTarget, rows)
}

// TempHpTotal é quanto de PV temporário um personagem tem.
func TempHpTotal(modifiers []string) int { return TempTotal(TempHpTarget, modifiers) }

// TempMpTotal é quanto de PM temporário um personagem tem.
func TempMpTotal(modifiers []string) int { return TempTotal(TempMpTarget, modifiers) }

// PlanDamage drena as poças de PV e derrama o resto no PV.
func PlanDamage(pools []TempPool, hpCurrent, amount int) DamagePlan {
	drain := PlanDrain(TempHpTarget, pools, amount)
	return DamagePlan{
		Drained: drain.Drained, Updates: drain.Updates, DeleteIDs: drain.DeleteIDs,
		// SEM PISO aqui: o PV desce abaixo de zero até o limiar da morte (p236),
		// e quem prende é o funil (`WithinHitPoints`). Um piso a mais neste passo
		// foi a terceira grafia do "piso zero", e o guarda de grafia única não a
		// via porque ela não tinha a forma `min(max(…))` (ALE-366).
		HpCurrent: hpCurrent - drain.Left, TempHpRemaining: drain.Remaining,
	}
}

// SpendPlan é o gasto de PM depois de a poça temporária pagar a parte dela.
type SpendPlan struct {
	Drained   []DamageDrain
	Updates   []EffectModifierWrite
	DeleteIDs []int64
	// MpCurrent é o poço de verdade depois do gasto, e ele NÃO desce abaixo de
	// zero: PM que falta é gesto recusado, e a recusa é de quem chama. Um poço
	// negativo gravado desenharia uma barra para trás.
	MpCurrent       int
	TempMpRemaining int
}

// PlanSpend gasta a poça temporária ANTES do poço — *"pontos temporários são
// sempre os primeiros a serem gastos"* (p106).
func PlanSpend(pools []TempPool, mpCurrent, amount int) SpendPlan {
	drain := PlanDrain(TempMpTarget, pools, amount)
	return SpendPlan{
		Drained: drain.Drained, Updates: drain.Updates, DeleteIDs: drain.DeleteIDs,
		MpCurrent: max(0, mpCurrent-drain.Left), TempMpRemaining: drain.Remaining,
	}
}

// withTempAmount reescreve o valor do modificador daquele alvo e recodifica,
// PRESERVANDO todos os outros campos.
//
// A ida e volta é por MAPA e não por struct tipada, e isso é deliberado: uma
// struct só carrega os campos que ela declara, então um campo que este código
// não conhece sumiria na regravação — em silêncio.
func withTempAmount(target string, Mods []map[string]any, Amount int) string {
	for _, m := range Mods {
		if IsTempModifier(target, m) {
			m["amount"] = Amount
		}
	}
	b, _ := json.Marshal(Mods)
	return string(b)
}

// IsTempModifier diz se um modificador cru é da poça daquele alvo.
func IsTempModifier(target string, m map[string]any) bool {
	t, ok := m["target"].(map[string]any)
	return ok && t["k"] == target
}

// IsTempHpModifier é o invólucro de PV, para quem só pergunta por ele.
func IsTempHpModifier(m map[string]any) bool { return IsTempModifier(TempHpTarget, m) }

// ToInt lê um número de um modificador cru, que vem do JSON como `any`.
func ToInt(v any) int {
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return 0
}

// ── o MANA DISPONÍVEL, que é a pergunta de quem vai pagar ───────────────────

// ManaOf é quanto de PM um personagem pode gastar AGORA: o poço mais a poça.
//
// Ela existe porque *"pontos temporários são sempre os primeiros a serem
// gastos"* (p106) tem duas metades, e a segunda é fácil de esquecer: se o
// gasto drena a poça primeiro, então a poça também CONTA para decidir se o
// gasto cabe. Sem isso, o bardo com 0 PM no poço e 4 na poça seria recusado ao
// conjurar — a poça existiria só para ser ignorada.
//
// Ela recebe os BLOBS pelo mesmo motivo do `TempTotal`: os chamadores têm
// formas diferentes na mão, e o que todos têm é o texto do modificador.
func ManaOf(mpCurrent int64, modifiers []string) int64 {
	return mpCurrent + int64(TempMpTotal(modifiers))
}

// ModifierBlobsOf é o texto dos modificadores de cada efeito em curso.
func ModifierBlobsOf(effects []EffectDTO) []string {
	blobs := make([]string, 0, len(effects))
	for _, e := range effects {
		blobs = append(blobs, e.Modifiers)
	}
	return blobs
}

// ManaAvailable é o mana que ESTA ficha pode gastar agora.
func (c CharacterDTO) ManaAvailable() int64 {
	return ManaOf(c.MpCurrent, ModifierBlobsOf(c.ActiveEffects))
}

// TempMp é a poça de PM desta ficha, para quem precisa dela separada do poço —
// a tela, que mostra as duas parcelas.
func (c CharacterDTO) TempMp() int {
	return TempMpTotal(ModifierBlobsOf(c.ActiveEffects))
}
