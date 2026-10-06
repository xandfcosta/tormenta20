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
	CritRange     int           `json:"critRange"`
	CritMult      int           `json:"critMult"`
	// NonLethal: TODO o dano desta arma é não letal (p236, p336). É propriedade
	// da ARMA e não parcela — a Piedosa diz "todo o dano causado", o que inclui
	// a Força e as parcelas dos outros encantos.
	NonLethal bool `json:"nonLethal,omitempty"`
}

// ComputeWeaponCards resolves the wielded-weapon cards for a raw Character under
// the given active conditionals (Fúria's global attack/damage mods land in
// attackAll/damageAll). Only catalog weapons in a hand slot, capped at two.
func (r *Ruleset) ComputeWeaponCards(ch Character, activeConditionals map[string]bool) []WeaponCard {
	effects := ApplyActiveConditionals(ComputeItemEffects(r.ActiveItemsFor(ch)), activeConditionals)
	attackAll := totalContribsFor(effects, ModifierTarget{K: "attack", Scope: "all"})
	damageAll := totalContribsFor(effects, ModifierTarget{K: "damage", Scope: "all"})
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
	forTotal := effectiveAttribute(ch, "strength", effects)
	dexTotal := effectiveAttribute(ch, "dexterity", effects)
	hasFinesse := parseChoiceSet(ch.ClassPowers).has["acuidade-com-arma"]
	// A carta de arma resolve Luta/Pontaria, que a penalidade de armadura nunca
	// alcança (p153) — a carga entra por completude, para esta chamada não
	// depender de saber quais perícias ficam de fora.
	load := loadBreakdownOf(ch, inventorySlotsTotal(ch, effects))

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
		skill, attribute := "Luta", "strength"
		strDamage := forTotal
		if w.Purpose == "ranged" {
			skill, attribute = "Pontaria", "dexterity"
			strDamage = 0
		} else {
			// Finesse (Adaga / Acuidade com Arma): use Destreza when it beats Força.
			dexAttack, dexDamage := weaponDexUse(w, hasFinesse, forTotal, dexTotal)
			if dexAttack {
				attribute = "dexterity"
			}
			if dexDamage {
				strDamage = dexTotal
			}
		}
		// weaponSkillState returns the stored Luta row (attribute strength); force
		// the resolved attribute so a finessed melee attack sums Destreza (ALE-31).
		state := weaponSkillState(ch, skill, attribute)
		state.Attribute = attribute
		ex := expertiseBreakdown(ch, state, effects, load)
		cards = append(cards, WeaponCard{
			Name:          it.Name,
			Skill:         skill,
			Attribute:     ex.Attribute,
			Attack:        ex.Total + attackAll.Total,
			Expertise:     ex,
			AttackAll:     attackAll,
			Damage:        w.Damage,
			StrDamage:     strDamage,
			DamageBonus:   strDamage + damageAll.Total + damageThis.Total,
			DamageAll:     damageAll,
			CritRange:     threatRangeOf(w.CritRange, effects),
			CritMult:      w.CritMult + StatFor(effects, ModifierTarget{K: "critMult"}).Total,
			NonLethal:     StatFor(effects, ModifierTarget{K: "nonLethalDamage", Scope: "this"}).Total > 0,
			ExtraDamage:   effects.ExtraDamage,
			CriticalBonus: effects.CriticalBonus[targetKey(ModifierTarget{K: "damage"})],
		})
		if len(cards) == 2 {
			break
		}
	}
	return cards
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
func threatRangeOf(base int, effects ItemEffects) int {
	width := 21 - base
	if factor, has := effects.Factors[targetKey(ModifierTarget{K: "critRange"})]; has {
		width = factor.Applied(width)
	}
	width += StatFor(effects, ModifierTarget{K: "critRange"}).Total
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
