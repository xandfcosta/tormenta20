package engine

// WeaponCard is one wielded-weapon formula card (combat-magic-stats.tsx
// WeaponFormulaCards): attack = the weapon's attack perícia (Luta melee/thrown,
// Pontaria ranged) + global attack mods; damage adds Força for melee/thrown (not
// ranged) + global damage mods. Go owns the NUMBERS; the structural row labels
// (½ nível, FOR, Treino) and the crit string are applied on the front. Per-weapon
// mods (desbalanceada, Certeira…) already ride the Luta/Pontaria mirror inside
// effects, so the expertise breakdown carries them — no scope:'this' added here.
// ExtraDamage é uma parcela de dano de outro tipo, somada por um encanto.
type ExtraDamage struct {
	Dice string `json:"dice"` // "1d6"
	Type string `json:"type"` // "fogo", "frio", "ácido"…
}

// IsMelee diz se esta arma bate CORPO A CORPO.
//
// Pela PERÍCIA e não por uma lista de nomes: quem decide é o catálogo, e o
// cartão já carrega a resposta dele. Ela existe porque `Skill != "Luta"` tinha
// três sítios — a manobra, o Golpe Mágico e a lista de armas da ficha —, e uma
// comparação de texto repetida é a que erra a grafia num deles.
func (w WeaponCard) IsMelee() bool { return w.Skill == "Luta" }

type WeaponCard struct {
	Name      string             `json:"name"`
	Skill     string             `json:"skill"`     // "Luta" | "Pontaria"
	Attribute string             `json:"attribute"` // the skill perícia's attribute
	Attack    int                `json:"attack"`
	Expertise ExpertiseBreakdown `json:"expertise"`
	AttackAll TotalContribs      `json:"attackAll"`
	Damage    string             `json:"damage"` // dice, e.g. "1d8"
	// DamageType é o tipo do dano da ARMA — "corte", "perfuracao", "impacto" —,
	// e ele vale para os dados dela e para os bônus numéricos que a acompanham
	// (a Força, o dano do encanto sem tipo próprio). As parcelas de OUTRO tipo
	// viajam no `ExtraDamage`, cada uma com o seu.
	DamageType string `json:"damageType,omitempty"`
	// ExtraDamage são as parcelas de OUTRO tipo de dano que os encantos somam —
	// o "+1d6 de fogo" de uma arma flamejante (p336). Elas viajam separadas do
	// `Damage` por duas razões, e as duas são regra:
	//
	//   - no crítico a arma multiplica os dados e a parcela NÃO: "bônus
	//     numéricos de dano, assim como dados extras, não são multiplicados"
	//     (p231). Juntá-las na mesma notação dobraria o fogo junto;
	//   - o TIPO tem de sobreviver até a mesa, porque é ele que dá a
	//     resistência a fogo onde agir.
	ExtraDamage []ExtraDamage `json:"extraDamage,omitempty"`
	// CriticalBonus é o dano que só existe no acerto crítico — os "+10 pontos
	// de dano" do encanto Dilacerante (p336). É bônus numérico, então não
	// multiplica, e fora do crítico ele não existe.
	CriticalBonus int           `json:"criticalBonus,omitempty"`
	StrDamage     int           `json:"strDamage"`   // Força folded into melee/thrown damage (0 ranged)
	DamageBonus   int           `json:"damageBonus"` // strDamage + damageAll.total
	DamageAll     TotalContribs `json:"damageAll"`
	// DamageOwn são as parcelas que valem só para ESTA arma: o `this` da
	// sobreposição (a melhoria Cruel) e o `weapon:<id>` da arma escolhida num
	// poder (a Especialização em Arma, p65). Elas entravam no `DamageBonus` sem
	// linha na decomposição, e a soma das linhas não fechava (ALE-491).
	DamageOwn TotalContribs `json:"damageOwn"`
	CritRange int           `json:"critRange"`
	CritMult  int           `json:"critMult"`
	// Announces são as condições que o livro prevê para quem for atingido por
	// esta arma (p335-336). Elas viajam da carta para o resultado do ataque, que
	// filtra pelo gatilho — o que é só de crítico não sai num acerto normal.
	Announces []AnnouncedCondition `json:"announces,omitempty"`
	// NonLethal: TODO o dano desta arma é não letal (p236, p336). É propriedade
	// da ARMA e não parcela — a Piedosa diz "todo o dano causado", o que inclui
	// a Força e as parcelas dos outros encantos.
	NonLethal bool `json:"nonLethal,omitempty"`
}

// WeaponDamageScope é o escopo de dano de UMA arma do catálogo — onde cai o
// +2 da Especialização em Arma escolhida (p65).
//
//	WeaponDamageScope("espada-longa") // "weapon:espada-longa"
func WeaponDamageScope(catalogID string) string { return "weapon:" + catalogID }

// Os escopos de dano por TIPO de arma, com os nomes que o livro dá a eles.
const (
	// MeleeLightOrAgileScope: "uma arma corpo a corpo leve ou ágil"
	// (Esgrimista, p48).
	MeleeLightOrAgileScope = "melee-light-or-agile"
	// RangedScope: "uma arma de ataque à distância" (Arqueiro, p50), que o
	// livro divide em arremesso e disparo (p142) — as duas entram.
	RangedScope = "ranged"
)

// weaponKindScopes são os escopos de dano por TIPO em que esta arma cai.
//
// O tipo é o PROPÓSITO que o catálogo dá à arma, um só: a machadinha é de
// arremesso, então o Esgrimista não soma nela nem quando ela é usada corpo a
// corpo. É a mesma leitura que decide se a Força entra no dano.
func weaponKindScopes(w *WeaponStats) []string {
	switch {
	case w.Purpose == "ranged" || w.Purpose == "thrown":
		return []string{RangedScope}
	case w.Purpose == "melee" && (w.Hand == "light" || hasTrait(w.Traits, "agil")):
		return []string{MeleeLightOrAgileScope}
	}
	return nil
}

// ownDamageOf junta as parcelas que valem só para esta arma, numa decomposição.
func ownDamageOf(parts ...TotalContribs) TotalContribs {
	out := TotalContribs{Contributions: []BreakdownContribution{}}
	for _, p := range parts {
		out.Total += p.Total
		out.Contributions = append(out.Contributions, p.Contributions...)
	}
	return out
}

// ComputeWeaponCards resolves the wielded-weapon cards for a raw Character under
// the given active conditionals (Fúria's global attack/damage mods land in
// attackAll/damageAll). Only catalog weapons in a hand slot, capped at two.
func (r *Ruleset) ComputeWeaponCards(ch Character, activeConditionals map[string]bool) []WeaponCard {
	effects := r.sheetEffects(ch, activeConditionals)
	shared := newCardBasis(ch, effects)
	// O `this` é o bônus que uma SOBREPOSIÇÃO dá — a melhoria Cruel, o material.
	// Ele não tinha leitor nenhum, e por isso a Cruel (+1 dano, 300 PO) e a Atroz
	// (+2, 3.000 PO) não faziam nada. O par com o ataque é o que escondia: o
	// `attack:this` da Certeira SOBE o ataque, porque a perícia da arma soma os
	// efeitos de ataque — e a mesma forma em dano caía no vazio.
	//
	// As duas chaves somam entre si de propósito: um +2 global e um +1 desta
	// arma são bônus de origens diferentes, e o não-empilhamento já agiu DENTRO
	// de cada uma.
	damageThis := totalContribsFor(effects, ModifierTarget{K: "damage", Scope: "this"})

	cards := []WeaponCard{}
	for _, it := range ch.Items {
		if it.Equipped == nil || (*it.Equipped != "wielded" && *it.Equipped != "wielded2") {
			continue
		}
		if it.CatalogID == nil {
			continue
		}
		catalog := r.getCatalogItem(*it.CatalogID)
		if catalog == nil || catalog.Weapon == nil {
			continue
		}
		w := catalog.Weapon
		weaponScope := WeaponDamageScope(*it.CatalogID)
		// O PASSO ANDA O DADO ANTES DE TUDO, e a ordem é da p231: o crítico
		// multiplica "os dados de dano do ataque (INCLUINDO quaisquer aumentos
		// por passos)". Andar depois multiplicaria o dado de antes.
		steps := DamageStepsOf(r.ownItemMods(it, catalog), it.Equipped) +
			StatFor(effects, ModifierTarget{K: "damageStep", Scope: weaponScope}).Total
		nonLethal := StatFor(effects, ModifierTarget{K: "nonLethalDamage", Scope: "this"}).Total > 0
		cards = append(cards, shared.card(ch, it.Name, w,
			append([]string{weaponScope}, weaponKindScopes(w)...), damageThis, steps, nonLethal))
		if len(cards) == 2 {
			break
		}
	}
	return append(cards, r.unarmedCard(ch, shared))
}

// cardBasis é o que toda carta da ficha divide: os bônus globais e os atributos
// que decidem Luta, Pontaria e o dano.
type cardBasis struct {
	effects    ItemEffects
	attackAll  TotalContribs
	damageAll  TotalContribs
	forTotal   int
	dexTotal   int
	hasFinesse bool
	load       LoadBreakdown
}

func newCardBasis(ch Character, effects ItemEffects) cardBasis {
	return cardBasis{
		effects:    effects,
		attackAll:  totalContribsFor(effects, ModifierTarget{K: "attack", Scope: "all"}),
		damageAll:  totalContribsFor(effects, ModifierTarget{K: "damage", Scope: "all"}),
		forTotal:   effectiveAttribute(ch, "strength", effects),
		dexTotal:   effectiveAttribute(ch, "dexterity", effects),
		hasFinesse: parseChoiceSet(ch.ClassPowers).has["acuidade-com-arma"],
		// A carta de arma resolve Luta/Pontaria, que a penalidade de armadura
		// nunca alcança (p153) — a carga entra por completude, para esta chamada
		// não depender de saber quais perícias ficam de fora.
		load: loadBreakdownOf(ch, inventorySlotsTotal(ch, effects)),
	}
}

// card monta uma carta: a arma, os ESCOPOS que valem só para ela (o id do
// catálogo, o tipo de arma, o desarmado) e os passos que já são dela.
func (b cardBasis) card(ch Character, name string, w *WeaponStats, scopes []string,
	own TotalContribs, steps int, nonLethal bool) WeaponCard {
	parts := []TotalContribs{own}
	for _, scope := range scopes {
		parts = append(parts, totalContribsFor(b.effects, ModifierTarget{K: "damage", Scope: scope}))
	}
	ownDamage := ownDamageOf(parts...)
	skill, attribute := "Luta", "strength"
	strDamage := b.forTotal
	if w.Purpose == "ranged" {
		skill, attribute = "Pontaria", "dexterity"
		strDamage = 0
	} else {
		// Finesse (Adaga / Acuidade com Arma): use Destreza when it beats Força.
		dexAttack, dexDamage := weaponDexUse(w, b.hasFinesse, b.forTotal, b.dexTotal)
		if dexAttack {
			attribute = "dexterity"
		}
		if dexDamage {
			strDamage = b.dexTotal
		}
	}
	// weaponSkillState returns the stored Luta row (attribute strength); force
	// the resolved attribute so a finessed melee attack sums Destreza (ALE-31).
	state := weaponSkillState(ch, skill, attribute)
	state.Attribute = attribute
	ex := expertiseBreakdown(ch, state, b.effects, b.load)
	damage := DamageStep(w.Damage, steps+StatFor(b.effects, ModifierTarget{K: "damageStep", Scope: "all"}).Total)
	critMult := w.CritMult + StatFor(b.effects, ModifierTarget{K: "critMult"}).Total
	for _, scope := range scopes {
		critMult += StatFor(b.effects, ModifierTarget{K: "critMult", Scope: scope}).Total
	}
	return WeaponCard{
		Name:          name,
		Skill:         skill,
		Attribute:     ex.Attribute,
		Attack:        ex.Total + b.attackAll.Total,
		Expertise:     ex,
		AttackAll:     b.attackAll,
		Damage:        damage,
		DamageType:    w.Type,
		StrDamage:     strDamage,
		DamageBonus:   strDamage + b.damageAll.Total + ownDamage.Total,
		DamageAll:     b.damageAll,
		DamageOwn:     ownDamage,
		CritRange:     threatRangeOf(w.CritRange, b.effects, scopes),
		CritMult:      critMult,
		NonLethal:     nonLethal,
		ExtraDamage:   b.effects.ExtraDamage,
		Announces:     b.effects.Announced,
		CriticalBonus: b.effects.CriticalBonus[targetKey(ModifierTarget{K: "damage"})],
	}
}

// weaponDexUse mirrors weapon-cards.ts weaponDexUse: whether a wielded weapon may
// use Destreza instead of Força on attack/damage (T20 p145). Only when DES beats
// FOR (the rule is optional, so the sheet takes the better). Attack finesse = the
// weapon's inherent flag (Adaga) OR the Acuidade power on a light-melee/thrown/
// ágil weapon; damage finesse is Acuidade-only. Ranged never applies (ALE-31).
func weaponDexUse(w *WeaponStats, hasFinesse bool, forTotal, dexTotal int) (attack, damage bool) {
	if w.Purpose == "ranged" || dexTotal <= forTotal {
		return false, false
	}
	finesse := hasFinesse &&
		((w.Hand == "light" && w.Purpose == "melee") || w.Purpose == "thrown" || hasTrait(w.Traits, "agil"))
	return w.Finesse || finesse, finesse
}

// DamageStepsOf soma os passos que estes modificadores dão À ARMA que os carrega.
//
// Do item, e não do acumulado — que é o contrário do `damage:this` logo acima,
// e de propósito. Aquele soma um BÔNUS, e a imprecisão custa um ponto na carta
// da arma vizinha; um passo troca o DADO, e um machado comum anunciando 1d12
// porque a espada da outra mão é de adamante seria a carta mentindo sobre a
// arma que a pessoa está olhando.
//
// A condição é avaliada com o `conditionMet` de sempre: o `wielded` do material
// vale porque esta arma está na mão, e um passo preso a flag ou a circunstância
// fica de fora até alguém escrever como ele liga.
//
// Exportada porque a MOCHILA pergunta o mesmo: o crachá do cartão diz o dado da
// arma com as sobreposições DELA, e a conta tem de ser a mesma da carta de
// combate — duas somas de passo seriam duas respostas para o mesmo dado.
//
// @example DamageStepsOf(adamante.Modifiers, &empunhada) // 1
func DamageStepsOf(own []Modifier, equipped *string) int {
	steps := 0
	for _, m := range own {
		if targetKey(m.Target) != "damageStep:this" || !conditionMet(m, equipped) {
			continue
		}
		steps += m.Amount
	}
	return steps
}

func hasTrait(traits []string, t string) bool {
	for _, x := range traits {
		if x == t {
			return true
		}
	}
	return false
}

// weaponSkillState mirrors expertise.ts expertiseStateFor: the stored Luta/Pontaria
// row if the character has one, else a default untrained state for the skill.
func weaponSkillState(ch Character, name, attribute string) CharacterExpertise {
	for _, ex := range ch.Expertises {
		if ex.Name == name {
			return ex
		}
	}
	return CharacterExpertise{Name: name, Attribute: attribute, Trained: false}
}

// threatRangeOf aplica à margem de ameaça o que o item declara (ALE-411).
//
// Aqui morava `w.CritRange` cru, e o modificador de `critRange` só chegava à
// ABA EFEITOS: a melhoria Precisa custava 300 PO, imprimia "+1 margem de
// ameaça" na ficha e o ataque continuava ameaçando em 19. É o defeito da
// ALE-406 outra vez, e ele só virou regra errada quando a ALE-364 deu ao
// `critRange` um consumidor.
//
// # O FATOR multiplica a LARGURA, e não o número
//
// "A margem de ameaça da arma duplica. Por exemplo, uma espada longa ameaçadora
// tem margem de ameaça 17" (p335). Dobrar o 19 daria 38; o que dobra é quantos
// resultados ameaçam — a espada longa ameaça em 19 e 20, são dois, dobram para
// quatro, e quatro resultados a partir de 20 começam em 17.
//
// # E a ORDEM é do livro, na mesma frase
//
// "Efeitos que duplicam a margem de ameaça são aplicados ANTES de quaisquer
// efeitos que a aumentem" (p335). Invertida, a espada longa ameaçadora e precisa
// daria 16 em vez de 15 — dobrar depois de somar dobra o ponto somado junto.
func threatRangeOf(base int, effects ItemEffects, scopes []string) int {
	width := 21 - base
	if factor, has := effects.Factors[targetKey(ModifierTarget{K: "critRange"})]; has {
		width = factor.Applied(width)
	}
	width += StatFor(effects, ModifierTarget{K: "critRange"}).Total
	// A margem que vale só para UM tipo de ataque — o Golpe Cruel, "com ataques
	// desarmados" (p77) — entra só na carta desse tipo.
	for _, scope := range scopes {
		width += StatFor(effects, ModifierTarget{K: "critRange", Scope: scope}).Total
	}
	// O PISO É 2, e não é arbitrário: "um 1 natural sempre é uma falha" (p221),
	// então um resultado 1 na faixa é um número que nunca pode ameaçar. Sem o
	// piso, margem suficiente faria a carta anunciar uma faixa que inclui o
	// natural 1 — e o `ResolveAttack` a recusaria em silêncio.
	if width > 19 {
		width = 19
	}
	if width < 1 {
		width = 1
	}
	return 21 - width
}
