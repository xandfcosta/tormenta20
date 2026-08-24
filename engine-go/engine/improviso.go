package engine

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// AS TABELAS DE IMPROVISO — Cap 6: ruínas (6-4, p272), eventos de perseguição
// (6-5, p274), recompensa e castigo, e o esqueleto de masmorra (6-2, p263).
//
// Elas moram no `engine` pela mesma razão da conta do encontro: são regra do
// livro. A diferença é que aqui o dado ROLA, e rolar é a única coisa desta
// família que não é função pura.
//
// ONDE O DADO ROLA foi decisão do dono (ALE-261). A SPA rola no navegador, e o
// comentário de lá defende isso — "o mestre pego sem resposta no meio da cena
// recebe uma sem ida ao servidor". No piloto ele rola no SERVIDOR, e o que
// muda o cálculo é que as tabelas JÁ estão no catálogo Go: rolar aqui mantém o
// mapeamento rolagem→linha num lugar só e não embarca tabela no navegador. Em
// produção é o mesmo binário que serve a página, então a "ida ao servidor" é a
// mesma ida que desenhou a tela.

// Rolagem é o resultado de um dado, e ele viaja junto com a linha porque o
// mestre quer VER o número: "saiu 4" é parte da resposta, não detalhe de
// implementação. Sem ele a tabela vira um oráculo que não se confere.
type Rolagem struct {
	Valor int
	Faces int
}

// RolaDado devolve 1..faces com aleatoriedade CRIPTOGRÁFICA.
//
// `crypto/rand` e não `math/rand`, e a razão não é segurança — é que o
// `math/rand` global do Go tem semente fixa por processo em versões antigas e
// comportamento surpreendente sob concorrência. Um servidor que serve várias
// mesas ao mesmo tempo não pode ter duas rolagens correlacionadas, e o custo
// aqui é irrelevante: são dezenas de rolagens por sessão, não milhões.
func RolaDado(faces int) (Rolagem, error) {
	if faces < 2 {
		return Rolagem{}, fmt.Errorf("dado precisa de pelo menos 2 faces, veio %d", faces)
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(faces)))
	if err != nil {
		return Rolagem{}, fmt.Errorf("rolar d%d: %w", faces, err)
	}
	return Rolagem{Valor: int(n.Int64()) + 1, Faces: faces}, nil
}

// ── as faixas ────────────────────────────────────────────────────────────────

// FaixaDeRolagem é uma linha que cobre um intervalo de resultados — "1-2" na
// tabela de ruínas.
type FaixaDeRolagem interface {
	Cobre(rolagem int) bool
}

// LinhaParaRolagem acha a linha que cobre a rolagem.
//
// Rolagem descoberta é ERRO e não silêncio, e essa é a decisão que importa: uma
// tabela com buraco devolveria a linha errada ou nenhuma, e o mestre leria o
// resultado de outra faixa como se fosse o dele. Melhor a tela dizer que a
// tabela está incompleta.
func LinhaParaRolagem[T FaixaDeRolagem](linhas []T, rolagem int, tabela string) (T, error) {
	var vazio T
	for _, l := range linhas {
		if l.Cobre(rolagem) {
			return l, nil
		}
	}
	return vazio, fmt.Errorf("%s: nenhuma linha cobre a rolagem %d", tabela, rolagem)
}

// ── a masmorra ───────────────────────────────────────────────────────────────

// SalasPorAmeaca é a razão do livro: "Calcule uma ameaça para cada três salas,
// com um misto de cenas de ação e exploração" (p263).
//
// O número vem do catálogo (`dungeon-design.json`) e não de uma constante aqui:
// ele é dado transcrito do livro, e dado transcrito mora no catálogo, que é
// onde a validação de schema o alcança.

// AmeacasPlanejadas arredonda PARA CIMA: sete salas com uma ameaça a cada três
// dão três, não duas. Duas deixariam a última salinha sem nada, e a regra do
// livro é uma cota mínima de tensão, não uma divisão exata.
func AmeacasPlanejadas(salas, salasPorAmeaca int) (int, error) {
	if salas <= 0 {
		return 0, fmt.Errorf("a masmorra precisa de pelo menos 1 sala, veio %d", salas)
	}
	if salasPorAmeaca <= 0 {
		return 0, fmt.Errorf("salas por ameaça precisa ser > 0, veio %d", salasPorAmeaca)
	}
	return (salas + salasPorAmeaca - 1) / salasPorAmeaca, nil
}
