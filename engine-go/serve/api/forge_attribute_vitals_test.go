package api

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// fereOHeroi grava o dano DIRETO no banco: é o arranjo do caso, não o código
// sob teste. O caminho de ferir de verdade é o da Mesa, e trazê-lo para cá
// poria dois handlers na frente da pergunta.
func fereOHeroi(t *testing.T, f sceneFixture, id, pvAtual, pmAtual int64) {
	t.Helper()
	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("herói %d: %v", id, err)
	}
	if err := f.s.queries.SetCharacterVitals(context.Background(), sqlcgen.SetCharacterVitalsParams{
		HpMax: row.Hpmax, HpCurrent: pvAtual, MpMax: row.Mpmax, MpCurrent: pmAtual,
		UpdatedAt: dbvalue.NowISO(), ID: id,
	}); err != nil {
		t.Fatalf("ferir o herói: %v", err)
	}
}

func osVitaisDe(t *testing.T, f sceneFixture, id int64) (pv, pvMax, pm, pmMax int64) {
	t.Helper()
	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("herói %d: %v", id, err)
	}
	return row.Hpcurrent, row.Hpmax, row.Mpcurrent, row.Mpmax
}

// umHeroiForjado devolve o id de um herói recém-nascido e o endereço dos
// atributos dele.
func umHeroiForjado(t *testing.T, f sceneFixture) (int64, string) {
	t.Helper()
	rec := postaAForja(t, f, f.jogador, "/personagens/nova", aFolhaPreenchida())
	id := oIDDoDestino(t, rec.Header().Get("Location"))
	return id, "/personagens/" + strconv.FormatInt(id, 10) + "/atributos"
}

// A cena de atributos chamava `fillPools` no fim de TODO passo bem-sucedido, e
// `fillPools` grava `HpCurrent = HpMax`. O guarda da cena confere id, existência
// e posse — e nada mais: não há checagem nenhuma de que o herói ainda está sendo
// forjado, e não há como haver, porque a tabela `characters` não guarda esse
// estado.
//
// O caminho do abuso são DOIS CLIQUES que não mudam nada na ficha: o `−` num
// atributo é sempre aceito dentro da faixa (gasta MENOS pontos), o `+` devolve
// o ponto, o espalhamento volta ao que era — e os dois passos enchiam os poços.
// Uma Lenda de nível 20 com 3 de 180 PV no meio de um combate saía com 180.
//
// Não é escalada de privilégio, porque exige ser o dono. Mas fura o MESTRE, e o
// sistema inteiro de vitais existe para o dano ser dele.
func TestTheAttributeStepDoesNotHealTheHero(t *testing.T) {
	f := newSceneFixture(t)
	id, atributos := umHeroiForjado(t, f)
	fereOHeroi(t, f, id, 3, 0)

	// O GESTO DO ABUSO, inteiro: desce a Constituição e devolve o ponto.
	for _, passo := range []string{"/constitution/-1", "/constitution/1"} {
		if code := postaAForja(t, f, f.jogador, atributos+passo, nil).Code; code != http.StatusOK {
			t.Fatalf("o passo %q respondeu %d", passo, code)
		}
	}

	pv, pvMax, _, _ := osVitaisDe(t, f, id)
	if pv != 3 {
		t.Errorf("o herói ferido saiu com %d de %d PV: dois cliques que não mudam nada na ficha "+
			"curaram %d pontos. O dano é do mestre.", pv, pvMax, pv-3)
	}
	// O CONTROLE: o máximo voltou ao que era, então o espalhamento de fato
	// fechou o ciclo. Sem ele, "o atual não subiu" poderia ser o passo não ter
	// acontecido. São 19 e não 20 porque o elfo tem Constituição −1 e a base
	// nasce em zero — o número está escrito à mão, e conferi-lo custou um
	// vermelho a mais.
	if pvMax != 19 {
		t.Errorf("o PV máximo ficou em %d e o ciclo devia fechar em 19 — o passo não foi de ida e volta", pvMax)
	}
}

// A outra metade: o atual ACOMPANHA o delta do máximo, nos dois sentidos.
//
// É a mesma regra que a mudança de NÍVEL já usa (`levelVitalsNext`), e ela tem
// de ser a mesma: com "prende na faixa" só para baixo, o ciclo `−` e `+` devolve
// dois pontos de PV por volta — o mesmo defeito, mais devagar.
func TestTheAttributeStepWalksTheWoundedPoolWithTheMax(t *testing.T) {
	f := newSceneFixture(t)
	id, atributos := umHeroiForjado(t, f)

	// O elfo tem Constituição −1: com a base em +1 o total é 0, e o guerreiro de
	// nível 1 fica com 20 de PV máximo. Os números são escritos à mão de
	// propósito — derivá-los do motor os faria andar junto com o defeito.
	if code := postaAForja(t, f, f.jogador, atributos+"/constitution/1", nil).Code; code != http.StatusOK {
		t.Fatalf("subir a Constituição respondeu %d", code)
	}
	if pv, pvMax, _, _ := osVitaisDe(t, f, id); pv != 20 || pvMax != 20 {
		t.Fatalf("o herói recém-nascido está em %d/%d e devia estar CHEIO em 20/20", pv, pvMax)
	}

	fereOHeroi(t, f, id, 3, 0)

	// Base +2 num elfo dá Con +1: +1 de PV máximo por nível, e o nível é 1.
	if code := postaAForja(t, f, f.jogador, atributos+"/constitution/1", nil).Code; code != http.StatusOK {
		t.Fatalf("subir a Constituição de novo respondeu %d", code)
	}
	if pv, pvMax, _, _ := osVitaisDe(t, f, id); pv != 4 || pvMax != 21 {
		t.Errorf("o ferido ficou em %d/%d, e acompanhando o delta seria 4/21", pv, pvMax)
	}

	// E DE VOLTA: o mesmo delta para baixo desfaz exatamente o passo.
	if code := postaAForja(t, f, f.jogador, atributos+"/constitution/-1", nil).Code; code != http.StatusOK {
		t.Fatalf("descer a Constituição respondeu %d", code)
	}
	if pv, pvMax, _, _ := osVitaisDe(t, f, id); pv != 3 || pvMax != 20 {
		t.Errorf("a volta deixou %d/%d, e o ciclo tem de fechar exato em 3/20 — "+
			"é isso que impede o gesto de virar uma bomba de PV", pv, pvMax)
	}
}

// O CONTROLE do conserto.
//
// A razão escrita no chamador — "o herói ainda está sendo forjado, então ele
// fica com os poços CHEIOS" — continua valendo, e é ela que impede alguém de
// abrir a ficha com 18 de 20 PV sem nunca ter apanhado. O que deixou de
// acontecer é a CURA de quem apanhou.
func TestTheNewbornStillLeavesTheForgeWithFullPools(t *testing.T) {
	f := newSceneFixture(t)
	id, atributos := umHeroiForjado(t, f)

	if pv, pvMax, pm, pmMax := osVitaisDe(t, f, id); pv != pvMax || pm != pmMax {
		t.Fatalf("o herói nasceu em %d/%d PV e %d/%d PM, e devia nascer CHEIO", pv, pvMax, pm, pmMax)
	}
	// Cheio continua cheio quando o máximo SOBE: dois pontos em Constituição
	// levam o elfo de Con 0 a Con +1, e o poço acompanha.
	for i := 0; i < 2; i++ {
		if code := postaAForja(t, f, f.jogador, atributos+"/constitution/1", nil).Code; code != http.StatusOK {
			t.Fatalf("subir a Constituição respondeu %d", code)
		}
	}
	if pv, pvMax, _, _ := osVitaisDe(t, f, id); pv != 21 || pvMax != 21 {
		t.Errorf("o recém-nascido ficou em %d/%d, e cheio segue cheio em 21/21", pv, pvMax)
	}
}
