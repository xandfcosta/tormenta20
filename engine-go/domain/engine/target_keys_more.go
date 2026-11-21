package engine

// moreTargetKeys são as chaves dos alvos que nasceram depois de o `targetKey` passar do teto de
// linhas: o arquivo dele está na linha de base, e a dívida velha só encolhe. Alvo que ninguém
// conhece continua sendo "", que não alcança pilha nenhuma.
func moreTargetKeys(t ModifierTarget) string {
	switch t.K {
	case "abilityPmCost":
		// O CUSTO DE TODA HABILIDADE, ou só o das de uma CLASSE (`Name`): o Alquebrado vale para qualquer uma
		// (p394), e a metade do Artista Completo, para as de bardo (p45).
		if t.Name != "" {
			return abilityPmCostTarget + ":" + t.Name
		}
		return abilityPmCostTarget
	case "expertiseAttribute":
		// A PERÍCIA E O ATRIBUTO NO NOME, e a licença não soma nada: como o `defenseAttribute`,
		// ela AUTORIZA aquele atributo a entrar no lugar do da perícia — "pode usar Destreza em
		// vez de Força em Atletismo" (p28, p129).
		return "expertiseAttribute:" + t.Name + ":" + t.Attribute
	case "weaponAttribute":
		// A LICENÇA DO ATRIBUTO DA ARMA: o alcance (`Name`, corpo a corpo ou à distância), a conta (`Scope`, ataque
		// ou dano) e o atributo que pode entrar no lugar (o Armeiro e a Balística, p68-69).
		return "weaponAttribute:" + t.Name + ":" + t.Scope + ":" + t.Attribute
	case "scaleAttribute":
		// O VERBETE E O ATRIBUTO NA CHAVE, e a licença não soma nada: ela autoriza a escala daquele
		// verbete — o "soma a Destreza" do Estilo de Disparo — a usar outro atributo (a Balística, p69).
		return "scaleAttribute:" + t.Name + ":" + t.Attribute
	case "alchemicalSaveAttribute":
		// O ATRIBUTO NA CHAVE, e a licença não soma nada: como o `expertiseAttribute`, ela AUTORIZA
		// aquele atributo a entrar no lugar do que o item pede na CD — "pode usar sua Inteligência em
		// vez de Destreza" (o Granadeiro, p69).
		return "alchemicalSaveAttribute:" + t.Attribute
	case "mundaneHealing":
		// A CURA MUNDANA (o bálsamo, p160), e não toda cura: a mágica é a da Cura Gentil (p133).
		return "mundaneHealing"
	case "poisonDc":
		// A CD DO VENENO que o personagem aplica (o Veneno Potente, p74), e não a CD das magias nem a
		// das habilidades: são pilhas diferentes no livro.
		return "poisonDc"
	case "alchemicalDamage":
		// O DANO DO PREPARADO ALQUÍMICO (p160), e não o de toda dose: a água benta fere e é
		// equipamento de aventura (p155), fora do que o Alquimista de Batalha soma (p68).
		return "alchemicalDamage"
	}
	return ""
}
