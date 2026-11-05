package engine

import "strings"

// AS MELHORIAS DA MESA: o que o mestre cria para se colocar num item, ao lado das do livro.
//
// O item guarda o id dela na mesma lista, e por isso `itemOf` a devolve como um verbete: a coleta, a
// carta e a conta não aprendem nada de novo. Fora da mesa o id não resolve, e ela soma zero, como o
// item de outro mundo.

// TableImprovementPrefix separa o id dela de todo id do livro.
const TableImprovementPrefix = "melhoria-da-mesa-"

// TableImprovement é uma melhoria que o mestre criou. `Family` vazia cabe em qualquer item.
//
// @example engine.TableImprovement{ID: "melhoria-da-mesa-1", Name: "Insígnia", Applies: engine.EveryoneIn(), Family: "apparel", Effect: …}
type TableImprovement struct {
	ID      string
	Name    string
	Applies Selector
	Family  string
	Effect  RuleEffect
}

// TableImprovements são as melhorias da mesa. Quem desenha a lista do item decide quais oferecer: a
// que já está no item segue na lista, para poder sair, mesmo que o mestre a tenha dado a outra pessoa.
func (r *Ruleset) TableImprovements() []TableImprovement { return r.mesa.Improvements }

func (r *Ruleset) tableImprovementEntry(id string) *CatalogItem {
	if !strings.HasPrefix(id, TableImprovementPrefix) {
		return nil
	}
	for _, improvement := range r.mesa.Improvements {
		// Quem perdeu o direito para de somar: o mestre manda no que vale agora, e não no que valia
		// quando o jogador a colocou. Ela segue na lista do item, para poder sair.
		if improvement.ID == id && improvement.Applies.Matches(Character{ID: int(r.holder)}) {
			return &CatalogItem{
				ID: id, Name: improvement.Name, Category: "improvement", AppliesTo: familiesOf(improvement.Family),
				Modifiers: []Modifier{tableImprovementModifier(improvement)},
			}
		}
	}
	return nil
}

// tableImprovementModifier é a soma dela. O número de golpe leva o `this` das melhorias do livro, que
// o prende à arma em que ela está. `inUse` é o que a impede de valer na mochila: o máximo de PV e PM
// não olha se o item está em uso, só a condição do modificador.
//
// O TIPO DE BÔNUS É A PRÓPRIA MELHORIA, e não `untyped`: duas melhorias diferentes somam, porque
// cada uma disputa só consigo mesma, e a mesma, em dois itens ou repetida, conta uma vez. Com
// `untyped` a mesma melhoria em duas adagas somava duas vezes, porque as duas dividem o escopo.
func tableImprovementModifier(improvement TableImprovement) Modifier {
	target := improvement.Effect.Target
	if aimsAWeapon(target.K) || target.K == "critRange" {
		target.Scope = "this"
	}
	return Modifier{
		Target: target, Amount: improvement.Effect.Amount, BonusType: improvement.ID,
		Condition: &ModifierCondition{C: "inUse"},
	}
}

// hostedHere prende à arma o que a melhoria da mesa dá nela: o `this` do dano é lido em todas as
// cartas, e o da margem em nenhuma; o escopo da arma é lido só na dela. O ataque não entra, porque o
// espelho de `mirrorWeaponAttackMods` já o leva.
func hostedHere(overlay *CatalogItem, mods []Modifier, host *CatalogItem) []Modifier {
	if host == nil {
		return mods
	}
	// A melhoria da mesa que deixou de caber neste item para de somar, pela mesma razão do dono.
	if strings.HasPrefix(overlay.ID, TableImprovementPrefix) && len(overlay.AppliesTo) > 0 &&
		overlay.AppliesTo[0] != ItemFamilyOf(host.Category) {
		return []Modifier{}
	}
	out := make([]Modifier, 0, len(mods))
	for _, modifier := range mods {
		if modifier.Condition != nil && modifier.Condition.C == "inUse" && modifier.Target.Scope == "this" && modifier.Target.K != "attack" {
			modifier.Target.Scope = WeaponDamageScope(host.ID)
		}
		out = append(out, modifier)
	}
	return out
}

func familiesOf(family string) []string {
	if family == "" {
		return nil
	}
	return []string{family}
}

// ItemFamilyOf é a família do item pela categoria: arma, armadura, escudo ou vestuário, que é onde
// cai todo o resto. É a família em que melhoria e encanto cabem.
//
// @example engine.ItemFamilyOf("weapon-martial") // "weapon"
func ItemFamilyOf(category string) string {
	switch {
	case strings.HasPrefix(category, "weapon-"):
		return "weapon"
	case strings.HasPrefix(category, "armor-"):
		return "armor"
	case category == "shield":
		return "shield"
	}
	return "apparel"
}
