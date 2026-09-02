package api

import (
	"fmt"

	"t20engine/tabuleiro"
)

// COMO UMA CASA DE TERRENO SE PARECE (ALE-203, escolha do dono).
//
// O desenho anterior eram quatro LAVAGENS de baixo contraste — hachura dourada a
// 14%, moldura azul, névoa, chanfro — e ele falhava em duas coisas ao mesmo
// tempo:
//
//  1. **Não se lia de longe.** Uma mesa olha o tabuleiro de um metro, às vezes
//     numa TV. 14% de opacidade sobre pedra escura é um sussurro.
//  2. **Não EMPILHAVA.** Folhagens são difícil E camuflagem (p267), e duas
//     lavagens somadas viram uma terceira coisa que não é nenhuma das duas — a
//     casa fica com "algo", e o mestre não sabe com o quê.
//
// O que entra: um ÍCONE num canto próprio de cada espécie, sobre uma tinta mais
// forte. Quatro espécies, quatro cantos — então duas na mesma casa continuam
// sendo duas coisas distintas, cada uma no lugar dela, em vez de uma mistura.
//
// # Os ícones dizem a REGRA, não a aparência
//
// `Shield` para cobertura porque ela é +5 na Defesa; `EyeOff` para camuflagem
// porque ela é 20% de chance de falha. O mestre não precisa lembrar que "névoa =
// camuflagem": ele vê o que a casa FAZ. É a mesma escolha que fez o `Ruler` e o
// `Radar` nomearem a régua e o gabarito.
//
// # O ícone SOME no zoom pequeno
//
// Abaixo de 32px de lado ele vira um borrão de seis pixels, e um borrão não
// informa nada — só suja. Quem o esconde é uma `@container` no CSS, medindo a
// casa: nesse regime a TINTA sozinha responde "tem alguma coisa aqui", que é o
// que cabe em 20px.

// desenhoDaEspecie é como uma espécie de terreno aparece na casa e no trilho.
type desenhoDaEspecie struct {
	// Icone é o nome do lucide, o mesmo que a casa e o botão do pincel usam. UM
	// desenho para os dois lugares: o mestre reconhece o pincel pelo que ele
	// PINTA, e não por uma legenda que ele teria de decorar.
	Icone string
	// Canto é o sufixo da classe que põe o ícone no lugar dele
	// (`terreno-canto-<Canto>`). Quatro espécies, quatro cantos.
	Canto string
}

// oDesenhoDasEspecies é a tabela, e ela é conferida contra a lista do domínio
// pelo `TestEveryKindHasADrawing`.
//
// Mapa e não campo no `tabuleiro.PincelDeTerreno`, porque nome de ícone do lucide
// é APARÊNCIA e o domínio não tem por que conhecê-lo. O preço dessa separação é a
// espécie nova poder nascer sem desenho — e é exatamente por isso que o guarda
// existe e que o `oDesenhoDe` recusa em vez de devolver um branco.
var oDesenhoDasEspecies = map[tabuleiro.TerrainKind]desenhoDaEspecie{
	tabuleiro.TerrenoDificil:    {Icone: "Waves", Canto: "noroeste"},
	tabuleiro.TerrenoCobertura:  {Icone: "Shield", Canto: "nordeste"},
	tabuleiro.TerrenoCamuflagem: {Icone: "EyeOff", Canto: "sudeste"},
	tabuleiro.TerrenoElevado:    {Icone: "Mountain", Canto: "sudoeste"},
}

// oDesenhoDe devolve o desenho da espécie, e ENTRA EM PÂNICO se não houver.
//
// Pânico e não um branco silencioso: uma espécie sem desenho pinta uma casa que
// não se distingue de nenhuma outra, e isso é indistinguível de "o pincel não
// funcionou". O `TestEveryKindHasADrawing` faz o pânico acontecer na suíte e não
// na mesa de alguém.
func oDesenhoDe(especie tabuleiro.TerrainKind) desenhoDaEspecie {
	d, tem := oDesenhoDasEspecies[especie]
	if !tem {
		panic(fmt.Sprintf("a espécie de terreno %q não tem desenho: acrescente-a em oDesenhoDasEspecies", especie))
	}
	return d
}

// aClasseDaCasa é o que a casa pintada veste: a espécie (que traz a tinta) e o
// canto do ícone.
func aClasseDaCasa(especie string) string {
	d := oDesenhoDe(tabuleiro.TerrainKind(especie))
	return "tabuleiro-terreno tabuleiro-" + especie + " terreno-canto-" + d.Canto
}
