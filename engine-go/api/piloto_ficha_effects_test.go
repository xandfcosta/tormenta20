package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"t20engine/db/sqlcgen"
	"t20engine/engine"
	"t20engine/plataforma"
)

// Os guardas do painel de EFEITOS (ALE-272, fatia 5).
//
// O que eles prendem é o que separa este painel de uma lista bonita: a condição
// MOVE os números, a mesa é avisada, e a fronteira de cada gesto é do servidor.

func aTelaDosEfeitos(t *testing.T, f pilotoFixture, id int64) string {
	t.Helper()
	return f.pede(t, f.jogador, http.MethodGet,
		fmt.Sprintf("/piloto/personagens/%d?tab=conditionals", id), "").Body.String()
}

func oEfeito(t *testing.T, f pilotoFixture, id int64, caminho string) *responseRecorderLike {
	t.Helper()
	alvo := fmt.Sprintf("/piloto/personagens/%d/efeitos/%s?tab=conditionals", id, caminho)
	rec := f.pede(t, f.jogador, http.MethodPost, alvo, "")
	return &responseRecorderLike{Code: rec.Code, Body: rec.Body.String()}
}

type responseRecorderLike struct {
	Code int
	Body string
}

func asCondicoesDe(t *testing.T, f pilotoFixture, id int64) []string {
	t.Helper()
	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler o personagem: %v", err)
	}
	return parseConditionBlob(row.Activeconditions)
}

// A CONDIÇÃO ENTRA, SAI, E MOVE OS NÚMEROS.
//
// Mover os números é o caso inteiro: uma condição que fosse só crachá foi o
// defeito da ALE-28. O Caído (p394) NÃO mexe na Defesa geral — ele separa −5
// contra corpo a corpo e +5 contra à distância —, então medir o total mediria
// justamente onde a regra não aparece, e daria "22 → 22" com cara de "não
// funcionou". O guarda olha as linhas DIRECIONAIS.
func TestACondicaoEntraSaiEMoveOsNumeros(t *testing.T) {
	f, id := oCombatente(t)

	aba := aTelaDoCombate(t, f, id)
	if strings.Contains(aba, "Contra corpo a corpo") {
		t.Fatal("a Defesa já vinha com linhas direcionais: o caso não mede a chegada do Caído")
	}

	if rec := oEfeito(t, f, id, "condicao/caido"); rec.Code != http.StatusOK {
		t.Fatalf("aplicar o Caído respondeu %d: %s", rec.Code, rec.Body)
	}
	if got := asCondicoesDe(t, f, id); len(got) != 1 || got[0] != "caido" {
		t.Fatalf("as condições gravadas são %v, quer [caido]", got)
	}
	comCaido := aTelaDoCombate(t, f, id)
	for _, esperado := range []string{"Contra corpo a corpo", "Contra ataques à distância"} {
		if !strings.Contains(comCaido, esperado) {
			t.Errorf("com o Caído aplicado a Defesa não mostra %q: a condição virou crachá", esperado)
		}
	}

	// O MESMO comando desliga: o gesto manda a condição, não o estado desejado.
	if rec := oEfeito(t, f, id, "condicao/caido"); rec.Code != http.StatusOK {
		t.Fatalf("remover o Caído respondeu %d", rec.Code)
	}
	if got := asCondicoesDe(t, f, id); len(got) != 0 {
		t.Errorf("o segundo toque não removeu o Caído: %v", got)
	}
}

// UMA CONDIÇÃO INVENTADA É RECUSADA, e nada é gravado.
//
// A autoridade é o CATÁLOGO, e não uma lista escrita na tela: um blob com uma
// condição fantasma injetaria na ficha um efeito que o livro não tem.
func TestUmaCondicaoInventadaERecusada(t *testing.T) {
	f, id := oCombatente(t)
	if recusa := aRecusaDaCena(oEfeito(t, f, id, "condicao/entediado").Body); recusa == "" {
		t.Error("uma condição que não existe foi aceita sem uma palavra na tela")
	}
	if got := asCondicoesDe(t, f, id); len(got) != 0 {
		t.Errorf("a recusa gravou assim mesmo: %v", got)
	}
}

// A CONDIÇÃO AVISA A MESA AO VIVO, e isto é a ALE-245 sobrevivendo ao porte.
//
// O motor deriva Defesa e perícias da condição, então uma aplicada sem aviso faz
// o jogador e o mestre verem números DIFERENTES do mesmo personagem, sem nada na
// tela dizendo que discordam. O `handleUpdateConditions` da API JSON é o único
// outro lugar que avisa — se a ficha em Datastar não avisasse, o porte teria
// REGREDIDO o conserto sem que nenhum teste percebesse.
func TestACondicaoAvisaAMesaAoVivo(t *testing.T) {
	// O guarda lê a FONTE do handler, e não um evento no fio: montar uma sessão
	// viva com este personagem na fila para ouvir um SSE é caro, e o que se quer
	// prender é que a CHAMADA não some — que é como a ALE-245 foi perdida da
	// primeira vez (um gancho que ninguém preenchia).
	fonte := lerFonte(t, "piloto_ficha_comandos_efeitos.go")
	corpo := recorteDaFuncao(t, fonte, "func toggleBookCondition")
	if !strings.Contains(corpo, "s.characterChanged(row.ID)") {
		t.Error("o comando de condição não chama `characterChanged`: o mestre aplica " +
			"Caído e a tela do jogador segue com a Defesa velha, sem nada acusar")
	}
	// CONTROLE: o recorte achou a função de verdade, e não uma string vazia.
	if !strings.Contains(corpo, "UpdateConditions") {
		t.Fatal("o recorte não pegou o corpo de `toggleBookCondition` — o guarda mediria o vazio")
	}
}

// O EFEITO APLICADO É DE QUEM O APLICOU, e a query não confere isso sozinha.
//
// O `DeleteEffectByID` apaga por id e mais nada. Sem a leitura de posse que o
// comando faz antes, um `@post` montado à mão encerraria o efeito de OUTRO
// personagem — e o 403 do `comandoDaFicha` não pega, porque a ficha do caminho é
// a do dono.
func TestOEfeitoDeOutraFichaNaoSeEncerra(t *testing.T) {
	f, meu := oCombatente(t)
	outro := seedCharacterAtLevel(t, f.s, f.jogador, "Vizinho", 1, 10, 10, 0, 0)
	alheio, err := f.s.queries.CreateActiveEffect(context.Background(), sqlcgen.CreateActiveEffectParams{
		Characterid: outro, Catalogid: "armadura-arcana", Scope: "scene",
		Modifiers: "[]", Createdat: plataforma.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear o efeito alheio: %v", err)
	}

	rec := oEfeito(t, f, meu, fmt.Sprintf("encerra/%d", alheio.ID))
	if recusa := aRecusaDaCena(rec.Body); recusa == "" {
		t.Error("encerrei o efeito de outro personagem pela minha ficha")
	}
	if _, err := f.s.queries.GetActiveEffectMeta(context.Background(), alheio.ID); err != nil {
		t.Error("o efeito alheio foi apagado assim mesmo")
	}
}

// AS POSTURAS SAEM DO CATÁLOGO, e a flag vem do PODER de mesmo id.
//
// A SPA guarda um `FLAG_ACTIVATIONS` escrito à mão; as duas posturas já estavam
// no `activations.json` como `kind: "stance"`. Derivar a flag do último pedaço
// do id acertaria as duas de hoje e erraria calado na terceira — por isso ela sai
// do `condition.flag` dos modificadores do poder.
func TestAsPosturasSaemDoCatalogoComAFlagDoPoder(t *testing.T) {
	posturas := stancesFromCatalog()
	if len(posturas) < 2 {
		t.Fatalf("o catálogo ofereceu %d posturas, e são pelo menos 2 (Fúria e Inspiração): "+
			"a leitura do `kind: stance` parou de casar", len(posturas))
	}
	furia, tem := posturas["furia"]
	if !tem {
		t.Fatalf("a flag `furia` não saiu do catálogo; saíram %v", chavesDe(posturas))
	}
	// Os números são do LIVRO e escritos à mão: Bárbaro p40, 2 PM.
	if furia.Name != "Fúria" || furia.PM != 2 || furia.Page != 40 {
		t.Errorf("a Fúria saiu como %+v, quer {Fúria 2 PM p40}", furia)
	}
	if _, tem := posturas["inspiracao"]; !tem {
		t.Error("a flag `inspiracao` não saiu do catálogo")
	}
}

func chavesDe(m map[string]stanceOfBook) []string {
	fora := make([]string, 0, len(m))
	for k := range m {
		fora = append(fora, k)
	}
	return fora
}

// A POSTURA NÃO APARECE NA SITUAÇÃO, porque o interruptor dela é dos Poderes.
//
// Entrar numa postura custa PM, e este painel não tem como cobrar. Deixá-la na
// lista de situação daria um interruptor que liga a Fúria DE GRAÇA — e o PM não
// seria cobrado em lugar nenhum, porque quem cobra é o gesto do outro painel.
func TestAPosturaNaoApareceNaListaDeSituacao(t *testing.T) {
	oferecidos := []engine.ConditionalEffect{
		{Source: "Fúria", Note: "Em Fúria", Amount: 2, BonusType: "morale", Flag: "furia",
			Target: engine.ModifierTarget{K: "attack", Scope: "all"}},
		{Source: "Botas do Bosque", Note: "em terreno de floresta", Amount: 2, BonusType: "untyped",
			Target: engine.ModifierTarget{K: "expertise", Name: "Furtividade"}},
	}
	linhas, _ := situationalRowsOf(oferecidos, map[string]bool{})

	if len(linhas) != 1 {
		t.Fatalf("saíram %d linhas de situação, quer 1: a postura entrou na lista", len(linhas))
	}
	if !strings.Contains(linhas[0].Label, "floresta") {
		t.Errorf("a linha que sobrou é %q, e devia ser a das botas", linhas[0].Label)
	}
}

// UM GRUPO COM MAIS DE UM MODIFICADOR É UM INTERRUPTOR SÓ.
//
// Um item caseiro com três modificadores é uma coisa na mesa; como três linhas, a
// pessoa deixaria metade do efeito ligado.
func TestOsModificadoresQueDividemFlagViramUmInterruptor(t *testing.T) {
	oferecidos := []engine.ConditionalEffect{
		{Source: "Manto Caseiro", Note: "com o manto vestido", Amount: 2, Flag: "homebrew-manto",
			Target: engine.ModifierTarget{K: "defense"}},
		{Source: "Manto Caseiro", Note: "com o manto vestido", Amount: 1, Flag: "homebrew-manto",
			Target: engine.ModifierTarget{K: "expertise", Name: "Furtividade"}},
	}
	linhas, _ := situationalRowsOf(oferecidos, map[string]bool{})
	if len(linhas) != 1 {
		t.Fatalf("saíram %d interruptores, quer 1", len(linhas))
	}
	if !linhas[0].Folded {
		t.Error("o grupo não se marcou como dobrado: a tela não diria que ele liga mais de um")
	}
	if len(linhas[0].Modifiers) != 2 {
		t.Errorf("o grupo mostra %d modificadores, quer 2", len(linhas[0].Modifiers))
	}
}

// lerFonte lê um arquivo do próprio pacote.
func lerFonte(t *testing.T, nome string) string {
	t.Helper()
	bruto, err := os.ReadFile(nome)
	if err != nil {
		t.Fatalf("ler %s: %v", nome, err)
	}
	return string(bruto)
}

// recorteDaFuncao devolve o corpo de uma função, do cabeçalho até a chave que a
// fecha na coluna zero.
func recorteDaFuncao(t *testing.T, fonte, cabecalho string) string {
	t.Helper()
	ini := strings.Index(fonte, cabecalho)
	if ini < 0 {
		t.Fatalf("não achei %q na fonte — a função foi renomeada?", cabecalho)
	}
	resto := fonte[ini:]
	fim := strings.Index(resto, "\n}\n")
	if fim < 0 {
		return resto
	}
	return resto[:fim]
}

// O PAINEL CHEGA NA TELA.
func TestOPainelDeEfeitosDesenhaOsQuatroBlocos(t *testing.T) {
	f, id := oCombatente(t)
	tela := aTelaDosEfeitos(t, f, id)

	for _, esperado := range []string{"Condições (p394)", "Efeitos ativos", "Aplicar condição", "Aplicar magia"} {
		if !strings.Contains(tela, esperado) {
			t.Errorf("a tela não tem %q", esperado)
		}
	}
}
