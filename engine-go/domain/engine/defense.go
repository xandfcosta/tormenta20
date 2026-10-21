package engine

// A DEFESA: 10 + o atributo que a ficha pode aplicar + os modificadores.
//
// Ela sai do `breakdowns.go` porque tem razão própria para mudar, e a razão
// apareceu: o atributo da Defesa deixou de ser "a Destreza, quando ela se
// aplica" e passou a ser uma ESCOLHA entre a Destreza e as que um poder
// autoriza no lugar dela (p69, p79, ALE-443).

type DefenseBreakdown struct {
	Base      int `json:"base"`
	ItemBonus int `json:"itemBonus"`
	Total     int `json:"total"`
	// Attribute é o atributo que ENTROU no `Base` — `dexterity` na ficha comum,
	// outro quando um poder permite usá-lo NO LUGAR dela (p69, p79), e VAZIO
	// quando a armadura pesada bloqueou todos.
	Attribute string `json:"attribute"`
	// AttributeAmount é quanto ele somou, dito pelo motor em vez de recalculado
	// pela tela: o `Base` é 10 + ele, e é essa subtração que o diálogo faz para
	// mostrar o 10 numa linha e o atributo na seguinte.
	AttributeAmount int `json:"attributeAmount"`
	// AttributeSource é a procedência da TROCA — "Autoconfiança". Vazia quando
	// quem entrou foi a Destreza, que não precisa de licença de ninguém.
	AttributeSource string `json:"attributeSource"`
	// Defesa contra ataques corpo a corpo e à distância. Iguais ao Total na
	// maioria das fichas; separam-se quando algo é DIRECIONAL, hoje só o Caído
	// (p394: −5 contra corpo a corpo, +5 contra à distância).
	VsMelee       int                     `json:"vsMelee"`
	VsRanged      int                     `json:"vsRanged"`
	Contributions []BreakdownContribution `json:"contributions"`
}

func defenseBreakdown(ch Character, e ItemEffects) DefenseBreakdown {
	stat := StatFor(e, ModifierTarget{K: "defense"})
	attribute := defenseAttribute(ch, e)
	base := 10 + attribute.Amount
	insolence := insolenciaDefense(ch, e)
	melee := StatFor(e, ModifierTarget{K: "defense", Scope: "melee"})
	ranged := StatFor(e, ModifierTarget{K: "defense", Scope: "ranged"})
	total := base + stat.Total + insolence
	contribs := stat.Contributions
	if insolence > 0 {
		contribs = concatContribs(contribs, []Contribution{
			{Source: "Insolência (p47)", BonusType: "untyped", Amount: insolence},
		})
	}
	return DefenseBreakdown{
		Base:            base,
		ItemBonus:       stat.Total + insolence,
		Total:           total,
		Attribute:       attribute.Key,
		AttributeAmount: attribute.Amount,
		AttributeSource: attribute.Source,
		VsMelee:         total + melee.Total,
		VsRanged:        total + ranged.Total,
		Contributions:   withNoteContribs(concatContribs(contribs, melee.Contributions, ranged.Contributions)),
	}
}

// defenseAttributeChoice é o atributo que entrou na Defesa, com a procedência
// da licença quando não foi a Destreza. O zero dele é a Defesa SEM atributo —
// a armadura pesada bloqueando a Destreza e nenhum poder abrindo exceção.
type defenseAttributeChoice struct {
	Key    string
	Amount int
	Source string
}

// defenseAttribute escolhe o atributo da Defesa: a MELHOR entre a Destreza
// (quando ela se aplica) e as que um poder permite usar NO LUGAR dela — a
// Inteligência da Blindagem e do Couraceiro (p69), o Carisma da Autoconfiança e
// da Armadura Brilhante (p79).
//
// # "Pode usar" é escolha, e a casa já responde a ela
//
// A arma com acuidade faz o mesmo (`weapons.go`): a regra é opcional, então a
// ficha toma a melhor e não pergunta nada. Perguntar seria pedir ao jogador uma
// decisão cuja resposta é sempre a mesma.
//
// # Trocar não é somar
//
// "Se fizer isso, não pode somar sua Destreza, mesmo que outras habilidades ou
// efeitos permitam isso" (p69) — então o atributo alternativo SUBSTITUI, e é
// por isso que esta função devolve UM e não uma soma.
//
// A alternativa só entra quando ganha do que já havia, e o `Amount` zero da
// escolha vazia é o que recusa uma Inteligência NEGATIVA de armadura pesada:
// "pode usar" não obriga ninguém a piorar a própria Defesa.
func defenseAttribute(ch Character, e ItemEffects) defenseAttributeChoice {
	best := defenseAttributeChoice{}
	if !e.Flags["cannot-apply-dex-to-defense"] {
		best = defenseAttributeChoice{Key: "dexterity", Amount: effectiveAttribute(ch, "dexterity", e)}
	}
	for _, key := range AttributeKeys {
		allowed := StatFor(e, ModifierTarget{K: "defenseAttribute", Name: key})
		if len(allowed.Contributions) == 0 {
			continue
		}
		if amount := effectiveAttribute(ch, key, e); amount > best.Amount {
			best = defenseAttributeChoice{Key: key, Amount: amount, Source: allowed.Contributions[0].Source}
		}
	}
	return best
}

// insolenciaDefense — Bucaneiro p47: "Você soma seu Carisma na Defesa, limitado
// pelo seu nível. Esta habilidade exige liberdade de movimentos; você não pode
// usá-la se estiver de armadura pesada ou na condição imóvel."
//
// O teto é o nível NA CLASSE (p226, "Limites de Nível"), com exemplo trabalhado
// na mesma página: "um bucaneiro de 2º nível com Car 3 soma +2 na Defesa".
//
// Vive aqui, e não como modificador de catálogo, porque o motor de itens não
// avalia `scale` fora de PV/PM e não tem noção de TETO — a Insolência precisa
// das duas coisas.
func insolenciaDefense(ch Character, e ItemEffects) int {
	if e.Flags["armadura-pesada"] || hasActiveCondition(ch, "imovel") {
		return 0
	}
	level := 0
	for _, entry := range ch.Classes {
		if entry.ClassName == "Bucaneiro" && entry.Level > level {
			level = entry.Level
		}
	}
	if level == 0 {
		return 0
	}
	return max(0, min(effectiveAttribute(ch, "charisma", e), level))
}

// hasActiveCondition reporta se a condição está ligada na ficha.
func hasActiveCondition(ch Character, id string) bool {
	for _, active := range parseStringArray(ch.ActiveConditions) {
		if active == id {
			return true
		}
	}
	return false
}
