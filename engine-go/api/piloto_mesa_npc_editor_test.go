package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"t20engine/creature"
	"t20engine/db/sqlcgen"
)

// Os guardas do EDITOR DE BLOCO (ALE-269).
//
// A forma do bloco tem guarda em `creature_block.go` (a validação) e a cópia do
// livro tem o oráculo do `piloto_verbete_para_bloco`. O que se prende aqui é o
// que só existe DESDE o editor: que o rascunho atravessa os gestos de forma sem
// perder o que estava digitado, que a AUSÊNCIA de mana sobrevive à ida e à volta
// pelo formulário, que a recusa fala DENTRO do editor, e que o id do rascunho não
// alcança o elenco de outra campanha.

// oRascunhoNoCorpo monta o corpo de sinais que o navegador mandaria.
//
// Escrito como TEXTO e não `json.Marshal` de um `rascunhoDoNPC`: marshalar a
// struct faria o teste mandar exatamente o que o servidor espera, e um campo
// renomeado passaria verde nos dois lados. Aqui o teste fala a língua do FIO.
func oRascunhoNoCorpo(dentro string) string {
	return `{"rascunho":{` + dentro + `}}`
}

const blocoMinimo = `"nd":1,"tipo":"humanoide","size":"medio","hp":10,"defesa":10,` +
	`"deslocamento":"9m (6q)","attacks":[],"skills":[],"specialAbilities":[]`

// oRascunhoDaResposta extrai o rascunho do quadro de sinais do SSE.
//
// Ler a CHAVE e não procurar o texto solto na resposta, e isto custou uma
// sabotagem para descobrir: `Contains(resposta, "Ogro Capitão")` passa verde com
// o sinal renomeado, porque o nome continua no corpo — ligado a coisa nenhuma. O
// que a tela precisa é do valor sob `rascunho`, e é isso que se afirma.
func oRascunhoDaResposta(t *testing.T, resposta string) map[string]any {
	t.Helper()
	const marca = "data: signals "
	i := strings.Index(resposta, marca)
	if i < 0 {
		t.Fatalf("a resposta não trouxe sinais:\n%s", resposta)
	}
	linha := resposta[i+len(marca):]
	if fim := strings.IndexByte(linha, '\n'); fim >= 0 {
		linha = linha[:fim]
	}
	var sinais struct {
		Rascunho map[string]any `json:"rascunho"`
	}
	if err := json.Unmarshal([]byte(linha), &sinais); err != nil {
		t.Fatalf("os sinais não são JSON: %v\n%s", err, linha)
	}
	if sinais.Rascunho == nil {
		t.Fatalf("a resposta não trouxe `rascunho`:\n%s", linha)
	}
	return sinais.Rascunho
}

// npcNoBanco lê o bloco guardado pelo nome, para as asserções não dependerem do
// id que o banco escolheu.
func npcNoBanco(t *testing.T, f pilotoFixture, nome string) creature.Block {
	t.Helper()
	for _, npc := range f.s.oElencoDaCampanha(context.Background(), f.campaignID) {
		if npc.Nome != nome {
			continue
		}
		linhas, err := f.s.queries.ListCampaignCreatures(context.Background(), f.campaignID)
		if err != nil {
			t.Fatalf("ler o elenco: %v", err)
		}
		for _, l := range linhas {
			if l.Name != nome {
				continue
			}
			var bloco creature.Block
			if err := json.Unmarshal([]byte(l.Block), &bloco); err != nil {
				t.Fatalf("o bloco de %q está ilegível: %v", nome, err)
			}
			return bloco
		}
	}
	t.Fatalf("%q não está no elenco", nome)
	return creature.Block{}
}

// TestOGestoDeFORMAnaoGravaEnaoPerdeODigitado — o coração desta superfície.
//
// Acrescentar um ataque PRECISA do servidor (Datastar não tem laço no cliente),
// e é aí que mora o risco: se o gesto gravasse, "Cancelar desfaz de verdade"
// seria mentira; se ele não devolvesse o rascunho inteiro, o nome que estava
// sendo digitado sumiria no clique.
//
// Este caso afirma as duas metades de uma vez: o banco continua vazio E o nome
// volta na resposta.
func TestOGestoDeFormaNaoGravaENaoPerdeODigitado(t *testing.T) {
	f := novoPiloto(t)

	resposta := f.posta(t, f.mestre,
		f.urlDaMesa()+"/elenco/npc/rascunho/"+listaDeAtaques+"/nova",
		oRascunhoNoCorpo(`"id":0,"nome":"Ogro Capitão","conjura":false,"bloco":{`+blocoMinimo+`}`))

	if elenco := f.s.oElencoDaCampanha(context.Background(), f.campaignID); len(elenco) != 0 {
		t.Errorf("acrescentar um ataque GRAVOU no elenco: %+v", elenco)
	}
	// O nome volta DENTRO de `rascunho`, e a asserção lê a chave em vez de
	// procurar o texto na resposta inteira. A primeira versão procurava
	// `Contains(resposta, "Ogro Capitão")` e passou verde com o sinal renomeado
	// para `naoerascunho` — o texto estava lá, ligado a coisa nenhuma, e o teste
	// não sabia a diferença.
	if nome := oRascunhoDaResposta(t, resposta)["nome"]; nome != "Ogro Capitão" {
		t.Errorf("o rascunho voltou com nome %v, esperado o que estava sendo digitado:\n%s", nome, resposta)
	}
	// A LINHA nova volta como HTML, e não só como sinal: um `data-bind` para uma
	// posição que a tela não desenhou é um campo que não existe.
	if !strings.Contains(resposta, `id="npc-ataques"`) {
		t.Errorf("a lista de ataques não foi redesenhada:\n%s", resposta)
	}
	if !strings.Contains(resposta, oCampoDaLinha(listaDeAtaques, 0, "name")) {
		t.Errorf("a linha nova não tem onde escrever o nome:\n%s", resposta)
	}
}

// TestTirarUmaLinhaTiraAQUELAlinha.
//
// Índice fora por um é a classe de erro clássica desta operação, e o sintoma na
// mesa é o pior possível: o mestre clica no lixo do terceiro ataque e some o
// segundo, que ele acabou de escrever.
func TestTirarUmaLinhaTiraAquelaLinha(t *testing.T) {
	f := novoPiloto(t)
	tres := `"attacks":[{"name":"Clava"},{"name":"Mordida"},{"name":"Cauda"}],"skills":[],"specialAbilities":[]`
	resposta := f.posta(t, f.mestre,
		f.urlDaMesa()+"/elenco/npc/rascunho/"+listaDeAtaques+"/1/remover",
		oRascunhoNoCorpo(`"id":0,"nome":"Hidra","conjura":false,"bloco":{"nd":1,"tipo":"monstro","size":"grande","hp":10,"defesa":10,"deslocamento":"9m",`+tres+`}`))

	if strings.Contains(resposta, "Mordida") {
		t.Errorf("a linha do meio sobreviveu ao remover:\n%s", resposta)
	}
	for _, sobrevivente := range []string{"Clava", "Cauda"} {
		if !strings.Contains(resposta, sobrevivente) {
			t.Errorf("%q sumiu junto com a linha removida:\n%s", sobrevivente, resposta)
		}
	}
}

// TestUmaLinhaQueNaoExisteRecusaEmVezDeEstourar.
//
// O índice vem do BOTÃO, e o botão pode ser de uma tela velha — outra aba já
// tirou a linha. Um `panic` aqui derrubaria a resposta inteira; a recusa com o
// número diz o que aconteceu.
func TestUmaLinhaQueNaoExisteRecusaEmVezDeEstourar(t *testing.T) {
	f := novoPiloto(t)
	resposta := f.posta(t, f.mestre,
		f.urlDaMesa()+"/elenco/npc/rascunho/"+listaDeAtaques+"/7/remover",
		oRascunhoNoCorpo(`"id":0,"nome":"Ogro","conjura":false,"bloco":{`+blocoMinimo+`}`))
	if !strings.Contains(resposta, "erroDoRascunho") {
		t.Errorf("a linha inexistente não recusou:\n%s", resposta)
	}
}

// TestAAUSENCIAdeManaSobreviveAoFormulario.
//
// A linha de PM só existe em quem conjura — o Centauro Xamã tem 20 PM (p290), o
// Bandido não tem linha nenhuma —, e um zero ali diria "tem mana e está sem", que
// é outro estado e o errado na hora de gastar.
//
// O formulário não sabe digitar "ausente": ele guarda um número e um interruptor.
// Este caso prende a tradução nos DOIS sentidos, porque é onde ela se perde.
func TestAAusenciaDeManaSobreviveAoFormulario(t *testing.T) {
	f := novoPiloto(t)
	base := f.urlDaMesa() + "/elenco/npc/rascunho/salvar"

	// Sem conjurar: o número no formulário é ignorado e o bloco fica SEM a linha.
	f.posta(t, f.mestre, base,
		oRascunhoNoCorpo(`"id":0,"nome":"Bandido","conjura":false,"bloco":{`+blocoMinimo+`,"pm":7}`))
	if pm := npcNoBanco(t, f, "Bandido").PM; pm != nil {
		t.Errorf("o Bandido guardou %d PM sem conjurar — a ausência virou número", *pm)
	}

	// Conjurando: o número atravessa.
	f.posta(t, f.mestre, base,
		oRascunhoNoCorpo(`"id":0,"nome":"Centauro Xamã","conjura":true,"bloco":{`+blocoMinimo+`,"pm":20}`))
	pm := npcNoBanco(t, f, "Centauro Xamã").PM
	if pm == nil || *pm != 20 {
		t.Errorf("o Centauro Xamã guardou %v PM, esperado 20", pm)
	}
}

// TestOFormularioNaoNasceComAPalavraUNDEFINED.
//
// Medido no navegador antes de virar teste: campo opcional com `omitempty` sai
// AUSENTE do sinal, e um `data-bind` para um caminho ausente escreve a palavra
// "undefined" dentro da caixa — que o mestre então salva como o efeito do ataque.
//
// O guarda é sobre o que o servidor MANDA: os campos opcionais têm de estar lá,
// com valor vazio, e não faltando.
func TestOFormularioNaoNasceComAPalavraUndefined(t *testing.T) {
	f := novoPiloto(t)
	comOpcionaisVazios := `"attacks":[{"name":"Clava","attackBonus":7,"damage":"1d6+3"}],` +
		`"skills":[{"name":"Furtividade","bonus":5}],"specialAbilities":[]`
	f.posta(t, f.mestre, f.urlDaMesa()+"/elenco/npc/rascunho/salvar",
		oRascunhoNoCorpo(`"id":0,"nome":"Ogro","conjura":false,"bloco":{"nd":1,"tipo":"monstro","size":"grande","hp":10,"defesa":10,"deslocamento":"9m",`+comOpcionaisVazios+`}`))

	id := oIDDoNPC(t, f, "Ogro")
	aberto := f.posta(t, f.mestre, f.urlDaMesa()+"/elenco/npc/"+id+"/editar", "{}")
	for _, campo := range []string{`"special"`, `"nota"`, `"pm"`} {
		if !strings.Contains(aberto, campo) {
			t.Errorf("o campo %s não veio no rascunho — a caixa dele nasceria escrita \"undefined\":\n%s", campo, aberto)
		}
	}
}

// TestSalvarSemNomeFalaDENTROdoEditor.
//
// A recusa do `comandoDoMestre` sai por padrão no `erroDoComando`, que é o rodapé
// do mestre — e o editor é um DIÁLOGO por cima dele. Medido no navegador: salvar
// sem nome não dizia absolutamente nada, porque a frase estava atrás do painel.
//
// E a frase é em PORTUGUÊS: ela era inglesa enquanto o formulário sempre foi
// português, e "creature name is required" ao lado de uma caixa escrita "Nome"
// manda o mestre procurar um campo que não existe.
func TestSalvarSemNomeFalaDentroDoEditor(t *testing.T) {
	f := novoPiloto(t)
	resposta := f.posta(t, f.mestre, f.urlDaMesa()+"/elenco/npc/rascunho/salvar",
		oRascunhoNoCorpo(`"id":0,"nome":"","conjura":false,"bloco":{`+blocoMinimo+`}`))

	if !strings.Contains(resposta, "erroDoRascunho") || !strings.Contains(resposta, "precisa de um nome") {
		t.Errorf("a recusa não falou no editor:\n%s", resposta)
	}
	if elenco := f.s.oElencoDaCampanha(context.Background(), f.campaignID); len(elenco) != 0 {
		t.Errorf("o NPC sem nome foi gravado assim mesmo: %+v", elenco)
	}
}

// TestOEditorNaoAlcancaOElencoDeOUTRAcampanha — a trava que mais importa.
//
// O id vem do RASCUNHO, que vem do navegador: sem a conferência de campanha, o
// mestre de uma mesa reescreveria a preparação de outra. É a mesma trava que o
// `oNPCDaCampanha` já fazia para o caminho, e o editor tinha de reusá-la em vez
// de confiar no número que chegou.
func TestOEditorNaoAlcancaOElencoDeOutraCampanha(t *testing.T) {
	f := novoPiloto(t)
	// Um NPC guardado numa campanha VIZINHA do mesmo mestre.
	vizinha := seedCampaign(t, f.s, f.mestre)
	agora := "2026-01-01T00:00:00Z"
	linha, err := f.s.queries.CreateCampaignCreature(context.Background(), sqlcgen.CreateCampaignCreatureParams{
		Campaignid: vizinha, Name: "Vilão da vizinha",
		Block:     `{` + blocoMinimo + `}`,
		Createdat: agora, Updatedat: agora,
	})
	if err != nil {
		t.Fatalf("semear o NPC vizinho: %v", err)
	}

	resposta := f.posta(t, f.mestre, f.urlDaMesa()+"/elenco/npc/rascunho/salvar",
		oRascunhoNoCorpo(`"id":`+strconv.FormatInt(linha.ID, 10)+`,"nome":"Sequestrado","conjura":false,"bloco":{`+blocoMinimo+`}`))

	if !strings.Contains(resposta, "não é desta campanha") {
		t.Errorf("o editor aceitou reescrever o elenco da vizinha:\n%s", resposta)
	}
	depois, err := f.s.queries.GetCampaignCreature(context.Background(), linha.ID)
	if err != nil {
		t.Fatalf("reler o NPC vizinho: %v", err)
	}
	if depois.Name != linha.Name {
		t.Errorf("o NPC da vizinha virou %q", depois.Name)
	}
}

// TestSoOMestreMexeNoElenco: a trava é do servidor, e não o botão escondido.
func TestSoOMestreMexeNoElenco(t *testing.T) {
	f := novoPiloto(t)
	for _, caminho := range []string{
		"/elenco/npc/novo",
		"/elenco/npc/rascunho/" + listaDeAtaques + "/nova",
		"/elenco/npc/rascunho/salvar",
	} {
		rec := f.pede(t, f.jogador, http.MethodPost, f.urlDaMesa()+caminho,
			oRascunhoNoCorpo(`"id":0,"nome":"Intruso","conjura":false,"bloco":{`+blocoMinimo+`}`))
		if rec.Code != http.StatusForbidden {
			t.Errorf("o jogador alcançou %s: %d", caminho, rec.Code)
		}
	}
}

// TestCriarDoZeroEEditarSaoOMESMOformulario.
//
// Dois caminhos com duas telas seriam duas telas para envelhecer, e o defeito
// apareceria como "criar do zero não tem a aba de perícias". A prova é que os
// dois abrem devolvendo a MESMA forma de rascunho — o que muda é a semente.
func TestCriarDoZeroEEditarSaoOMesmoFormulario(t *testing.T) {
	f := novoPiloto(t)
	f.posta(t, f.mestre, f.urlDaMesa()+"/elenco/npc/rascunho/salvar",
		oRascunhoNoCorpo(`"id":0,"nome":"Ogro","conjura":false,"bloco":{`+blocoMinimo+`}`))

	doZero := f.posta(t, f.mestre, f.urlDaMesa()+"/elenco/npc/novo", "{}")
	daCopia := f.posta(t, f.mestre, f.urlDaMesa()+"/elenco/npc/"+oIDDoNPC(t, f, "Ogro")+"/editar", "{}")

	for _, campo := range []string{`"nd"`, `"tipo"`, `"size"`, `"attacks"`, `"skills"`, `"specialAbilities"`, `"conjura"`} {
		if !strings.Contains(doZero, campo) {
			t.Errorf("criar do zero não tem %s:\n%s", campo, doZero)
		}
		if !strings.Contains(daCopia, campo) {
			t.Errorf("editar não tem %s:\n%s", campo, daCopia)
		}
	}
	// A diferença é a SEMENTE, e ela precisa aparecer: o do zero não tem id.
	if !strings.Contains(doZero, `"id":0`) {
		t.Errorf("criar do zero nasceu com id:\n%s", doZero)
	}
	if strings.Contains(daCopia, `"id":0`) {
		t.Errorf("editar abriu sem o id do NPC:\n%s", daCopia)
	}
}

// oIDDoNPC acha o id pelo nome, para o teste não depender do número que o banco
// escolheu.
func oIDDoNPC(t *testing.T, f pilotoFixture, nome string) string {
	t.Helper()
	for _, npc := range f.s.oElencoDaCampanha(context.Background(), f.campaignID) {
		if npc.Nome == nome {
			return strconv.FormatInt(npc.ID, 10)
		}
	}
	t.Fatalf("%q não está no elenco", nome)
	return ""
}
