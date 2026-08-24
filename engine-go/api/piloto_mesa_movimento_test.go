package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"t20engine/tabuleiro"
)

// Os guardas do MOVIMENTO na Mesa (ALE-266).
//
// A régua do livro tem guarda no `engine` (custo, diagonal, terreno difícil,
// alcance) e a autorização tem no `tabuleiro` (`assertMovable`). O que se prende
// aqui é a COMPOSIÇÃO: que a cena pergunta a coisa certa, e que as paradas se
// ACUMULAM em vez de se substituírem.

// noTabuleiro abre um tabuleiro com uma peça do personagem do jogador, e devolve
// o id dela.
func (f pilotoFixture) noTabuleiro(t *testing.T) string {
	t.Helper()
	f.abreTabuleiro(t, "pedra")
	entryID := f.naFila(t)
	posto, err := f.s.boards.AddToken(context.Background(), f.sessionID,
		tabuleiro.BoardToken{Label: "Arcanista", X: 0, Y: 0, EntryID: &entryID, CharacterID: &f.charID}, true)
	if err != nil {
		t.Fatalf("pôr a peça: %v", err)
	}
	return posto.Tokens[len(posto.Tokens)-1].ID
}

// TestAsParadasSeACUMULAMemVezDeSeSubstituirem — o coração desta fatia.
//
// Uma parada por clique, e o caminho ESTENDE. Se cada clique recomeçasse do
// lugar da peça, o contorno seria impossível de expressar — que é exatamente o
// defeito da SPA que a ALE-266 abriu.
func TestAsParadasSeAcumulamEmVezDeSeSubstituirem(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.noTabuleiro(t)
	base := f.urlDaMesa() + "/tabuleiro/" + tokenID

	// O mestre move sem orçamento, então ele serve para medir o acúmulo sem a
	// regra da vez entrar no meio.
	if rec := f.pede(t, f.mestre, "POST", base+"/parada/2/0", ""); rec.Code != http.StatusOK {
		t.Fatalf("primeira parada deu %d", rec.Code)
	}
	primeiro := f.s.boards.Get(context.Background(), f.sessionID).Pending
	if primeiro == nil || len(primeiro.Path) != 3 {
		t.Fatalf("o primeiro caminho ficou %+v", primeiro)
	}

	if rec := f.pede(t, f.mestre, "POST", base+"/parada/2/2", ""); rec.Code != http.StatusOK {
		t.Fatalf("segunda parada deu %d", rec.Code)
	}
	depois := f.s.boards.Get(context.Background(), f.sessionID).Pending
	if depois == nil {
		t.Fatal("o movimento sumiu na segunda parada")
	}
	// (0,0)→(2,0) são 3 quadrados; mais (2,1),(2,2) dão 5. Substituir daria 3.
	if len(depois.Path) != 5 {
		t.Errorf("o caminho ficou com %d quadrados: %+v — a segunda parada substituiu em vez de estender", len(depois.Path), depois.Path)
	}
	if fim := depois.Path[len(depois.Path)-1]; fim.X != 2 || fim.Y != 2 {
		t.Errorf("o caminho termina em %+v", fim)
	}
}

// TestOMovimentoSoPousaNoCONFIRMAR: a peça não anda enquanto o movimento é
// proposta. É o que deixa a pessoa contornar em vários cliques sem a mesa ver a
// peça pulando de casa em casa.
func TestOMovimentoSoPousaNoConfirmar(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.noTabuleiro(t)
	base := f.urlDaMesa() + "/tabuleiro/" + tokenID
	onde := func() (int, int) {
		p := tabuleiro.FindToken(f.s.boards.Get(context.Background(), f.sessionID), tokenID)
		return p.X, p.Y
	}

	if rec := f.pede(t, f.mestre, "POST", base+"/parada/3/1", ""); rec.Code != http.StatusOK {
		t.Fatalf("propor deu %d", rec.Code)
	}
	if x, y := onde(); x != 0 || y != 0 {
		t.Errorf("a peça andou na PROPOSTA, para %d,%d", x, y)
	}

	if rec := f.pede(t, f.mestre, "POST", base+"/confirmar", ""); rec.Code != http.StatusOK {
		t.Fatalf("confirmar deu %d", rec.Code)
	}
	if x, y := onde(); x != 3 || y != 1 {
		t.Errorf("depois de confirmar a peça está em %d,%d", x, y)
	}
	if f.s.boards.Get(context.Background(), f.sessionID).Pending != nil {
		t.Error("o movimento continuou pendente depois de confirmado")
	}
}

// E CANCELAR não mexe na peça: ela volta a poder ser movida de onde estava.
func TestCancelarNaoMexeNaPeca(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.noTabuleiro(t)
	base := f.urlDaMesa() + "/tabuleiro/" + tokenID

	if rec := f.pede(t, f.mestre, "POST", base+"/parada/4/4", ""); rec.Code != http.StatusOK {
		t.Fatalf("propor deu %d", rec.Code)
	}
	if rec := f.pede(t, f.mestre, "POST", base+"/cancelar", ""); rec.Code != http.StatusOK {
		t.Fatalf("cancelar deu %d", rec.Code)
	}

	b := f.s.boards.Get(context.Background(), f.sessionID)
	if b.Pending != nil {
		t.Error("o cancelamento não limpou a proposta")
	}
	if p := tabuleiro.FindToken(b, tokenID); p.X != 0 || p.Y != 0 {
		t.Errorf("a peça ficou em %d,%d depois do cancelamento", p.X, p.Y)
	}
}

// TestOJogadorNaoMoveAPecaDeOUTREM — a autorização é do `tabuleiro`, e a recusa
// vem com a FRASE que a regra escreve.
//
// Não é 403: quem chega aqui é da mesa e podia estar movendo a própria peça. A
// diferença importa porque a frase é o que a pessoa lê — "a peça não é sua" diz
// o que fazer, e "proibido" não.
func TestOJogadorNaoMoveAPecaDeOutrem(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")
	posto, err := f.s.boards.AddToken(context.Background(), f.sessionID,
		tabuleiro.BoardToken{Label: "Ogro", X: 5, Y: 5}, true)
	if err != nil {
		t.Fatalf("pôr o Ogro: %v", err)
	}
	ogro := posto.Tokens[len(posto.Tokens)-1].ID

	corpo := f.pede(t, f.jogador, "POST",
		f.urlDaMesa()+"/tabuleiro/"+ogro+"/parada/6/5", "").Body.String()
	if !strings.Contains(corpo, "não é sua") {
		t.Errorf("a recusa não explica de quem é a peça; sinais = %s", trechoDeSinais(corpo))
	}
	if f.s.boards.Get(context.Background(), f.sessionID).Pending != nil {
		t.Error("o movimento recusado virou proposta mesmo assim")
	}
}

// TestOAlcanceSOapareceQuandoHaOrcamento.
//
// Quem tem teto é o jogador NA VEZ dele. O mestre move sem orçamento (-1), e
// desenhar alcance para ele seria inventar um limite que a regra não põe — foi
// isto que fez a casa alcançável deixar de ser o alvo do clique e virar pintura.
func TestOAlcanceSoApareceQuandoHaOrcamento(t *testing.T) {
	f := novoPiloto(t)
	f.noTabuleiro(t)
	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/scene/start", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}
	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/initiative/next-turn", ""); rec.Code != http.StatusOK {
		t.Fatalf("avançar deu %d", rec.Code)
	}

	// O CONTROLE: o jogador vê o tabuleiro. Sem isto, "não achei alcance" seria
	// verdade também numa cena sem mapa.
	doJogador := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if !strings.Contains(doJogador, "tabuleiro-plano") {
		t.Fatal("o jogador não viu o tabuleiro")
	}
	if !strings.Contains(doJogador, "tabuleiro-alcance") {
		t.Error("é a vez do jogador e ele não viu até onde pode andar")
	}

	doMestre := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if strings.Contains(doMestre, "tabuleiro-alcance") {
		t.Error("o mestre não tem orçamento e mesmo assim viu um teto desenhado")
	}
}

// TestARecusaDeUmaParadaFALAnoTabuleiro.
//
// O arrasto (ALE-264) quebrou a invariante em que este arquivo se apoiava: com
// CLIQUE só se acerta casa oferecida, mas soltar acontece onde o dedo estiver,
// inclusive fora do alcance. A recusa passou a ser alcançável de verdade — e
// ela saía em `erroDoComando`, que é o sinal do RODAPÉ DO MESTRE. O jogador não
// renderiza rodapé nenhum: a frase existia no fio e não tinha onde pousar, e a
// parada era engolida em silêncio.
//
// Prende as DUAS metades, porque uma sem a outra não é o conserto: que a frase
// sai no sinal certo, e que a região do tabuleiro tem onde acendê-la.
func TestARecusaDeUmaParadaFalaNoTabuleiro(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.noTabuleiro(t)
	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/scene/start", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}
	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/initiative/next-turn", ""); rec.Code != http.StatusOK {
		t.Fatalf("avançar deu %d", rec.Code)
	}
	base := f.urlDaMesa() + "/tabuleiro/" + tokenID

	// O CANAL: a região do tabuleiro tem o elemento ligado ao sinal. Sem esta
	// asserção, "a frase saiu" seria verdade sobre uma tela que não a mostra —
	// que é exatamente o defeito que este guarda existe para pegar.
	doJogador := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if !strings.Contains(doJogador, "$erroDoMovimento") {
		t.Fatal("o tabuleiro do jogador não tem onde acender a recusa de uma parada")
	}

	// O deslocamento padrão são 6 quadrados (T20 p106); 9 não cabem.
	recusado := f.pede(t, f.jogador, "POST", base+"/parada/9/0", "").Body.String()
	sinais := trechoDeSinais(recusado)
	if !strings.Contains(sinais, "erroDoMovimento") {
		t.Errorf("a recusa não saiu no sinal do movimento; sinais = %s", sinais)
	}
	if strings.Contains(sinais, `"erroDoMovimento":""`) {
		t.Errorf("a recusa saiu VAZIA — a parada foi engolida em silêncio; sinais = %s", sinais)
	}

	// E APAGA no acerto: um sinal que só se escreve quando dá errado deixa a
	// recusa de duas paradas atrás acesa sobre uma que funcionou.
	aceito := f.pede(t, f.jogador, "POST", base+"/parada/2/0", "").Body.String()
	if !strings.Contains(trechoDeSinais(aceito), `"erroDoMovimento":""`) {
		t.Errorf("a parada válida não apagou a recusa anterior; sinais = %s", trechoDeSinais(aceito))
	}
}

// TestOQueSOBRAdoDeslocamentoAparecePorEscrito.
//
// A realimentação que o dono pediu por nome: sem ela a pessoa empilha paradas
// que no fim somam mais do que ela anda, e descobre no bloqueio sem saber o que
// desfazer. O alcance desenhado é o aviso mudo; este número é o falado.
//
// Guarda também a CONTA, que estava sem dono: `Alcance` e `Restante` são os dois
// valores de UMA chamada de `AlcanceDaProximaParada`, e enquanto ninguém
// afirmava o segundo dava para movê-lo de lugar sem nenhum teste piscar.
func TestOQueSobraDoDeslocamentoAparecePorEscrito(t *testing.T) {
	f := novoPiloto(t)
	tokenID := f.noTabuleiro(t)
	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/scene/start", ""); rec.Code != http.StatusOK {
		t.Fatalf("iniciar cena deu %d", rec.Code)
	}
	if rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/initiative/next-turn", ""); rec.Code != http.StatusOK {
		t.Fatalf("avançar deu %d", rec.Code)
	}

	// Duas casas em linha reta custam 2 do deslocamento padrão de 6 (T20 p106).
	if rec := f.pede(t, f.jogador, "POST", f.urlDaMesa()+"/tabuleiro/"+tokenID+"/parada/2/0", ""); rec.Code != http.StatusOK {
		t.Fatalf("a parada deu %d", rec.Code)
	}
	tela := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()

	// O CONTROLE: a frase do movimento está na tela. Sem ele, não achar "sobram"
	// seria verdade também numa tela sem movimento proposto nenhum.
	if !strings.Contains(tela, "quadrados") {
		t.Fatal("o jogador não viu o movimento proposto")
	}
	if !strings.Contains(tela, "2 de 6") {
		t.Errorf("a tela não diz o gasto contra o teto")
	}
	if !strings.Contains(tela, "sobram 4") {
		t.Errorf("a tela não diz quanto ainda dá para andar")
	}
}
