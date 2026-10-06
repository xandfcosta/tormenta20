package table

// O ALVO DA VEZ.
//
// Um gesto de combate precisa de duas coisas: quem age e contra quem. Quem age
// é a VEZ, e o servidor já a resolve sozinho (p231). Contra quem só existia no
// menu da peça do tabuleiro — então o jogador sem mapa aberto não tinha gesto
// nenhum, que é a metade do A→B que faltava fechar.

// turnTarget é uma linha da fila oferecida como alvo, e ela tem NOME e nada
// mais.
//
// O TIPO É A TRAVA, e é por isso que ele existe em vez de a barra receber a
// `tableRow`: o jogador não vê estatística de alvo — num jogo de interpretação o
// personagem não sabe a Defesa do ogro, logo quem joga também não sabe (decisão
// do dono, ALE-423). Com a linha da fila à mão, imprimir o PV seria uma linha de
// template que ninguém barra em revisão; com este tipo, não há o que imprimir.
type turnTarget struct {
	EntryID string
	Label   string
}

// turnTargetsOf são as linhas da fila que NÃO são minhas.
//
// O ALIADO ENTRA. Quem decide se um gesto contra o companheiro acontece é o
// mestre, e a p234 não o proíbe — filtrar aqui seria a tela arbitrando. O que
// sai é o próprio personagem, e por uma razão de gesto e não de regra: o
// `Propose` já recusa atacar a si mesmo, e oferecer uma opção que existe para
// levar recusa é um erro desenhado.
func turnTargetsOf(queue []tableRow) []turnTarget {
	targets := make([]turnTarget, 0, len(queue))
	for _, row := range queue {
		if row.Mine {
			continue
		}
		targets = append(targets, turnTarget{EntryID: row.ID, Label: row.Label})
	}
	return targets
}
