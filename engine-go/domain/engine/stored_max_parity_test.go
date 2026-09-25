package engine

import (
	"path/filepath"
	"testing"
)

// O MÁXIMO GRAVADO TEM DE SER O MÁXIMO DERIVADO.
//
// # O que este guarda existe para acusar
//
// O `hpMax` e o `mpMax` do bloco `char` de cada oráculo são número ESCRITO À
// MÃO: o `char` sai verbatim do `_fixtures.json`, que é a entrada carregada da
// era do TypeScript, e o `genoracle` recalcula tudo MENOS ele. Um catálogo que
// mude o poço de uma classe, ou uma raça que passe a somar um atributo, não
// alcança esse número.
//
// # O terreno dele MUDOU DE CASA, e ele ficou
//
// Ele nasceu medindo as colunas `hpMax`/`mpMax` de `characters` — que saíram na
// migração 00015, porque a decisão do dono é que o personagem acompanhe a regra
// do mundo (ALE-355). A invariante não morreu com elas: mudou de dono. Hoje o
// número gravado à mão é o da FIXTURE, e comparar os dois continua sendo o
// controle de duas naturezas que pegou o bardo.
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
	compared := 0

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

			derived := BookRuleset(catalogs).VitalsForCharacter(oracle.Char.Character)
			compared++
			if oracle.Char.HpMax != derived.PvMax {
				t.Errorf("PV máximo gravado %d, derivado %d (diferença de %d).\n"+
					"O número na coluna é o retrato de um catálogo anterior — ver o cabeçalho.",
					oracle.Char.HpMax, derived.PvMax, oracle.Char.HpMax-derived.PvMax)
			}
			if oracle.Char.MpMax != derived.PmMax {
				t.Errorf("PM máximo gravado %d, derivado %d (diferença de %d).\n"+
					"O número na coluna é o retrato de um catálogo anterior — ver o cabeçalho.",
					oracle.Char.MpMax, derived.PmMax, oracle.Char.MpMax-derived.PmMax)
			}
		})
	}

	if compared != parityOracleCount {
		t.Fatalf("o guarda comparou %d personagens de %d — o extrator do `char` parou de casar,\n"+
			"e zero divergências não quer dizer nada quando nada foi lido.", compared, parityOracleCount)
	}
}
