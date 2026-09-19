package sheetui

import (
	"os"
	"strings"
	"testing"

	"t20engine/domain/engine"
)

// Os guardas do painel de EFEITOS.
//
// O que eles prendem é o que separa este painel de uma lista bonita: a condição
// MOVE os números, a mesa é avisada, e a fronteira de cada gesto é do servidor.

type responseRecorderLike struct {
	Code int
	Body string
}

// A CONDIÇÃO AVISA A MESA AO VIVO.
//
// O motor deriva Defesa e perícias da condição, então uma aplicada sem aviso faz
// o jogador e o mestre verem números DIFERENTES do mesmo personagem, sem nada na
// tela dizendo que discordam.
func TestAConditionAnnouncesItselfToTheLiveTable(t *testing.T) {
	// O guarda lê a FONTE do handler, e não um evento no fio: montar uma sessão
	// viva com este personagem na fila para ouvir um SSE é caro, e o que se quer
	// prender é que a CHAMADA não some — é assim que um aviso se perde, num
	// gancho que ninguém preenche.
	fonte := lerFonte(t, "effects_commands.go")
	corpo := functionSlice(t, fonte, "func toggleBookCondition")
	// CONTROLE: o recorte achou a função de verdade, e não uma string vazia.
	//
	// A âncora é a CHAMADA ao caso de uso, e não mais o `UpdateConditions` — a
	// escrita desceu para o `character.Plays` na ALE-351, e um controle preso ao
	// nome da consulta passou a medir o vazio.
	gravacao := strings.Index(corpo, "s.plays.ToggleBookCondition")
	if gravacao < 0 {
		t.Fatal("o recorte não pegou o corpo de `toggleBookCondition` — o guarda mediria o vazio")
	}
	aviso := strings.Index(corpo, "s.deps.CharacterChanged(row.ID)")
	if aviso < 0 {
		t.Fatal("o comando de condição não chama `CharacterChanged`: o mestre aplica " +
			"Caído e a tela do jogador segue com a Defesa velha, sem nada acusar")
	}
	// E a ORDEM, que é a outra metade da regra: avisar sobre algo que ainda pode
	// falhar faria a mesa buscar o estado velho e acreditar nele.
	if aviso < gravacao {
		t.Error("o aviso à mesa sai ANTES da gravação: a mesa buscaria o estado velho")
	}
}

// AS POSTURAS SAEM DO CATÁLOGO, e a flag vem do PODER de mesmo id.
//
// Derivar a flag do último pedaço do id acerta as duas de hoje e erra calado na
// terceira — por isso ela sai do `condition.flag` dos modificadores do poder.
func TestStancesComeFromTheCatalogWithThePowerFlag(t *testing.T) {
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
func TestAStanceDoesNotShowInTheConditionalList(t *testing.T) {
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
func TestModifiersSharingAFlagBecomeOneSwitch(t *testing.T) {
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

// functionSlice devolve o corpo de uma função, do cabeçalho até a chave que a
// fecha na coluna zero.
func functionSlice(t *testing.T, fonte, cabecalho string) string {
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
