package engine

// moreTargetKeys são as chaves dos alvos que nasceram depois de o `targetKey` passar do teto de
// linhas: o arquivo dele está na linha de base, e a dívida velha só encolhe. Alvo que ninguém
// conhece continua sendo "", que não alcança pilha nenhuma.
func moreTargetKeys(t ModifierTarget) string {
	switch t.K {
	case "expertiseAttribute":
		// A PERÍCIA E O ATRIBUTO NO NOME, e a licença não soma nada: como o `defenseAttribute`,
		// ela AUTORIZA aquele atributo a entrar no lugar do da perícia — "pode usar Destreza em
		// vez de Força em Atletismo" (p28, p129).
		return "expertiseAttribute:" + t.Name + ":" + t.Attribute
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
