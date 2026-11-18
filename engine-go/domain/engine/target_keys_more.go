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
	}
	return ""
}
