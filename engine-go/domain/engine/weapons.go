package engine

import (
	"slices"
	"strconv"
	"strings"
)

// WeaponCard is one wielded-weapon formula card (combat-magic-stats.tsx
// WeaponFormulaCards): attack = the weapon's attack perícia (Luta melee, Pontaria
// thrown/fired) + global attack mods; damage adds Força for melee/thrown (not
// fired) + global damage mods. Go owns the NUMBERS; the structural row labels
// (½ nível, FOR, Treino) and the crit string are applied on the front. Per-weapon
// mods (desbalanceada, Certeira…) ride the weapon scope inside effects, so they
// land on this weapon's cards through `AttackOwn` — no scope:'this' added here.
// ExtraDamage é uma parcela de dano de outro tipo, somada por um encanto.
type ExtraDamage struct {
	Dice string `json:"dice"` // "1d6"
	Type string `json:"type"` // "fogo", "frio", "ácido"…
	// Scope é a arma de que o dado é: o encanto Flamejante da espada não queima pela adaga. Vazio vale
	// em toda carta. Fica fora do JSON porque a carta já sai filtrada (`diceOfThisCard`).
	Scope string `json:"-"`
	// Range é a faixa a que o dado se limita no ataque À DISTÂNCIA: "curto" é o +1d8 da Escaramuça, que vale
	// "em alcance curto" (p51). Vazio vale a qualquer distância. Quem o tira do golpe é a linha
	// `TargetBeyondShortRange`, quando o tabuleiro mede o alvo além de 9m.
	Range string `json:"range,omitempty"`
	// SourceID é o poder ou item de onde o dado veio: o Roubo de Mana conta os dados do Ataque
	// Furtivo (p74), e o "+4d6" sozinho não diz de quem é. Fora do fio, como o `Scope`: quem o lê
	// é a proposta, na carta viva.
	SourceID string `json:"-"`
}

// SneakAttackPowerID é o Ataque Furtivo do ladino (p73), cujo dado cresce +1d6 a cada dois níveis.
const SneakAttackPowerID = "class.ladino.ataque-furtivo-1d6"

// SneakDice são os d6 do Ataque Furtivo que esta carta rola, com o interruptor ligado: zero
// quando ele está desligado ou a ficha não o tem.
//
//	card.SneakDice() // 4, no ladino 7 com o furtivo ligado
func (card WeaponCard) SneakDice() int {
	total := 0
	for _, die := range card.ExtraDamage {
		if die.SourceID != SneakAttackPowerID {
			continue
		}
		count, _, _ := strings.Cut(die.Dice, "d")
		if n, err := strconv.Atoi(count); err == nil {
			total += n
		}
	}
	return total
}

// IsMelee diz se esta arma bate CORPO A CORPO.
//
// Pela PERÍCIA e não por uma lista de nomes: quem decide é o catálogo, e o
// cartão já carrega a resposta dele. Ela existe porque `Skill != "Luta"` tinha
// três sítios — a manobra, o Golpe Mágico e a lista de armas da ficha —, e uma
// comparação de texto repetida é a que erra a grafia num deles.
func (w WeaponCard) IsMelee() bool { return w.Skill == "Luta" }

type WeaponCard struct {
	Name string `json:"name"`
	// CatalogID é o item do catálogo, vazio no desarmado. Não vai para o fio:
	// quem o lê é o rolar de novo do Mestre em Arma, que escolhe uma arma do
	// catálogo (p66, ALE-508).
	CatalogID string `json:"-"`
	// Natural diz que esta é uma arma natural, e o que ela dá no ataque extra. Fora do fio:
	// só o gesto de atacar e as Ações a leem.
	Natural *NaturalStrike `json:"-"`
	// Style é o ataque adicional que um estilo ligado dá a esta arma (p125). Fora do fio, como o
	// `Natural`.
	Style *StyleStrike `json:"-"`
	// PaidExtras são os ataques pagos que um poder dá depois de agredir — o Ataque Extra, o Golpe Relâmpago
	// (`paid_extra_strike.go`). Fora do fio, como o `Natural`.
	PaidExtras []PaidExtraStrike `json:"-"`
	// Unarmed marca a carta do ataque desarmado, que é a única que o Golpe Relâmpago alcança (p76).
	Unarmed bool `json:"-"`
	// DamageVariants são as linhas do dano desarmado, no fio para o golpe congelado — ver `DamageVariant`.
	DamageVariants []DamageVariant `json:"damageVariants,omitempty"`
	// SwitchesDamageFreely é o ataque desarmado do Estilo Desarmado: letal ou não letal, "sem
	// penalidades" (p128). Fora do fio, porque só o pedido de atacar o lê.
	SwitchesDamageFreely bool `json:"-"`
	// ItemID e Doses são a linha da mochila e o que a Injeção alquímica guarda (p165).
	// Não vão para o fio, porque só o gesto de atacar e a tela os leem.
	ItemID int64    `json:"-"`
	Doses  []string `json:"-"`
	Skill  string   `json:"skill"` // "Luta" | "Pontaria"
	// OtherUse é a mesma arma usada do outro jeito, nulo se ela só tem um: a lança
	// que voa, a azagaia que bate (ver `otherUseOf`). Não vai para o fio, porque só
	// o gesto de atacar e a conta da ficha a leem.
	OtherUse *WeaponCard `json:"-"`
	// Improvised é o uso que a arma não foi feita para ter, e que por isso paga a penalidade: a espada
	// arremessada, a azagaia corpo a corpo. Fora do fio, como o `OtherUse`.
	Improvised bool `json:"-"`
	// RefusesBoost diz que o impulso pedido não vale neste uso: o Golpe Divino e a
	// arma à distância. Não vai para o fio, porque só o pedido de atacar o lê.
	RefusesBoost bool `json:"-"`
	// BoostPM é o PM que o impulso pedido custa NESTA arma: o mesmo em toda carta, salvo a arma Harmonizada
	// para aquele poder, que o corta (p165). Zero na carta sem impulso. Fora do fio, como o `RefusesBoost`.
	BoostPM int `json:"-"`
	// BoostBuys é o que o impulso pedido compra NESTA arma, escrito ("+2d12"): a Arma Sagrada troca o dado do
	// Golpe Divino só na arma preferida da divindade (p82), e a tira de cada linha tem de dizer o dela.
	BoostBuys string `json:"-"`
	// ShootsIntoMelee é o Disparo Preciso (p125): atira em combate corpo a corpo
	// sem a penalidade. Não vai para o fio, porque só o ataque o lê.
	ShootsIntoMelee bool               `json:"-"`
	Attribute       string             `json:"attribute"` // the skill perícia's attribute
	Attack          int                `json:"attack"`
	Expertise       ExpertiseBreakdown `json:"expertise"`
	AttackAll       TotalContribs      `json:"attackAll"`
	// AttackOwn é o ataque que vale só para esta arma — o Foco em Arma.
	AttackOwn TotalContribs `json:"attackOwn,omitzero"`
	Damage    string        `json:"damage"` // dice, e.g. "1d8"
	// DexBy diz quem pôs a Destreza no lugar da Força no ATAQUE: a própria arma
	// (a adaga, p146) ou a Acuidade com Arma. Vazio quando é a Força. DexDamage
	// diz que ela entrou também no DANO, o que só a Acuidade faz — e é o que
	// deixa a conta da ficha rotular a parcela certa. Fora do fio de propósito:
	// é explicação para a tela, e o fio já tem o número.
	DexBy string `json:"-"`
	// AttributeSwapNote é a frase da licença que trocou o atributo do ATAQUE — "Armeiro: Inteligência no lugar
	// de Força" (p68), "Balística: Inteligência no lugar de Destreza" (p69) —; o atributo trocado é o próprio
	// `Attribute`. DamageSwapAttribute e DamageSwapNote são o atributo e a frase da que trocou o do DANO (o
	// Armeiro): a licença aceita qualquer atributo, e a tela escreve o que a carta trouxe. Vazios quando nada
	// trocou. Fora do fio, como o `DexBy`: só a ficha os lê, para escrever o atributo e o porquê.
	AttributeSwapNote   string `json:"-"`
	DamageSwapAttribute string `json:"-"`
	DamageSwapNote      string `json:"-"`
	DexDamage           bool   `json:"-"`
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
	// IgnoresDamageReduction é quanto da RD do alvo este golpe ignora: o desarmado do Punhos de Adamante (p77), o
	// Ataque Especial com o Romper Resistências (p66). Ver `ignored_reduction.go`.
	IgnoresDamageReduction int `json:"ignoresDamageReduction,omitempty"`
	// RerollsLowDice é o Destruidor (p41, p65): quem ataca pode rolar de novo os 1 e 2 dos dados DESTA arma.
	// Vai no JSON porque quem confere é o dano rolado depois, a partir da carta congelada. Ver `destroyer.go`.
	RerollsLowDice bool `json:"rerollsLowDice,omitempty"`
	// CriticalBonus é o dano que só existe no acerto crítico — os "+10 pontos
	// de dano" do encanto Dilacerante (p336). É bônus numérico, então não
	// multiplica, e fora do crítico ele não existe.
	CriticalBonus int `json:"criticalBonus,omitempty"`
	// AttributeDamage é o atributo somado no dano: a Força corpo a corpo e no arremesso, a Destreza pela Acuidade
	// com Arma, ou o que a licença `weaponAttribute` trocou (o Armeiro, p68). Zero na arma de disparo (p142).
	AttributeDamage int           `json:"attributeDamage"`
	DamageBonus     int           `json:"damageBonus"` // attributeDamage + damageAll.total
	DamageAll       TotalContribs `json:"damageAll"`
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
	// IgnoresSituations são as linhas da Tabela 5-3 que esta arma não sofre: a Caçadora anula "penalidades por
	// camuflagem leve e total e por cobertura leve" (p335). Viaja no JSON porque o golpe congelado as lê.
	IgnoresSituations []IgnoredSituation `json:"ignoresSituations,omitempty"`
}

// IgnoredSituation é uma linha da Tabela 5-3 que a arma não sofre, e QUEM a anula: o mestre lê "cobertura leve
// anulada pela Caçadora" na linha do ataque, e não a ausência da cobertura que ele pintou.
type IgnoredSituation struct {
	Row SpecialSituation `json:"row"`
	By  string           `json:"by"`
}

// ownContribsOf junta as parcelas que valem só para esta arma, numa decomposição
// — do dano ou do ataque.
func ownContribsOf(parts ...TotalContribs) TotalContribs {
	out := TotalContribs{Contributions: []BreakdownContribution{}}
	for _, p := range parts {
		out.Total += p.Total
		out.Contributions = append(out.Contributions, p.Contributions...)
	}
	return out
}

// ComputeWeaponCards são as cartas de ataque de um personagem: as armas empunhadas, o desarmado
// e as armas naturais (`naturalCards`), nessa ordem.
//
// It resolves the wielded-weapon cards for a raw Character under
// the given active conditionals (Fúria's global attack/damage mods land in
// attackAll/damageAll). Only catalog weapons in a hand slot, capped at two, so the
// index of a card is the index of the weapon; the other use rides inside the card.
func (r *Ruleset) ComputeWeaponCards(ch Character, activeConditionals map[string]bool) []WeaponCard {
	return r.weaponCardsWith(ch, activeConditionals, nil)
}

// weaponCardsWith são as cartas com fontes a mais — ver `sheetEffectsWith`.
func (r *Ruleset) weaponCardsWith(ch Character, activeConditionals map[string]bool, extra []ActiveItem) []WeaponCard {
	effects := r.sheetEffectsWith(ch, activeConditionals, extra)
	shared := newCardBasis(ch, effects)
	// O dano que a melhoria, o material ou o encanto dão À ARMA chega no escopo dela (`onThisWeapon`), e
	// cada carta o lê pelos próprios escopos. Aqui morava um leitor do `this` que toda carta somava: era
	// por ele que a Cruel da espada subia a adaga da outra mão.

	cards := []WeaponCard{}
	for _, it := range ch.Items {
		// "Na forma selvagem você não pode […] empunhar itens" (p63).
		if ch.WildShape != nil {
			break
		}
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
		cards = append(cards, r.itemCard(ch, it, catalog, *it.Equipped, effects, shared))
		if len(cards) == 2 {
			break
		}
	}
	r.offerStyleStrikes(ch, activeConditionals, effects, cards)
	cards = append(append(cards, r.unarmedCard(ch, shared)), r.naturalCards(ch, shared, activeConditionals)...)
	offerPaidExtraStrikes(cards, r.paidExtraStrikes(ch, effects.AbilityCostRules()))
	return cards
}

// cardBasis é o que toda carta da ficha divide: os bônus globais e os atributos
// que decidem Luta, Pontaria e o dano.
type cardBasis struct {
	effects    ItemEffects
	attackAll  TotalContribs
	damageAll  TotalContribs
	forTotal   int
	dexTotal   int
	intTotal   int
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
		intTotal:   effectiveAttribute(ch, "intelligence", effects),
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
	steps int, nonLethal bool) WeaponCard {
	if slices.Contains(scopes, MeleeScope) || (slices.Contains(scopes, ThrownScope) && b.effects.Flags[savannaRageFlag]) {
		scopes = append(slices.Clone(scopes), RageScope)
	}
	// O PASSO DE DANO DE CADA ESCOPO da carta, como o multiplicador logo abaixo: o da arma (`weapon:<id>`), o
	// do desarmado, e o de uma família (o arremesso do Arremessador, p28).
	for _, scope := range scopes {
		steps += StatFor(b.effects, ModifierTarget{K: "damageStep", Scope: scope}).Total
	}
	parts := []TotalContribs{}
	for _, scope := range scopes {
		parts = append(parts, totalContribsFor(b.effects, ModifierTarget{K: "damage", Scope: scope}))
	}
	ownDamage := ownContribsOf(parts...)
	// O ATAQUE DESTA ARMA, pelos mesmos escopos do dano: o Foco em Arma é +2 no
	// `weapon:<id>` escolhido (p128, ALE-510).
	attackParts := []TotalContribs{}
	for _, scope := range scopes {
		attackParts = append(attackParts, totalContribsFor(b.effects, ModifierTarget{K: "attack", Scope: scope}))
		// "Sofre –5 nos testes de ataque" (p106). Mora aqui e não em modificador de
		// item porque o escopo é de todas as armas marcadas: duas na mão somariam −10.
		if scope == OversizedWeaponScope {
			attackParts = append(attackParts, TotalContribs{
				Total:         OversizedWeaponPenalty,
				Contributions: []BreakdownContribution{{Source: "arma de categoria maior", Amount: OversizedWeaponPenalty}},
			})
		}
	}
	ownAttack := ownContribsOf(attackParts...)
	if len(ownAttack.Contributions) == 0 {
		// Nulo e não vazio: a carta sem ataque próprio sai do fio como antes.
		ownAttack = TotalContribs{}
	}
	skill, attribute := "Luta", "strength"
	attributeDamage := b.forTotal
	dexBy, dexDamageUsed := "", false
	if w.Purpose == "ranged" {
		skill, attribute = "Pontaria", "dexterity"
		attributeDamage = 0
	} else if w.Purpose == "thrown" {
		// A arma de arremesso é de ataque à distância: Pontaria no teste e Força no dano (p142). O Arremesso
		// Potente troca a Destreza pela Força no teste, e a Acuidade a Força pela Destreza no dano (p124).
		// Os dois dizem "pode", e a ficha fica com a maior.
		skill, attribute = "Pontaria", "dexterity"
		if b.effects.Flags["arremesso-potente"] && b.forTotal > b.dexTotal {
			attribute = "strength"
		}
		if b.hasFinesse && b.dexTotal > b.forTotal {
			attributeDamage, dexDamageUsed = b.dexTotal, true
		}
	} else {
		// Finesse (Adaga / Acuidade com Arma): use Destreza when it beats Força.
		dexAttack, dexDamage := weaponDexUse(w, b.hasFinesse, b.forTotal, b.dexTotal)
		if dexAttack {
			attribute = "dexterity"
			dexBy = DexByFinesse
			if w.Finesse {
				dexBy = DexByWeapon
			}
		}
		if dexDamage {
			attributeDamage = b.dexTotal
		}
		dexDamageUsed = dexDamage
	}
	attribute, attackLicense := b.licensedWeaponAttribute(ch, w, "attack", attribute, scopes)
	if attackLicense.Source != "" {
		dexBy = ""
	}
	// AS DUAS CONTAS SÃO INDEPENDENTES (decisão do dono): o Armeiro oferece a Inteligência "em vez de Força nos
	// testes de ataque e rolagens de dano" (p68), e cada conta fica com o maior que lhe é permitido. A adaga ataca
	// pela Destreza por regra da arma e ainda soma a Força no dano — que a Inteligência pode substituir. A arma de
	// disparo não soma atributo no dano (p142), e não há o que trocar.
	damageSwapAttribute, damageSwapNote := "", ""
	if w.Purpose != "ranged" {
		damageBase := "strength"
		if dexDamageUsed {
			damageBase = "dexterity"
		}
		if damageAttribute, damageLicense := b.licensedWeaponAttribute(ch, w, "damage", damageBase, scopes); damageLicense.Source != "" {
			attributeDamage, dexDamageUsed = effectiveAttribute(ch, damageAttribute, b.effects), false
			damageSwapAttribute, damageSwapNote = damageAttribute, damageLicense.Note
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
		Name:  name,
		Skill: skill,
		// "sem sofrer a penalidade de –5 no teste de ataque": o Disparo Preciso (p125), e o Grudar o Cano com a
		// arma de fogo contra o adjacente (p48), quando o jogador liga o interruptor.
		ShootsIntoMelee: b.effects.Flags["disparo-preciso"] || shootsIntoMeleeHere(b.effects, scopes),
		// O PÃO E CIRCO: "pode escolher causar dano não letal sem sofrer a penalidade de –5" (p91). Só a arma
		// letal: ficar letal com a não letal continua pagando a linha da Tabela 5-3.
		SwitchesDamageFreely:   b.effects.Flags["pao-e-circo"] && !nonLethal,
		Attribute:              ex.Attribute,
		Attack:                 ex.Total + b.attackAll.Total + ownAttack.Total,
		AttackOwn:              ownAttack,
		Expertise:              ex,
		AttackAll:              b.attackAll,
		Damage:                 damage,
		DamageType:             w.Type,
		AttributeDamage:        attributeDamage,
		DexBy:                  dexBy,
		AttributeSwapNote:      attackLicense.Note,
		DamageSwapAttribute:    damageSwapAttribute,
		DamageSwapNote:         damageSwapNote,
		DexDamage:              dexDamageUsed,
		DamageBonus:            attributeDamage + b.damageAll.Total + ownDamage.Total,
		DamageAll:              b.damageAll,
		DamageOwn:              ownDamage,
		CritRange:              threatRangeOf(w.CritRange, b.effects, scopes),
		CritMult:               critMult,
		NonLethal:              nonLethal,
		ExtraDamage:            diceOfThisCard(b.effects.ExtraDamage, scopes),
		Announces:              announcesOfThisCard(b.effects.Announced, scopes),
		CriticalBonus:          b.effects.CriticalBonus[targetKey(ModifierTarget{K: "damage"})],
		IgnoresDamageReduction: ignoredReductionOf(b.effects, scopes),
		RerollsLowDice:         rerollsLowDiceOf(b.effects, scopes),
	}
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
	// O fator da arma, como o da Ameaçadora: ele também vem antes de somar.
	for _, scope := range scopes {
		if factor, has := effects.Factors[targetKey(ModifierTarget{K: "critRange", Scope: scope})]; has {
			width = factor.Applied(width)
		}
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
	// natural 1 — e o `JudgeAttack` a recusaria em silêncio.
	if width > 19 {
		width = 19
	}
	if width < 1 {
		width = 1
	}
	return 21 - width
}
