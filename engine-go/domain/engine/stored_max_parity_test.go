package engine

import (
	"path/filepath"
	"testing"
)

// O MÁXIMO GRAVADO TEM DE SER O MÁXIMO DERIVADO.
//
// # O que este guarda existe para acusar
//
// `hpMax` e `mpMax` são COLUNAS, recomputadas só em gesto de escrita — nascer,
// passo de atributo, degrau de nível. Um catálogo que mude o poço de uma classe,
// ou uma raça que passe a somar um atributo, **não alcança ninguém** até o
// próximo desses gestos. O número gravado vira o retrato de um mundo que mudou.
//
// A decisão de produto é que ele acompanhe: se a regra do mundo mudou, o
// personagem muda junto (ALE-355). Enquanto o máximo for coluna, este guarda é o
// que mede a distância entre as duas coisas — e quando ele passar a ser
// derivado, o guarda deixa de ter o que comparar e sai junto com a coluna.
//
// # Por que nenhuma suíte via isso
//
// O oráculo de cada personagem carrega as DUAS respostas — o `char.hpMax`
// gravado e o `vitals.pvMax` derivado — e há uma suíte de paridade para cada
// uma. As duas passam. **Nenhuma compara as duas chaves do mesmo arquivo**, e é
// exatamente aí que a divergência morava: o `bardo-versatil-nv7` está com 31 PM
// gravados contra 33 derivados desde que a Carisma total dele subiu de 3 para 5,
// e o poder "Magias (1º círculo)" concede `maxPm += 1 × Carisma` (p43).
//
// # O denominador
//
// Ele não conta reprovados, conta COMPARAÇÕES: um extrator que parasse de achar
// o `char` daria zero divergências e passaria dizendo que está tudo certo.
func TestTheStoredMaxMatchesTheDerivedMax(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	catalogs := primeFromDump(t, dir)
	comparados := 0

	for _, slug := range parityOracleSlugs(t, dir) {
		t.Run(slug, func(t *testing.T) {
			var oracle struct {
				Char struct {
					Character
					HpMax int `json:"hpMax"`
					MpMax int `json:"mpMax"`
				} `json:"char"`
			}
			readJSON(t, filepath.Join(dir, slug), &oracle)

			derivado := catalogs.VitalsForCharacter(oracle.Char.Character)
			comparados++
			if oracle.Char.HpMax != derivado.PvMax {
				t.Errorf("PV máximo gravado %d, derivado %d (diferença de %d).\n"+
					"O número na coluna é o retrato de um catálogo anterior — ver o cabeçalho.",
					oracle.Char.HpMax, derivado.PvMax, oracle.Char.HpMax-derivado.PvMax)
			}
			if oracle.Char.MpMax != derivado.PmMax {
				t.Errorf("PM máximo gravado %d, derivado %d (diferença de %d).\n"+
					"O número na coluna é o retrato de um catálogo anterior — ver o cabeçalho.",
					oracle.Char.MpMax, derivado.PmMax, oracle.Char.MpMax-derivado.PmMax)
			}
		})
	}

	if comparados != parityOracleCount {
		t.Fatalf("o guarda comparou %d personagens de %d — o extrator do `char` parou de casar,\n"+
			"e zero divergências não quer dizer nada quando nada foi lido.", comparados, parityOracleCount)
	}
}
