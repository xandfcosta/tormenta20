package engine

// O MODIFICADOR QUE MULTIPLICA (ALE-390).
//
// Quase toda regra do livro SOMA, e é por isso que o motor nasceu só com soma.
// Algumas MULTIPLICAM, e elas não cabiam em lugar nenhum: o Lento reduz o
// deslocamento à metade (p395), o Imóvel o reduz a 0m (p394), e a Alta Arcana
// corta o custo em PM pela metade.
//
// # Ele age DEPOIS da soma, e quem diz isso é o livro
//
// A Alta Arcana escreve por extenso: "reduzido à metade (APÓS aplicar
// aprimoramentos e outros modificadores)". Então o fator não é mais uma parcela
// da pilha — ele opera sobre o total que a pilha produziu, e sobre a BASE
// junto: o Lento não corta o bônus de item do personagem, corta o quanto ele
// anda.
//
// # Dois fatores no mesmo alvo: vale o MAIS SEVERO, e eles não compõem
//
// É a regra que o livro já dá para condição — "aplique apenas o mais severo"
// (p394) —, e os dois fatores que existem hoje vêm de condição: um personagem
// Lento e Imóvel não anda 2,25m, anda 0m. Compor daria um número que o livro
// não imprime em lugar nenhum.
//
// Se um dia chegar um fator que NÃO seja de condição, esta é a linha a revisar.

// Ratio é um fator racional, em inteiros: {1,2} é metade, {2,1} é dobro, {0,1}
// é zero.
//
// Racional e não float porque o motor é modelado em INTEIROS, e porque a
// divisão inteira é exatamente o arredondamento que o livro pede — "arredonde
// para baixo para o primeiro incremento de 1,5m" (p395) é o que sai sozinho ao
// dividir um total em QUADRADOS por dois.
//
// @example engine.Ratio{Num: 1, Den: 2} // metade
type Ratio struct {
	Num int `json:"num"`
	Den int `json:"den"`
}

// isSet diz se o fator foi declarado. O zero (`{0,0}`) é "não há fator", e não
// "multiplique por zero" — quem zera escreve `{0,1}`.
func (r Ratio) isSet() bool { return r.Den != 0 }

// Applied multiplica um total, truncando para baixo.
//
// TRUNCA para o zero e não para baixo aritmético: em deslocamento não há
// negativo (quem não anda, não anda para trás), e é o mesmo que a
// `SquaresForDisplacement` já faz.
//
// @example engine.Ratio{Num: 1, Den: 2}.Applied(5) // 2
func (r Ratio) Applied(total int) int {
	if !r.isSet() {
		return total
	}
	return total * r.Num / r.Den
}

// severest devolve o fator que corta MAIS. Sem fator declarado, o outro vence;
// sem nenhum, o vazio.
//
// Compara por produto cruzado para não passar por ponto flutuante: `a` é mais
// severo que `b` quando `a.Num*b.Den < b.Num*a.Den`.
func severest(a, b Ratio) Ratio {
	switch {
	case !a.isSet():
		return b
	case !b.isSet():
		return a
	case a.Num*b.Den < b.Num*a.Den:
		return a
	default:
		return b
	}
}
