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
	// AttributeBlockedBy é o ITEM que calou a Destreza — "Cota de malha" —, e
	// vem vazio quando ela entrou.
	//
	// Quem o diz é o motor porque a tela não tem como saber: ela vê um
	// `Attribute` vazio e não a fonte que o esvaziou. Enquanto o rótulo era
	// escrito à mão, ele dizia "bloqueada por armadura pesada" para uma
	// armadura de COURO (ALE-444) — e nomear o item é melhor que acertar a
	// categoria, porque a ficha pode ter mais de uma peça vestida.
	AttributeBlockedBy string `json:"attributeBlockedBy"`
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
	melee := StatFor(e, ModifierTarget{K: "defense", Scope: "melee"})
	ranged := StatFor(e, ModifierTarget{K: "defense", Scope: "ranged"})
	total := base + stat.Total
	contribs := stat.Contributions
	return DefenseBreakdown{
		Base:               base,
		ItemBonus:          stat.Total,
		Total:              total,
		Attribute:          attribute.Key,
		AttributeAmount:    attribute.Amount,
		AttributeSource:    attribute.Source,
		AttributeBlockedBy: attribute.BlockedBy,
		VsMelee:            total + melee.Total,
		VsRanged:           total + ranged.Total,
		Contributions:      withNoteContribs(concatContribs(contribs, melee.Contributions, ranged.Contributions)),
	}
}

// defenseAttributeChoice é o atributo que entrou na Defesa, com a procedência
// da licença quando não foi a Destreza. O zero dele é a Defesa SEM atributo —
// a armadura pesada bloqueando a Destreza e nenhum poder abrindo exceção.
type defenseAttributeChoice struct {
	Key    string
	Amount int
	Source string
	// BlockedBy é a FONTE que acendeu a bandeira que calou a Destreza. Ela fica
	// escrita mesmo quando um poder abre exceção e outro atributo entra: a
	// Destreza continua bloqueada, e é ela que a linha apagada do diálogo
	// explica.
	BlockedBy string
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
	if e.Flags["cannot-apply-dex-to-defense"] {
		best.BlockedBy = blockedDexSource(e)
	} else {
		best.Key, best.Amount = "dexterity", effectiveAttribute(ch, "dexterity", e)
	}
	for _, key := range AttributeKeys {
		allowed := StatFor(e, ModifierTarget{K: "defenseAttribute", Name: key})
		if len(allowed.Contributions) == 0 {
			continue
		}
		if amount := effectiveAttribute(ch, key, e); amount > best.Amount {
			best.Key, best.Amount = key, amount
			best.Source = allowed.Contributions[0].Source
		}
	}
	return best
}

// blockedDexSource nomeia a peça que acendeu a bandeira que cala a Destreza.
//
// A PRIMEIRA contribuição basta, e é a regra do livro que decide isso: só
// armadura pesada acende a bandeira (p152) e "não se pode vestir uma armadura
// sobre outra" (p152), então há no máximo uma. Listar todas pediria uma linha
// que o diálogo não tem onde pôr.
func blockedDexSource(e ItemEffects) string {
	return e.FlagSources["cannot-apply-dex-to-defense"]
}
