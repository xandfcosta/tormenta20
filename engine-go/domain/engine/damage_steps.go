package engine

import "strings"

// O PASSO DE DANO, e a Tabela 3-2 do livro (p143).
//
// *"Alguns efeitos podem aumentar ou diminuir o dano da arma em um ou mais
// «passos» […] Sempre que precisar aumentar ou diminuir o dano de uma arma em
// um ou mais passos, consulte a Tabela 3-2: Dano de Armas"* (p143).
//
// # A tabela é de LINHAS, e a coluna Normal é a porta de entrada
//
// Ela tem oito linhas e seis colunas (−2 a +3), e NÃO é uma escada só: a
// notação de entrada escolhe a LINHA pela célula da coluna Normal, e o degrau
// é a coluna. Três notações aparecem em mais de um lugar — o `2d6` é apelido do
// `1d12` na linha dele e é a coluna −1 da linha do `2d8` —, e é por isso que a
// entrada é SÓ pela coluna Normal: as outras cinco colunas são saídas.
//
// # O degrau se lê DE UMA VEZ, porque a página não é associativa
//
// `2d8 +1` é `3d8` e `2d10 −1` é `2d8`. Somar dois passos de um em um sairia do
// 2d8 para o 2d10 e daria `3d10`, que não é o que a linha do 2d8 imprime. Um
// modificador de +2 entra como +2.
//
// # Fora das seis colunas, a linha SATURA
//
// A página não imprime sétima coluna, e compor para alcançá-la teria de
// atravessar células que ela não relaciona — o parágrafo acima mostra que isso
// inventa número. O `1` da ponta de baixo e o `4d12 (máximo)` da de cima são os
// limites que o livro escreve.
var damageStepRows = [][6]string{
	// −2       −1       Normal    +1       +2        +3
	{"1", "1d2", "1d3", "1d4", "1d6", "1d8"},
	{"1d2", "1d3", "1d4", "1d6", "1d8", "1d10"},
	{"1d3", "1d4", "1d6", "1d8", "1d10", "1d12"},
	{"1d4", "1d6", "1d8", "1d10", "1d12", "3d6"},
	{"1d6", "1d8", "1d10", "1d12", "3d6", "4d6"},
	{"1d8", "1d10", "1d12", "3d6", "4d6", "4d8"},
	{"1d10", "2d6", "2d8", "3d8", "4d8", "4d10"},
	{"2d6", "2d8", "2d10", "3d10", "4d10", "4d12"},
}

// damageStepAliases são as notações que o livro imprime JUNTO de outra na
// mesma célula da coluna Normal: "1d8 ou 2d4" e "1d12, 2d6 ou 3d4" (p143). Uma
// arma de 2d4 anda pela linha do 1d8.
var damageStepAliases = map[string]string{"2d4": "1d8", "2d6": "1d12", "3d4": "1d12"}

// aColunaNormal é o índice da coluna Normal nas linhas acima.
const aColunaNormal = 2

// DamageStep devolve o dano da arma andado `steps` passos na Tabela 3-2 (p143).
//
// Notação que a tabela não imprime volta INTACTA — a rede causa dano "0", e
// inventar um degrau para ela seria inventar regra.
//
// @example DamageStep("1d8", 1) // "1d10"
func DamageStep(notation string, steps int) string {
	if steps == 0 || notation == "" {
		return notation
	}
	// ARMA DUPLA: "cada «ponta» conta como uma arma separada para efeitos de
	// melhorias e encantos" (p143), então cada lado anda pela linha dele.
	if strings.Contains(notation, "/") {
		ends := strings.Split(notation, "/")
		for i, end := range ends {
			ends[i] = DamageStep(end, steps)
		}
		return strings.Join(ends, "/")
	}
	entry := notation
	if alias, found := damageStepAliases[entry]; found {
		entry = alias
	}
	for _, row := range damageStepRows {
		if row[aColunaNormal] != entry {
			continue
		}
		column := aColunaNormal + steps
		if column < 0 {
			column = 0
		}
		if column > len(row)-1 {
			column = len(row) - 1
		}
		return row[column]
	}
	return notation
}
