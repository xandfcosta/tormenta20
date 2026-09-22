package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
	"testing"
)

func effectScreen(t *testing.T, f sceneFixture, id int64) string {
	t.Helper()
	return f.requests(t, f.player, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=conditionals", id), "").Body.String()
}

func effect(t *testing.T, f sceneFixture, id int64, path string) *responseRecorderLike {
	t.Helper()
	target := fmt.Sprintf("/personagens/%d/efeitos/%s?tab=conditionals", id, path)
	rec := f.requests(t, f.player, http.MethodPost, target, "")
	return &responseRecorderLike{Code: rec.Code, Body: rec.Body.String()}
}

func conditions(t *testing.T, f sceneFixture, id int64) []string {
	t.Helper()
	row, err := f.s.sceneCore().Queries().GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler o personagem: %v", err)
	}
	return sheet.UnmarshalStrings(row.Activeconditions)
}

// A CONDIÇÃO ENTRA, SAI, E MOVE OS NÚMEROS.
//
// Mover os números é o caso inteiro: uma condição que fosse só crachá foi o
// defeito da ALE-28. O Caído (p394) NÃO mexe na Defesa geral — ele separa −5
// contra corpo a corpo e +5 contra à distância —, então medir o total mediria
// justamente onde a regra não aparece, e daria "22 → 22" com cara de "não
// funcionou". O guarda olha as linhas DIRECIONAIS.
func TestAConditionEntersLeavesAndMovesTheNumbers(t *testing.T) {
	f, id := fighterFixture(t)

	aba := combatScreen(t, f, id)
	if strings.Contains(aba, "Contra corpo a corpo") {
		t.Fatal("a Defesa já vinha com linhas direcionais: o caso não mede a chegada do Caído")
	}

	if rec := effect(t, f, id, "condicao/caido"); rec.Code != http.StatusOK {
		t.Fatalf("aplicar o Caído respondeu %d: %s", rec.Code, rec.Body)
	}
	if got := conditions(t, f, id); len(got) != 1 || got[0] != "caido" {
		t.Fatalf("as condições gravadas são %v, quer [caido]", got)
	}
	withProne := combatScreen(t, f, id)
	for _, want := range []string{"Contra corpo a corpo", "Contra ataques à distância"} {
		if !strings.Contains(withProne, want) {
			t.Errorf("com o Caído aplicado a Defesa não mostra %q: a condição virou crachá", want)
		}
	}

	// O MESMO comando desliga: o gesto manda a condição, não o estado desejado.
	if rec := effect(t, f, id, "condicao/caido"); rec.Code != http.StatusOK {
		t.Fatalf("remover o Caído respondeu %d", rec.Code)
	}
	if got := conditions(t, f, id); len(got) != 0 {
		t.Errorf("o segundo toque não removeu o Caído: %v", got)
	}
}

// UMA CONDIÇÃO INVENTADA É RECUSADA, e nada é gravado.
//
// A autoridade é o CATÁLOGO, e não uma lista escrita na tela: um blob com uma
// condição fantasma injetaria na ficha um efeito que o livro não tem.
func TestAnInventedConditionIsRefused(t *testing.T) {
	f, id := fighterFixture(t)
	if refusal := sceneRefusal(effect(t, f, id, "condicao/entediado").Body); refusal == "" {
		t.Error("uma condição que não existe foi aceita sem uma palavra na tela")
	}
	if got := conditions(t, f, id); len(got) != 0 {
		t.Errorf("a recusa gravou assim mesmo: %v", got)
	}
}

// O EFEITO APLICADO É DE QUEM O APLICOU, e a query não confere isso sozinha.
//
// O `DeleteEffectByID` apaga por id e mais nada. Sem a leitura de posse que o
// comando faz antes, um `@post` montado à mão encerraria o efeito de OUTRO
// personagem — e o 403 do `sheetCommand` não pega, porque a ficha do caminho é
// a do dono.
func TestAnEffectFromAnotherSheetCannotBeEnded(t *testing.T) {
	f, mine := fighterFixture(t)
	other := seedCharacterAtLevel(t, f.s, f.player, "Vizinho", "Guerreiro", 1, 0, 0)
	foreign, err := f.s.sceneCore().Queries().CreateActiveEffect(context.Background(), sqlcgen.CreateActiveEffectParams{
		Characterid: other, Catalogid: "armadura-arcana", Scope: "scene",
		Modifiers: "[]", Createdat: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear o efeito alheio: %v", err)
	}

	rec := effect(t, f, mine, fmt.Sprintf("encerra/%d", foreign.ID))
	if refusal := sceneRefusal(rec.Body); refusal == "" {
		t.Error("encerrei o efeito de outro personagem pela minha ficha")
	}
	if _, err := f.s.sceneCore().Queries().GetActiveEffectMeta(context.Background(), foreign.ID); err != nil {
		t.Error("o efeito alheio foi apagado assim mesmo")
	}
}

// O PAINEL CHEGA NA TELA.
func TestTheEffectsPanelDrawsTheFourBlocks(t *testing.T) {
	f, id := fighterFixture(t)
	screen := effectScreen(t, f, id)

	for _, want := range []string{"Condições (p394)", "Efeitos ativos", "Aplicar condição", "Aplicar magia"} {
		if !strings.Contains(screen, want) {
			t.Errorf("a tela não tem %q", want)
		}
	}
}

// AS DUAS POÇAS DE PV TEMPORÁRIO CONVIVEM, e a segunda não apaga a primeira.
//
// O livro: *"Pontos Temporários. Certos efeitos fornecem PV ou PM temporários.
// Eles são SOMADOS a seus pontos atuais, mesmo que ultrapassem o máximo"*
// (p106). Ele não tem vale-o-maior em lugar nenhum — marca à mão os poderes
// cujos pontos são "cumulativos" (p45, p54, p76) e nunca diz que duas fontes se
// excluem.
//
// O caso precisa de DUAS fontes e de uma MAIOR primeiro: com uma só, o gesto
// certo e o errado dão o mesmo resultado. O Campo de Força (30 de cena) entra
// antes da Alma de Bronze (nível + Força), que é a poça menor — e era ela que
// sumia, sem erro nenhum, porque o plano a declarava superada.
func TestATempHpPoolDoesNotWipeTheOneAlreadyThere(t *testing.T) {
	f, id := barbaro(t, 5)

	if rec := effect(t, f, id, "aplica/campo-de-forca"); rec.Code != http.StatusOK {
		t.Fatalf("aplicar o Campo de Força devolveu %d", rec.Code)
	}
	if refusal := powerCommand(t, f, id, "postura/furia/entra", `{"stance_degrees":0}`); refusal != "" {
		t.Fatalf("entrar na Fúria foi recusado: %q", refusal)
	}

	active := effects(t, f, id)
	for _, source := range []string{"campo-de-forca", "class.barbaro.alma-de-bronze"} {
		if !active[source] {
			t.Errorf("a poça de %q não está na ficha: %v", source, active)
		}
	}

	// E AS DUAS CHEGAM AO HTML DA CENA, cada uma com o próprio valor.
	//
	// "Chega ao HTML" e não "aparece na tela", e a diferença é medida: o cartão
	// de efeito ativo desenha só o NOME; o valor mora no diálogo de detalhe, que
	// nasce fechado dentro de um `fixed inset-0`. Uma mensagem dizendo "a aba não
	// mostra" afirmaria sobre PRESENÇA VISUAL o que este instrumento não olha
	// (ALE-347, conferido no navegador).
	screen := effectScreen(t, f, id)
	for _, howMuch := range []string{"+30", "+9"} {
		if !strings.Contains(screen, howMuch) {
			t.Errorf("o HTML da aba não traz %q de PV temporários", howMuch)
		}
	}
}

// UMA MAGIA SUSTENTADA POR VEZ (T20 p227).
//
// "Você pode manter diversas habilidades sustentadas, pagando o custo de cada
// uma, mas apenas uma magia sustentada por vez." O limite é sobre MAGIA, e a
// recusa nomeia a que está de pé — senão a resposta é "não" sem dizer o que
// desligar.
func TestOnlyOneSustainedSpellAtATime(t *testing.T) {
	f := newSceneFixture(t)
	id := f.charID

	if rec := effect(t, f, id, "aplica/velocidade"); rec.Code != http.StatusOK {
		t.Fatalf("conjurar a primeira sustentada deu %d", rec.Code)
	}
	// O CONTROLE: a primeira entrou mesmo, senão a segunda "recusada" seria uma
	// recusa sobre nada.
	if sustentadas(t, f, id) != 1 {
		t.Fatalf("o controle falhou: a primeira sustentada não entrou na ficha")
	}

	rec := effect(t, f, id, "aplica/oracao")

	// A FRASE INTEIRA, e não só o nome: "Velocidade" aparece na lista de magias
	// da própria tela, e casar só o nome passaria verde sem recusa nenhuma.
	if !strings.Contains(rec.Body, "Velocidade já está sustentada") {
		t.Errorf("a recusa tem de nomear a magia sustentada que está de pé, e veio %q",
			rec.Body[:min(240, len(rec.Body))])
	}
	if n := sustentadas(t, f, id); n != 1 {
		t.Errorf("a ficha ficou com %d magias sustentadas, e o livro permite UMA", n)
	}
}

// sustentadas conta as magias sustentadas de uma ficha.
func sustentadas(t *testing.T, f sceneFixture, id int64) int {
	t.Helper()
	rows, err := f.s.sceneCore().Queries().ListActiveEffectsByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("listar efeitos: %v", err)
	}
	n := 0
	for _, l := range rows {
		if l.Scope == "sustained" {
			n++
		}
	}
	return n
}

// A PROCEDÊNCIA DE UM BÔNUS SAI COM O NOME DO LIVRO, e não com o id.
//
// A decomposição da ficha diz de onde cada número vem, e para efeito de magia
// ela vinha escrita "armadura-arcana (cena)": o `engine.Catalogs` não carrega
// magias, então o resolvedor de nomes do motor nunca teve o que resolver e caía
// no id. Quem lê a ficha não conhece o id.
func TestAnAppliedSpellIsNamedByItsBookName(t *testing.T) {
	f := newSceneFixture(t)
	id := f.charID
	if rec := effect(t, f, id, "aplica/armadura-arcana"); rec.Code != http.StatusOK {
		t.Fatalf("aplicar a magia deu %d", rec.Code)
	}
	// A DECOMPOSIÇÃO mora na aba COMBATE: é lá que o bônus de Defesa diz de
	// onde veio, e foi lá que o id apareceu.
	screen := f.requests(t, f.player, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=combat", id), "").Body.String()

	// O CONTROLE primeiro: a procedência TEM de estar na tela, senão as duas
	// asserções abaixo medem a ausência do bloco inteiro e passam verde.
	if !strings.Contains(screen, "(cena)") {
		t.Fatal("a procedência do efeito não chegou à tela: o teste mediria o nada")
	}
	if strings.Contains(screen, "armadura-arcana (") {
		t.Error("a procedência saiu com o id da magia; quem lê a ficha não conhece o id")
	}
	if !strings.Contains(screen, "Armadura Arcana (") {
		t.Error("a procedência tem de trazer o nome do livro")
	}
}

// A DURAÇÃO DO EFEITO É LIDA EM PORTUGUÊS na ficha.
//
// A coluna `scope` grava em inglês porque é fronteira, e a tela traduz. A
// palavra da sustentada nasceu na ALE-365 e não tinha tradução: a aba mostrava
// "sustained", que é identificador vazando para quem lê.
func TestTheSheetReadsTheEffectDurationInPortuguese(t *testing.T) {
	f := newSceneFixture(t)
	id := f.charID
	if rec := effect(t, f, id, "aplica/velocidade"); rec.Code != http.StatusOK {
		t.Fatalf("conjurar a sustentada deu %d", rec.Code)
	}
	screen := effectScreen(t, f, id)

	if !strings.Contains(screen, "Velocidade") {
		t.Fatal("o controle falhou: o efeito não chegou à aba")
	}
	if strings.Contains(screen, ">sustained<") || strings.Contains(screen, " sustained") {
		t.Error("a duração saiu em inglês: `sustained` é a grafia da COLUNA, não da tela")
	}
	if !strings.Contains(screen, "enquanto for sustentada") {
		t.Error("a aba tem de dizer até quando o efeito vale, em português")
	}
}

// A DECOMPOSIÇÃO NÃO DIZ O MESMO NOME DUAS VEZES.
//
// A linha traz a procedência ("Armadura Arcana (cena)") e, embaixo, a nota do
// modificador. Enquanto a procedência saía com o id, as duas linhas diziam
// coisas diferentes e a nota era a única a dar o nome; com o id consertado ela
// virou eco, custando uma linha a 390px para não informar nada.
func TestTheBreakdownDoesNotEchoTheSourceName(t *testing.T) {
	f := newSceneFixture(t)
	id := f.charID
	if rec := effect(t, f, id, "aplica/armadura-arcana"); rec.Code != http.StatusOK {
		t.Fatalf("aplicar a magia deu %d", rec.Code)
	}
	screen := f.requests(t, f.player, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=combat", id), "").Body.String()

	if !strings.Contains(screen, "Armadura Arcana (cena)") {
		t.Fatal("o controle falhou: a procedência não chegou à decomposição")
	}
	if n := strings.Count(screen, "Armadura Arcana"); n != 1 {
		t.Errorf("o nome aparece %d vezes na decomposição, e uma basta", n)
	}
}
