package engine

import (
	"encoding/json"
	"math"
	"sort"
)

// O motor de RESOLUÇÃO: recebe uma `[]ActiveItem` já coletada e a resolve em
// `ItemEffects` — não-empilhamento por `bonusType`, flags, condicionais.
//
// Ele não lê catálogo de propósito: ler o livro é trabalho da camada de COLETA
// (collect.go), e todo `ActiveItem` chega montado. É essa divisão que deixa a
// regra inteira de não-empilhamento ser testada com dado inline.

// ─── Tipos ────────────────────────────────────────────────────────────

// ModifierTarget é uma união achatada por `k`: só os campos que aquele `k` usa
// são preenchidos (`omitempty` mantém a forma do JSON 1:1).
type ModifierTarget struct {
	K         string `json:"k"`
	Name      string `json:"name,omitempty"`      // expertise, expertiseRemovePenalty, attribute, maneuver, flag
	Attribute string `json:"attribute,omitempty"` // expertiseByAttribute
	Scope     string `json:"scope,omitempty"`     // attack, damage ('this' | 'all')
	School    string `json:"school,omitempty"`    // catalyst
}

// ModifierCondition is a condição do modificador, achatada do mesmo jeito.
type ModifierCondition struct {
	C     string `json:"c"`
	Type  string `json:"type,omitempty"`  // terrain
	Trait string `json:"trait,omitempty"` // against
	Note  string `json:"note,omitempty"`  // context
	Flag  string `json:"flag,omitempty"`  // flagOn, flagOff
	Label string `json:"label,omitempty"` // flagOn, flagOff
}

// VitalScale é a escala de um vital. Só o coletor de vitais a lê; o motor de
// resolução a ignora. Ela viaja no `Modifier` para a camada de coleta devolver
// os mods de maxPv/maxPm byte a byte iguais ao oráculo.
type VitalScale struct {
	Per       string `json:"per"`
	Step      int    `json:"step,omitempty"`
	Round     string `json:"round,omitempty"`
	Attribute string `json:"attribute,omitempty"`
}

// amountInEngineUnits converte o valor do catálogo para a unidade do motor.
//
// # O DESLOCAMENTO vem em METROS e é guardado em QUADRADOS
//
// O livro mede deslocamento em metros e joga num grid de 1,5m, e TODO valor que
// ele imprime é múltiplo de 1,5 — medido no catálogo: {-3, 1,5, 6, 9, 12}.
// Guardar em metros obrigava a arredondar, e as botas reforçadas (+1,5m, p159)
// viravam +2m: meio metro de bônus inventado pela fronteira do JSON.
//
// Em quadrados a conta fecha exata, e é a MESMA decisão que o
// `board_movement.go` já tinha tomado para o mapa — "a conta é feita em
// QUADRADOS inteiros, nunca em metros… metro é coisa de tela" (p236). De
// quebra, a metade do Lento (p395: "arredonde para baixo para o primeiro
// incremento de 1,5m") vira divisão inteira, sem regra de arredondamento
// escrita à mão.
//
// Arredonda em vez de truncar porque um verbete NOVO pode trazer um valor que
// não seja múltiplo de 1,5 — e aí o vizinho certo é melhor que o de baixo. Se
// isso acontecer, o lugar de consertar é o verbete.
func amountInEngineUnits(target ModifierTarget, amount float64) int {
	if target.K == "displacement" {
		return int(math.Round(amount / SquareMetres))
	}
	return int(math.Round(amount))
}

// amountInBookUnits é a VOLTA, e ela não é enfeite: sem a simetria a conversão
// não é idempotente, e ler de volta o que o motor escreveu converte duas vezes
// — a armadura ia de −3m para −2 quadrados e de −2 para −1 na segunda leitura.
//
// Quem denunciou foi o `roundTrip` dos casos de paridade, que serializa a saída
// do motor e a relê. Ele não estava medindo unidade nenhuma: ele mede FORMA, e
// tropeçou na assimetria de graça.
//
// O efeito colateral é o que se queria: o FIO fala a unidade do LIVRO. O
// oráculo mostra −3 e +1,5 como a página imprime, e quem revisa o diff contra o
// livro compara os mesmos números.
func amountInBookUnits(target ModifierTarget, amount int) float64 {
	if target.K == "displacement" {
		return float64(amount) * SquareMetres
	}
	return float64(amount)
}

// MarshalJSON emite o modificador na unidade do LIVRO — ver `amountInBookUnits`.
func (m Modifier) MarshalJSON() ([]byte, error) {
	type wire struct {
		Target    ModifierTarget     `json:"target"`
		Amount    float64            `json:"amount"`
		BonusType string             `json:"bonusType"`
		Condition *ModifierCondition `json:"condition,omitempty"`
		Note      string             `json:"note,omitempty"`
		Scale     *VitalScale        `json:"scale,omitempty"`
		Factor    *Ratio             `json:"factor,omitempty"`
	}
	return json.Marshal(wire{
		Target:    m.Target,
		Amount:    amountInBookUnits(m.Target, m.Amount),
		BonusType: m.BonusType,
		Condition: m.Condition,
		Note:      m.Note,
		Scale:     m.Scale,
		Factor:    m.Factor,
	})
}

// Modifier é um modificador de item. O `scale` (maxPv/maxPm) é ignorado pelo
// motor de resolução e preservado para o despejo de paridade da coleta.
type Modifier struct {
	Target    ModifierTarget     `json:"target"`
	Amount    int                `json:"amount"`
	BonusType string             `json:"bonusType"`
	Condition *ModifierCondition `json:"condition,omitempty"`
	Note      string             `json:"note,omitempty"`
	Scale     *VitalScale        `json:"scale,omitempty"`
	// Factor MULTIPLICA o total do alvo, depois da soma — ver `factor.go`. Um
	// modificador com fator ignora o `Amount`: ele não é parcela da pilha.
	Factor *Ratio `json:"factor,omitempty"`
}

// UnmarshalJSON leva o `amount` do catálogo para a unidade do MOTOR.
//
// O motor é modelado em INTEIROS (ver types.go), e quem traz o valor é o
// catálogo, que escreve na unidade do LIVRO. Para quase todo alvo as duas
// coincidem e o valor passa intocado. O DESLOCAMENTO é a exceção, e ela tem
// razão própria — ver `amountInEngineUnits`.
func (m *Modifier) UnmarshalJSON(b []byte) error {
	var shadow struct {
		Target    ModifierTarget     `json:"target"`
		Amount    float64            `json:"amount"`
		BonusType string             `json:"bonusType"`
		Condition *ModifierCondition `json:"condition"`
		Note      string             `json:"note"`
		Scale     *VitalScale        `json:"scale"`
		Factor    *Ratio             `json:"factor"`
	}
	if err := json.Unmarshal(b, &shadow); err != nil {
		return err
	}
	*m = Modifier{
		Target:    shadow.Target,
		Amount:    amountInEngineUnits(shadow.Target, shadow.Amount),
		BonusType: shadow.BonusType,
		Condition: shadow.Condition,
		Note:      shadow.Note,
		Scale:     shadow.Scale,
		Factor:    shadow.Factor,
	}
	return nil
}

// ActiveItem é um item com os modificadores dele. `Equipped` é ponteiro para o
// estado NULO (item presente e não equipado) se distinguir de "vestido".
type ActiveItem struct {
	// SourceID é o id ESTÁVEL do verbete que concede — `armadura-completa`,
	// `anao-devagar-e-sempre`. Ele entrou na ALE-386 porque a coleta só guardava
	// o `Source`, que é o TEXTO DE TELA: uma frase montada com prefixo e
	// concatenação, que muda quando alguém corrige um acento. Uma regra da mesa
	// que precise dizer "cancele o que vem daquele item" não tem como
	// endereçá-lo por uma frase.
	//
	// Vazio é legítimo: as condições não vêm de verbete nenhum.
	SourceID  string     `json:"sourceId,omitempty"`
	Source    string     `json:"source"`
	Equipped  *string    `json:"equipped"`
	Modifiers []Modifier `json:"modifiers"`
}

type Contribution struct {
	// SourceID atravessa a dobra junto com o `Source` — ver `ActiveItem`.
	SourceID  string `json:"sourceId,omitempty"`
	Source    string `json:"source"`
	BonusType string `json:"bonusType"`
	Amount    int    `json:"amount"`
	Note      string `json:"note,omitempty"`
}

type AggregatedStat struct {
	Total         int            `json:"total"`
	Contributions []Contribution `json:"contributions"`
}

type ConditionalEffect struct {
	// Term é o ENDEREÇO deste condicional — o mesmo `engine.TermID` que o
	// silêncio da mesa usa. É por ele que o opt-in do jogador casa de volta com o
	// modificador que o gerou.
	//
	// Ele nasce AQUI e não é recalculado depois, porque a `Condition` e a `Scale`
	// do modificador se PERDEM nesta conversão: a partir de um
	// `ConditionalEffect` pronto já não há como endereçá-lo.
	Term string `json:"term"`
	// SourceID atravessa junto — ver `ActiveItem`.
	SourceID  string         `json:"sourceId,omitempty"`
	Source    string         `json:"source"`
	BonusType string         `json:"bonusType"`
	Amount    int            `json:"amount"`
	Note      string         `json:"note"`
	Target    ModifierTarget `json:"target"`
	Flag      string         `json:"flag,omitempty"`
}

// ItemEffects são os efeitos já resolvidos. `Flags` é um Set (mapa); o
// `MarshalJSON` o emite como array ORDENADO, para a paridade de JSON com o
// oráculo não depender de ordem.
type ItemEffects struct {
	ByTarget map[string]AggregatedStat
	Flags    map[string]bool
	// Factors é o fator JÁ RESOLVIDO por alvo — o mais severo vence, e eles não
	// compõem (ver `factor.go`). Quem o APLICA é a decomposição, porque ele age
	// sobre o total COM a base, e a base não passa por aqui.
	Factors     map[string]Ratio
	Conditional []ConditionalEffect
}

// ItemEffectsWire é a forma que o ItemEffects assume NO FIO: as flags viram
// array ordenado em vez do Set interno, e os nils viram vazios.
//
// Tem nome próprio (em vez de struct anônima) porque é o contrato que o gerador
// de tipos TS emite — refletir a struct em memória produziria
// `Flags: Record<string, boolean>`, que é mentira.
type ItemEffectsWire struct {
	ByTarget    map[string]AggregatedStat `json:"byTarget"`
	Flags       []string                  `json:"flags"`
	Conditional []ConditionalEffect       `json:"conditional"`
}

func (e ItemEffects) MarshalJSON() ([]byte, error) {
	byTarget := e.ByTarget
	if byTarget == nil {
		byTarget = map[string]AggregatedStat{}
	}
	conditional := e.Conditional
	if conditional == nil {
		conditional = []ConditionalEffect{}
	}
	return json.Marshal(ItemEffectsWire{byTarget, e.FlagList(), conditional})
}

// FlagList devolve os nomes das flags ativas ordenados — a forma estável para o
// JSON e para comparar por valor.
func (e ItemEffects) FlagList() []string {
	out := make([]string, 0, len(e.Flags))
	for f := range e.Flags {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

// ─── targetKey (stable identity per target shape) ─────────────────────

func targetKey(t ModifierTarget) string {
	switch t.K {
	case "expertise":
		return "expertise:" + t.Name
	case "expertiseAll":
		return "expertiseAll"
	case "expertiseRemovePenalty":
		return "expertiseRemovePenalty:" + t.Name
	case "expertiseByAttribute":
		return "expertiseByAttribute:" + t.Attribute
	case "attribute":
		return "attribute:" + t.Name
	case "defense":
		// Escopo separa a Defesa DIRECIONAL do Caído da geral: chaves distintas
		// não competem entre si, que é exatamente o que a p394 pede ao dizer que
		// a dele é "cumulativa com outras condições".
		if t.Scope != "" && t.Scope != "all" {
			return "defense:" + t.Scope
		}
		return "defense"
	case "defenseDexCap":
		return "defenseDexCap"
	case "resistance":
		return "resistance"
	case "fearResistance":
		return "fearResistance"
	case "attack":
		return "attack:" + t.Scope
	case "damage":
		return "damage:" + t.Scope
	case "critRange":
		return "critRange"
	case "critMult":
		return "critMult"
	case "pmLimit":
		return "pmLimit"
	case "pmCost":
		return "pmCost"
	case "damageReduction":
		return "damageReduction"
	case "catalyst":
		return "catalyst:" + t.School
	case "spellDC":
		return "spellDC"
	case "inventorySlots":
		return "inventorySlots"
	case "displacement":
		// Escopo separa a redução por ARMADURA do resto, e a razão é a p20: o
		// anão diz, com todas as letras, que seu deslocamento "não é reduzido
		// por uso de armadura ou excesso de carga". Sem chave própria não há
		// como isentar uma fonte sem isentar as outras — as botas reforçadas
		// (+1,5m) e uma magia de lentidão cairiam junto.
		if t.Scope != "" {
			return "displacement:" + t.Scope
		}
		return "displacement"
	case "flySpeed":
		return "flySpeed"
	case "armorPenalty":
		return "armorPenalty"
	case "armorPenaltyExpertises":
		return "armorPenaltyExpertises"
	case "tempHp":
		return "tempHp"
	case "tempMp":
		return "tempMp"
	case "maxPv":
		return "maxPv"
	case "maxPm":
		return "maxPm"
	case "maneuver":
		return "maneuver:" + t.Name
	case "flag":
		return "flag:" + t.Name
	}
	return ""
}

// ─── condition helpers ────────────────────────────────────────────────

func isUnconditional(m Modifier) bool {
	if m.Condition == nil {
		return true
	}
	switch m.Condition.C {
	case "always", "wielded", "vested":
		return true
	case "terrain", "against", "context", "flagOn":
		return false
	case "flagOff":
		// Avaliada automaticamente contra as flags já coletadas na passada
		// principal — nunca é uma condicional que a pessoa liga.
		return true
	}
	return true
}

func isWielded(equipped *string) bool {
	return equipped != nil && (*equipped == "wielded" || *equipped == "wielded2")
}

func conditionMet(m Modifier, equipped *string) bool {
	if m.Condition == nil || m.Condition.C == "always" {
		return true
	}
	switch m.Condition.C {
	case "wielded":
		return isWielded(equipped)
	case "vested":
		return equipped != nil && *equipped == "vested"
	case "flagOff":
		// O `flagOff` já passou pelo portão de flags no laço principal.
		return true
	}
	return false
}

func describeCondition(m Modifier) string {
	if m.Condition == nil {
		return ""
	}
	switch m.Condition.C {
	case "terrain":
		return "terreno: " + m.Condition.Type
	case "against":
		return "contra: " + m.Condition.Trait
	case "context":
		return m.Condition.Note
	case "flagOn", "flagOff":
		return m.Condition.Label
	}
	return ""
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// ─── non-stacking resolution ──────────────────────────────────────────

// AS DUAS FRASES DO EMPILHAMENTO (T20 p107), num dono só.
//
// Elas existem como função porque DOIS percursos as usam: o `resolveStack`, que
// dobra uma lista, e o sistema `resolveStacking` da resolução em ECS, que
// percorre entidades. Os percursos são diferentes de propósito; a REGRA não
// pode ser, ou ela diverge na primeira vez que alguém mexer num dos dois.

// accumulates diz se o tipo de bônus SOMA em vez de disputar. Só o `untyped`.
func accumulates(bonusType string) bool { return bonusType == "untyped" }

// strongerThan é o critério da disputa: vence o de maior valor ABSOLUTO — uma
// penalidade de −3 é mais forte que uma de −1.
func strongerThan(amount, than int) bool { return absInt(amount) > absInt(than) }

// resolveStack aplica a regra de NÃO-EMPILHAMENTO do T20: dentro de um alvo,
// entradas do mesmo `bonusType` guardam só a de maior valor ABSOLUTO; `untyped`
// empilha à vontade. A ordem das contribuições segue a de PRIMEIRA APARIÇÃO de
// cada `bonusType`, que é o que faz a paridade de JSON valer.
func resolveStack(contribs []Contribution) AggregatedStat {
	order := []string{}
	byType := map[string][]Contribution{}
	for _, c := range contribs {
		if _, ok := byType[c.BonusType]; !ok {
			order = append(order, c.BonusType)
		}
		byType[c.BonusType] = append(byType[c.BonusType], c)
	}

	kept := []Contribution{}
	for _, bt := range order {
		list := byType[bt]
		if accumulates(bt) {
			kept = append(kept, list...)
			continue
		}
		best := list[0]
		for _, e := range list {
			if strongerThan(e.Amount, best.Amount) {
				best = e
			}
		}
		kept = append(kept, best)
	}

	total := 0
	for _, c := range kept {
		total += c.Amount
	}
	return AggregatedStat{Total: total, Contributions: kept}
}

// ConditionalDisplayInput é uma linha de efeito condicional entregue ao
// `ResolveConditionalDisplay` (as linhas de uma postura ativa).
type ConditionalDisplayInput struct {
	Target    ModifierTarget `json:"target"`
	BonusType string         `json:"bonusType"`
	Amount    int            `json:"amount"`
}

type ConditionalDisplayRow struct {
	Target ModifierTarget `json:"target"`
	Amount int            `json:"amount"`
}

// ResolveConditionalDisplay resolve as linhas de uma postura ativa para exibir:
// agrupa por identidade de alvo, roda a mesma resolução por `bonusType`, e
// devolve só as linhas {alvo, valor} que sobreviveram.
func ResolveConditionalDisplay(effects []ConditionalDisplayInput) []ConditionalDisplayRow {
	order := []string{}
	targets := map[string]ModifierTarget{}
	byKey := map[string][]Contribution{}
	for _, e := range effects {
		key := targetKey(e.Target)
		if _, ok := byKey[key]; !ok {
			order = append(order, key)
			targets[key] = e.Target
		}
		byKey[key] = append(byKey[key], Contribution{BonusType: e.BonusType, Amount: e.Amount})
	}
	kept := []ConditionalDisplayRow{}
	for _, key := range order {
		for _, c := range resolveStack(byKey[key]).Contributions {
			kept = append(kept, ConditionalDisplayRow{Target: targets[key], Amount: c.Amount})
		}
	}
	return kept
}

// ─── computeItemEffects ───────────────────────────────────────────────

// ComputeItemEffects dobra um conjunto de `ActiveItem` em `ItemEffects`
// resolvidos.
//
// Desde a ALE-378 (fatia 3) ela é uma linha: a resolução mora em SISTEMAS, com
// o TERMO como entidade — ver `resolve_ecs.go`. O nome fica porque ele está em
// dez chamadores e descreve o que a coisa faz; o que mudou é quem faz.
//
// @example engine.ComputeItemEffects(world.ActiveItemsFor(ch))
func ComputeItemEffects(items []ActiveItem) ItemEffects {
	return resolveInWorld(items)
}

// firstNonEmpty devolve a primeira não vazia.
func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// StatFor procura o agregado de um alvo, devolvendo zerado quando não há.
func StatFor(effects ItemEffects, target ModifierTarget) AggregatedStat {
	if stat, ok := effects.ByTarget[targetKey(target)]; ok {
		return stat
	}
	return AggregatedStat{Total: 0, Contributions: []Contribution{}}
}

// FlagGroupID é o endereço de um GRUPO de condicionais que dividem uma flag.
//
// Ele existe porque a tela oferece UM interruptor para o grupo — "um item
// caseiro com três modificadores é uma coisa só na mesa" —, e um interruptor
// que grave o endereço de UM dos membros liga um terço da regra. O grupo tem de
// ter endereço PRÓPRIO, e não emprestado do primeiro membro.
//
// O prefixo é o que impede as duas famílias de chave de colidirem na mesma
// coluna: endereço de termo sempre tem `::`, e endereço de grupo nunca tem.
//
// @example engine.FlagGroupID("furia") // "flag:furia"
func FlagGroupID(flag string) string { return "flag:" + flag }

// ApplyActiveConditionals dobra de volta em `byTarget` os efeitos condicionais
// cujos ids estão em `activeIds`, refazendo a resolução de não-empilhamento por
// alvo. Condicionais de FLAG são ignoradas (ainda não há tela para ligá-las).
func ApplyActiveConditionals(effects ItemEffects, activeIds map[string]bool) ItemEffects {
	if len(activeIds) == 0 {
		return effects
	}
	buckets := map[string][]Contribution{}
	for key, agg := range effects.ByTarget {
		buckets[key] = append([]Contribution{}, agg.Contributions...)
	}
	// Ao contrário do `ComputeItemEffects` acima, esta não precisa ordenar chaves:
	// ela emite um mapa, e a ordem das contribuições dentro de cada chave já é
	// estável porque cada balde é uma fatia acrescida na ordem da fonte.
	remaining := []ConditionalEffect{}
	for _, c := range effects.Conditional {
		// LIGADO PELO TERMO ou pelo GRUPO da flag. O grupo é o que faz o
		// interruptor único da tela ligar a regra INTEIRA: só pelo termo, um
		// condicional de três modificadores entraria com um.
		if !activeIds[c.Term] && !(c.Flag != "" && activeIds[FlagGroupID(c.Flag)]) {
			remaining = append(remaining, c)
			continue
		}
		if c.Target.K == "flag" {
			continue
		}
		key := targetKey(c.Target)
		fold := Contribution{SourceID: c.SourceID, Source: c.Source + " (cond.)", BonusType: c.BonusType, Amount: c.Amount}
		if c.Note != "" {
			fold.Note = c.Note
		}
		buckets[key] = append(buckets[key], fold)
	}
	byTarget := map[string]AggregatedStat{}
	for key := range buckets {
		byTarget[key] = resolveStack(buckets[key])
	}
	// Os FATORES atravessam intactos: nenhum condicional traz fator hoje — os
	// dois que existem vêm da tabela de condições, que os declara sem condição —
	// e perdê-los aqui faria a ficha com opt-in ligado andar mais que a sem.
	return ItemEffects{
		ByTarget: byTarget, Flags: effects.Flags, Factors: effects.Factors, Conditional: remaining,
	}
}
