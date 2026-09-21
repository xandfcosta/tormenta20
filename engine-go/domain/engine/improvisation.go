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
// O dado rola no SERVIDOR: as tabelas já estão no catálogo Go, então rolar aqui
// mantém o mapeamento rolagem→linha num lugar só e não embarca tabela no
// navegador.

// Roll é o resultado de um dado, e ele viaja junto com a linha porque o
// mestre quer VER o número: "saiu 4" é parte da resposta, não detalhe de
// implementação. Sem ele a tabela vira um oráculo que não se confere.
type Roll struct {
	Value int
	Faces int
}

// RollDie devolve 1..faces com aleatoriedade CRIPTOGRÁFICA.
//
// `crypto/rand` e não `math/rand`, e a razão não é segurança: o `math/rand`
// global tem comportamento surpreendente sob concorrência, e um servidor que
// atende várias mesas ao mesmo tempo não pode ter duas rolagens
// correlacionadas. O custo é irrelevante — são dezenas de rolagens por sessão,
// não milhões.
func RollDie(faces int) (Roll, error) {
	if faces < 2 {
		return Roll{}, fmt.Errorf("dado precisa de pelo menos 2 faces, veio %d", faces)
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(faces)))
	if err != nil {
		return Roll{}, fmt.Errorf("rolar d%d: %w", faces, err)
	}
	return Roll{Value: int(n.Int64()) + 1, Faces: faces}, nil
}

// ── as faixas ────────────────────────────────────────────────────────────────

// RollRange é uma linha que cobre um intervalo de resultados — "1-2" na
// tabela de ruínas.
type RollRange interface {
	Covers(scroll int) bool
}

// RowForRoll acha a linha que cobre a rolagem.
//
// Rolagem descoberta é ERRO e não silêncio, e essa é a decisão que importa: uma
// tabela com buraco devolveria a linha errada ou nenhuma, e o mestre leria o
// resultado de outra faixa como se fosse o dele. Melhor a tela dizer que a
// tabela está incompleta.
func RowForRoll[T RollRange](rows []T, scroll int, table string) (T, error) {
	var empty T
	for _, l := range rows {
		if l.Covers(scroll) {
			return l, nil
		}
	}
	return empty, fmt.Errorf("%s: nenhuma linha cobre a rolagem %d", table, scroll)
}

// ── a masmorra ───────────────────────────────────────────────────────────────

// SalasPorAmeaca é a razão do livro: "Calcule uma ameaça para cada três salas,
// com um misto de cenas de ação e exploração" (p263).
//
// O número vem do catálogo (`dungeon-design.json`) e não de uma constante aqui:
// ele é dado transcrito do livro, e dado transcrito mora no catálogo, que é
// onde a validação de schema o alcança.

// PlannedThreats arredonda PARA CIMA: sete salas com uma ameaça a cada três
// dão três, não duas. Duas deixariam a última salinha sem nada, e a regra do
// livro é uma cota mínima de tensão, não uma divisão exata.
func PlannedThreats(rooms, roomsByThreat int) (int, error) {
	if rooms <= 0 {
		return 0, fmt.Errorf("a masmorra precisa de pelo menos 1 sala, veio %d", rooms)
	}
	if roomsByThreat <= 0 {
		return 0, fmt.Errorf("salas por ameaça precisa ser > 0, veio %d", roomsByThreat)
	}
	return (rooms + roomsByThreat - 1) / roomsByThreat, nil
}
