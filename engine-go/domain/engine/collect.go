package engine

import (
	"fmt"
	"strings"
)

// The top of the COLLECTION layer: ActiveItemsFor (the entry point) and the
// item-level modifier assembly — catalog mods, overlays (melhorias/materiais),
// non-proficiency penalties (p142), weapon-attack mirroring, and the two opt-in
// homebrew rules. Together with collect_entities.go (race/origin/class/tormenta)
// it produces the []ActiveItem the resolution engine (ComputeItemEffects)
// consumes.

// vestedWear is the shared 'vested' wear-state pointer used by every
// non-equipment ActiveItem (race/origin/class/effect). Read-only.
var vestedWear = "vested"

const deformidadeExpertiseBonus = 2 // deformidade.ts DEFORMIDADE_PERICIA_BONUS

// homebrewVestedOK mirrors items/homebrew.ts HOMEBREW_VESTED_OK — esotéricos
// that may be WORN and still grant their empunhado bonus (opt-in).
var homebrewVestedOK = map[string]bool{"medalhao-de-prata": true}

// expertiseNamesSet mirrors expertises.ts EXPERTISE_NAMES — the valid perícia
// names Deformidade may target.
var expertiseNamesSet = toSet([]string{
	"Acrobacia", "Adestramento", "Atletismo", "Atuação", "Cavalgar", "Conhecimento",
	"Cura", "Diplomacia", "Enganação", "Fortitude", "Furtividade", "Guerra",
	"Iniciativa", "Intimidação", "Intuição", "Investigação", "Jogatina", "Ladinagem",
	"Luta", "Misticismo", "Nobreza", "Ofício", "Percepção", "Pilotagem", "Pontaria",
	"Reflexos", "Religião", "Sobrevivência", "Vontade",
})

// ActiveItemsFor: collect every active modifier
// source into []ActiveItem — the input the resolution engine consumes. Order is
// significativa: o despejo de paridade compara byte a byte.
func (r *Ruleset) ActiveItemsFor(ch Character) []ActiveItem {
	proficiencies := parseProficiencySet(ch.Proficiencies)
	items := []ActiveItem{}
	for _, it := range ch.Items {
		if it.Equipped == nil {
			continue
		}
		items = append(items, r.itemActiveItem(it, proficiencies))
	}
	for _, eff := range ch.ActiveEffects {
		mods := parseEffectModifiers(eff.Modifiers)
		if len(mods) == 0 {
			continue
		}
		scope := DurationLabel(eff.Scope)
		items = append(items, ActiveItem{
			SourceID:  eff.CatalogID,
			Source:    fmt.Sprintf("%s (%s)", r.appliedEffectName(eff.CatalogID, mods), scope),
			Equipped:  &vestedWear,
			Modifiers: mods,
		})
	}
	items = append(items, r.raceActiveItems(ch)...)
	if origin := r.originActiveItem(ch); origin != nil {
		items = append(items, *origin)
	}
	items = append(items, r.classActiveItems(ch)...)
	items = append(items, r.generalPowerActiveItem(ch)...)
	if tormenta := r.tormentaCarismaItem(ch); tormenta != nil {
		items = append(items, *tormenta)
	}
	if cond := conditionActiveItem(ch); cond != nil {
		items = append(items, *cond)
	}
	return applySilences(r.mesa.Silences, ch, append(items, r.campaignGrants(ch)...))
}

// conditionActiveItem builds the p394 status conditions as a synthetic
// ActiveItem, so their numeric penalties flow through the resolution engine and
// obey non-stacking like everything else.
//
// It is appended LAST, and the position is load-bearing: the oracle compares
// byte-equal, so moving it moves every downstream contribution list.
func conditionActiveItem(ch Character) *ActiveItem {
	ids := parseStringArray(ch.ActiveConditions)
	mods := []Modifier{}
	for _, id := range ids {
		mods = append(mods, conditionModifiers(id)...)
	}
	if len(mods) == 0 {
		return nil
	}
	// Sem id: a fonte é o CONJUNTO das condições ativas, não um verbete.
	return &ActiveItem{Source: "Condições", Equipped: &vestedWear, Modifiers: mods}
}

func conditionModifiers(id string) []Modifier {
	return conditionModifierTable[id]
}

// condMod builds a status-condition modifier (bonusType "condition": book p394
// "aplique apenas o mais severo" → resolveStack keeps the worst per target).
func condMod(target ModifierTarget, amount int) Modifier {
	return Modifier{Target: target, Amount: amount, BonusType: "condition"}
}

func condDefense(n int) Modifier { return condMod(ModifierTarget{K: "defense"}, n) }
func condAllSkills(n int) Modifier {
	return condMod(ModifierTarget{K: "expertiseAll"}, n)
}
func condSkill(name string, n int) Modifier {
	return condMod(ModifierTarget{K: "expertise", Name: name}, n)
}
func condByAttr(attr string, n int) Modifier {
	return condMod(ModifierTarget{K: "expertiseByAttribute", Attribute: attr}, n)
}
func condAttack(n int) Modifier {
	return condMod(ModifierTarget{K: "attack", Scope: "all"}, n)
}
func condForDesCon(n int) []Modifier {
	return []Modifier{condByAttr("strength", n), condByAttr("dexterity", n), condByAttr("constitution", n)}
}

// autoFailReflexosFlag marca a falha AUTOMÁTICA em Reflexos do Indefeso (p394).
// Booleano, não número: a ficha mostra "falha automática" na linha em vez de um
// total, porque virá-la em −5 inventaria uma regra mais branda que a do livro.
const autoFailReflexosFlag = "auto-fail-reflexos"

// condFactor é a condição que MULTIPLICA o alvo — ver `factor.go`. O `Amount`
// fica em zero de propósito: um modificador com fator não é parcela da pilha.
func condFactor(target ModifierTarget, num, den int) Modifier {
	return Modifier{Target: target, BonusType: "condition", Factor: &Ratio{Num: num, Den: den}}
}

// lentoMods e imovelMods são as duas reduções de MOVIMENTO do livro, e as duas
// são fator porque nenhuma é uma parcela:
//
//	"LENTO. Todas as formas de deslocamento do personagem são reduzidas à metade
//	(arredonde para baixo para o primeiro incremento de 1,5m)" (p395)
//	"IMÓVEL. Todas as formas de deslocamento do personagem são reduzidas a 0m."
//	(p394)
var (
	lentoMods  = []Modifier{condFactor(ModifierTarget{K: "displacement"}, 1, 2)}
	imovelMods = []Modifier{condFactor(ModifierTarget{K: "displacement"}, 0, 1)}
)

func condFlag(name string) Modifier {
	return Modifier{Target: ModifierTarget{K: "flag", Name: name}, Amount: 1, BonusType: "condition"}
}

// condDamageReduction — o Petrificado concede RD 8 (p394). Vai por modificador
// como qualquer outra fonte, e não cravado no cálculo de RD, para que item e
// magia possam conceder RD pelo mesmo caminho quando chegar a hora.
// condDefenseVs é a Defesa DIRECIONAL do Caído (p394). Escopo separado para não
// competir com a Defesa geral das outras condições — o livro diz, entre
// parênteses, que a dele é "cumulativa com outras condições".
func condDefenseVs(scope string, n int) Modifier {
	return Modifier{Target: ModifierTarget{K: "defense", Scope: scope}, Amount: n, BonusType: "condition"}
}

func condDamageReduction(n int) Modifier {
	return Modifier{Target: ModifierTarget{K: "damageReduction"}, Amount: n, BonusType: "condition"}
}

// condPmCost é o único modificador de condição que NÃO é penalidade em teste: o
// Alquebrado encarece as habilidades do personagem.
func condPmCost(n int) Modifier {
	return Modifier{Target: ModifierTarget{K: "pmCost"}, Amount: n, BonusType: "condition"}
}

func condIntSabCar(n int) []Modifier {
	return []Modifier{condByAttr("intelligence", n), condByAttr("wisdom", n), condByAttr("charisma", n)}
}

// As condições que a p394 usa para DEFINIR outras. Compor em vez de copiar os
// números é o que impede uma derivada de ficar com metade do efeito: copiada, a
// derivada fica com a metade que alguém transcreveu.
var (
	vulneravelMods   = []Modifier{condDefense(-2)}
	desprevenidoMods = []Modifier{condDefense(-5), condSkill("Reflexos", -5)}
	fracoMods        = condForDesCon(-2)
	debilitadoMods   = condForDesCon(-5)
	// "INDEFESO. O personagem fica desprevenido, MAS sofre −10 na Defesa, falha
	// automaticamente em testes de Reflexos […]" — o "mas" faz o −10 SUBSTITUIR o
	// −5 do desprevenido. A falha automática em Reflexos não é número: vai como
	// FLAG, que é o mecanismo do motor para efeito booleano.
	indefesoMods = []Modifier{condDefense(-10), condFlag(autoFailReflexosFlag)}
)

// withPlus devolve os modificadores de uma condição citada mais os próprios.
func withPlus(base []Modifier, extra ...Modifier) []Modifier {
	return append(append([]Modifier{}, base...), extra...)
}

// conditionModifierTable IS the source of the condition rule (p394). It was a
// duplicate of a TS table while there were two engines; with one engine there is
// nothing to keep in sync, and a diff here is a rule change to check in the book.
// Cada linha derivada cita o texto da p394 que a obriga.
var conditionModifierTable = map[string][]Modifier{
	"abalado":      {condAllSkills(-2)},
	"lento":        lentoMods,
	"imovel":       imovelMods,
	"apavorado":    {condAllSkills(-5)},
	"vulneravel":   vulneravelMods,
	"desprevenido": desprevenidoMods,
	"indefeso":     indefesoMods,
	"fraco":        fracoMods,
	"debilitado":   debilitadoMods,
	"frustrado":    condIntSabCar(-2),
	"esmorecido":   condIntSabCar(-5),

	// "ATORDOADO. O personagem fica desprevenido e não pode fazer ações."
	"atordoado": desprevenidoMods,
	// "SURPREENDIDO. O personagem fica desprevenido e não pode fazer ações."
	"surpreendido": desprevenidoMods,
	// "PARALISADO. Fica imóvel e indefeso […]"
	"paralisado": withPlus(indefesoMods, imovelMods...),
	// "INCONSCIENTE. O personagem fica indefeso e não pode fazer ações […]"
	"inconsciente": indefesoMods,
	// "PETRIFICADO. O personagem fica inconsciente e recebe redução de dano 8."
	"petrificado": withPlus(indefesoMods, condDamageReduction(8)),

	// "FATIGADO. O personagem fica fraco e vulnerável."
	"fatigado": withPlus(fracoMods, vulneravelMods...),
	// "EXAUSTO. O personagem fica debilitado, lento e vulnerável."
	"exausto": withPlus(debilitadoMods, append(append([]Modifier{}, vulneravelMods...), lentoMods...)...),
	// "CEGO. O personagem fica desprevenido e lento […] e sofre −5 em testes de
	// perícias baseadas em Força ou Destreza."
	//
	// O "e lento" ficou de fora por anos porque não havia como escrevê-lo: o
	// motor só somava, e meia velocidade não é uma parcela (ALE-390).
	"cego": withPlus(desprevenidoMods,
		append([]Modifier{condByAttr("strength", -5), condByAttr("dexterity", -5)}, lentoMods...)...),
	// "AGARRADO. O personagem fica desprevenido e imóvel, sofre −2 em testes de
	// ataque […]"
	"agarrado": withPlus(desprevenidoMods, condAttack(-2)),
	// "ENREDADO. O personagem fica lento, vulnerável e sofre −2 em testes de
	// ataque."
	"enredado": withPlus(vulneravelMods, condAttack(-2)),

	// "ALQUEBRADO. O custo em pontos de mana das habilidades do personagem
	// aumenta em +1." Aumento, não redução — soma normalmente (p226).
	"alquebrado": {condPmCost(1)},

	"ofuscado":  {condAttack(-2), condSkill("Percepção", -2)},
	"fascinado": {condSkill("Percepção", -5)},
	"surdo":     {condSkill("Iniciativa", -5)},
	// "CAÍDO. O personagem sofre −5 na Defesa contra ataques corpo a corpo e
	// recebe +5 na Defesa contra ataques à distância (cumulativos com outras
	// condições). Além disso, sofre −5 em ataques corpo a corpo" — que no motor
	// são testes de Luta.
	"caido": {
		condSkill("Luta", -5),
		condDefenseVs("melee", -5),
		condDefenseVs("ranged", 5),
	},
}

// itemActiveItem ports the per-item branch of activeItemsFor's map: base +
// overlay + material mods (ownMods), then penalties, mirrors, and homebrew, in
// the exact TS concatenation order.
func (r *Ruleset) itemActiveItem(it CharacterItem, prof map[string]bool) ActiveItem {
	var catalog *CatalogItem
	if it.CatalogID != nil {
		catalog = r.getCatalogItem(*it.CatalogID)
	}
	improvementIDs := parseStringArray(it.Improvements)

	ownMods := []Modifier{}
	if catalog != nil {
		ownMods = append(ownMods, catalog.Modifiers...)
	}
	for _, id := range improvementIDs {
		ownMods = append(ownMods, overlayModsWithProvenance(r.getCatalogItem(id))...)
	}
	if it.Material != nil {
		ownMods = append(ownMods, overlayModsWithProvenance(r.getCatalogItem(*it.Material))...)
	}

	mods := append([]Modifier{}, ownMods...)
	if catalog != nil {
		mods = append(mods, nonProficiencyPenalties(catalog, prof)...)
	}
	mods = append(mods, mirrorWeaponAttackMods(catalog, ownMods)...)
	mods = append(mods, equilibradaHomebrewMods(catalog, improvementIDs)...)
	mods = append(mods, vestedEsotericHomebrewMods(it.Equipped, catalog, ownMods)...)
	return ActiveItem{SourceID: catalogItemID(it), Source: it.Name, Equipped: it.Equipped, Modifiers: mods}
}

// overlayModsWithProvenance: an overlay's modifiers with the
// overlay NAME folded into each note (so breakdown rows name the melhoria/material).
func overlayModsWithProvenance(overlay *CatalogItem) []Modifier {
	if overlay == nil {
		return []Modifier{}
	}
	out := make([]Modifier, 0, len(overlay.Modifiers))
	for _, m := range overlay.Modifiers {
		nm := m
		if !strings.Contains(m.Note, overlay.Name) {
			if m.Note != "" {
				nm.Note = overlay.Name + ": " + m.Note
			} else {
				nm.Note = overlay.Name
			}
		}
		out = append(out, nm)
	}
	return out
}

// mirrorWeaponAttackMods: a weapon's own {attack,scope:this}
// mods mirrored onto its Luta/Pontaria perícia (T20 attacks are expertise tests).
func mirrorWeaponAttackMods(catalog *CatalogItem, ownMods []Modifier) []Modifier {
	if catalog == nil || catalog.Weapon == nil {
		return []Modifier{}
	}
	expertise := attackExpertiseFor(catalog.Weapon.Purpose)
	out := []Modifier{}
	for _, m := range ownMods {
		if m.Target.K != "attack" || m.Target.Scope != "this" {
			continue
		}
		condition := m.Condition
		if condition == nil {
			condition = &ModifierCondition{C: "wielded"}
		}
		note := m.Note
		if note == "" {
			note = "bônus desta arma"
		}
		out = append(out, Modifier{
			Target:    ModifierTarget{K: "expertise", Name: expertise},
			Amount:    m.Amount,
			BonusType: "untyped",
			Condition: condition,
			Note:      note,
		})
	}
	return out
}

// equilibradaHomebrewMods: an opt-in +2 that nets out a
// desbalanceada weapon's -2 when the Equilibrada melhoria is attached.
func equilibradaHomebrewMods(catalog *CatalogItem, improvementIDs []string) []Modifier {
	if catalog == nil || catalog.Weapon == nil {
		return []Modifier{}
	}
	if !contains(catalog.Weapon.Traits, "desbalanceada") {
		return []Modifier{}
	}
	if !contains(improvementIDs, "melhoria-equilibrada") {
		return []Modifier{}
	}
	return []Modifier{{
		Target:    ModifierTarget{K: "expertise", Name: attackExpertiseFor(catalog.Weapon.Purpose)},
		Amount:    2,
		BonusType: "untyped",
		Condition: &ModifierCondition{C: "context", Note: "Homebrew: Equilibrada anula a desbalanceada (-2 → 0)"},
		Note:      "anula desbalanceada",
	}}
}

// vestedEsotericHomebrewMods: a HOMEBREW_VESTED_OK esotérico
// worn (vested) keeps its wielded-gated bonuses behind one flagOn toggle.
func vestedEsotericHomebrewMods(equipped *string, catalog *CatalogItem, ownMods []Modifier) []Modifier {
	if equipped == nil || *equipped != "vested" {
		return []Modifier{}
	}
	if catalog == nil || !homebrewVestedOK[catalog.ID] {
		return []Modifier{}
	}
	out := []Modifier{}
	for _, m := range ownMods {
		if m.Condition == nil || m.Condition.C != "wielded" {
			continue
		}
		nm := m
		nm.Condition = &ModifierCondition{
			C:     "flagOn",
			Flag:  "homebrew-vestido-" + catalog.ID,
			Label: "Homebrew: esotérico vestido mantém o bônus (RAW exige empunhar, p159)",
		}
		out = append(out, nm)
	}
	return out
}

// nonProficiencyPenalties: T20 p142 penalties for using a
// weapon/armor/shield without the required proficiency.
func nonProficiencyPenalties(catalog *CatalogItem, prof map[string]bool) []Modifier {
	required := requiredProficiency(catalog)
	if required == "" || prof[required] {
		return []Modifier{}
	}
	if strings.HasPrefix(catalog.Category, "weapon-") {
		return weaponNonProficiency(catalog)
	}
	basePenalty := -1
	for _, m := range catalog.Modifiers {
		if m.Target.K == "armorPenalty" {
			basePenalty = m.Amount
			break
		}
	}
	return []Modifier{
		{
			Target:    ModifierTarget{K: "flag", Name: "cannot-apply-dex-to-defense"},
			Amount:    1,
			BonusType: "untyped",
			Condition: &ModifierCondition{C: "vested"},
			Note:      "sem proficiência",
		},
		{
			Target:    ModifierTarget{K: "expertiseAll"},
			Amount:    basePenalty,
			BonusType: "untyped",
			Condition: &ModifierCondition{C: "vested"},
			Note:      catalog.Name + " sem proficiência",
		},
	}
}

func weaponNonProficiency(catalog *CatalogItem) []Modifier {
	purpose := ""
	if catalog.Weapon != nil {
		purpose = catalog.Weapon.Purpose
	}
	return []Modifier{
		{
			Target:    ModifierTarget{K: "attack", Scope: "this"},
			Amount:    -5,
			BonusType: "untyped",
			Condition: &ModifierCondition{C: "wielded"},
			Note:      "sem proficiência",
		},
		{
			Target:    ModifierTarget{K: "expertise", Name: attackExpertiseFor(purpose)},
			Amount:    -5,
			BonusType: "untyped",
			Condition: &ModifierCondition{C: "wielded"},
			Note:      "sem proficiência: -5 em testes de ataque (p142)",
		},
	}
}

// attackExpertiseFor maps a weapon purpose to its attack perícia (T20 resolves
// attacks as expertise tests): melee → Luta, everything else → Pontaria.
func attackExpertiseFor(purpose string) string {
	if purpose == "melee" {
		return "Luta"
	}
	return "Pontaria"
}

// effectSourceName ports the item branch of entities/character/effect-source.ts:
// an ActiveEffect's display name. Spell/activation catalogs aren't primed in the
// engine (no seed effect needs them), so this covers the manual pool + item
// sources and falls back to the raw id — e cai no id cru como último recurso.
// appliedEffectName nomeia a PROCEDÊNCIA de um efeito aplicado.
//
// O `effectSourceName` sozinho não dá conta do efeito de MAGIA: o
// `engine.Catalogs` carrega itens, raças, origens e poderes, e não carrega
// magias — então a decomposição da aba Combate saía escrita
// "armadura-arcana (cena)", com o id na cara de quem lê a ficha (ALE-365).
//
// O nome do livro já está DENTRO do efeito, na nota do modificador que o
// catálogo transcreveu ("Armadura Arcana"), e é ela que entra aqui. Ensinar o
// motor a carregar as 198 magias resolveria também, e custaria uma entrada nova
// no despejo que alimenta o oráculo — preço alto para um rótulo que o dado já
// carrega.
func (r *Ruleset) appliedEffectName(catalogID string, mods []Modifier) string {
	if name := r.effectSourceName(catalogID); name != catalogID {
		return name
	}
	for _, m := range mods {
		if m.Note != "" {
			return m.Note
		}
	}
	return catalogID
}

func (r *Ruleset) effectSourceName(catalogID string) string {
	if catalogID == "manual-temp-hp" {
		return "PV temporários (manual)"
	}
	if item := r.getCatalogItem(catalogID); item != nil {
		return item.Name
	}
	return catalogID
}

// catalogItemID é o id do verbete de um item da ficha, ou vazio quando o item é
// custom — inventado pela pessoa, sem entrada no livro.
func catalogItemID(it CharacterItem) string {
	if it.CatalogID == nil {
		return ""
	}
	return *it.CatalogID
}
