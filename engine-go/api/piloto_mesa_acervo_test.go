package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// Os guardas do ACERVO que distingue ABERTO de FECHADO (ALE-205, fatia 3).
//
// O papercut que os motivou está nas palavras do dono: com 148 lugares, o que
// está na mesa AGORA aparecia no meio dos outros sem nada que o distinguisse —
// porque o `ShowPlace` arquivava a cena atual antes de trocar, e ela voltava
// para a lista como qualquer outra.

// guardaOLugar encerra a cena aberta, o que a manda para o acervo, e devolve o
// id dela.
func (f pilotoFixture) guardaOLugar(t *testing.T, nome string) int64 {
	t.Helper()
	if rec := f.pede(t, f.mestre, http.MethodPost, f.urlDaMesa()+"/tabuleiro/encerrar", ""); rec.Code != http.StatusOK {
		t.Fatalf("encerrar deu %d", rec.Code)
	}
	for _, l := range f.s.boards.Places(context.Background(), f.campaignID) {
		if l.Name == nome {
			return l.ID
		}
	}
	t.Fatalf("%q não foi para o acervo", nome)
	return 0
}

// A LINHA DO QUE ESTÁ NA MESA se distingue das outras, e não oferece "Reabrir".
//
// Reabrir o que já está aberto criaria uma SEGUNDA aba da mesma cena, e as duas
// seriam duas verdades sobre onde as peças estão — com a que fechasse por último
// apagando a outra no acervo, porque o `Archive` sobrescreve pelo nome.
func TestOAcervoDizQualCenaEstaNaMesa(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "taverna") // "Taverna do Javali"
	f.guardaOLugar(t, "Taverna do Javali")
	// Ela volta para a mesa, agora numa aba.
	taverna := f.abreSegunda(t, "Taverna do Javali")

	tela := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()

	if !strings.Contains(tela, "nesta mesa agora") {
		t.Fatal("o acervo não distingue a cena que está na mesa das 147 que não estão")
	}
	// A asserção é ESCOPADA na linha do acervo, e não na página: a barra de abas
	// também escreve `tabuleiro/aba/<id>` (no gesto de mostrar à mesa), então uma
	// busca solta acharia a rota certa no lugar errado e passaria verde sobre uma
	// lista que não leva a lugar nenhum.
	linha := aLinhaDoAcervo(t, tela, "nesta mesa agora")
	// O gesto que sobra é IR até ela, pela mesma rota que a barra de abas usa.
	if !strings.Contains(linha, "/tabuleiro/aba/"+taverna.ID) {
		t.Error("a linha da cena aberta não leva à aba dela")
	}
	if !strings.Contains(linha, ">Ver</button>") {
		t.Error("a linha da cena aberta não oferece o gesto de ir até ela")
	}
	if strings.Contains(linha, ">Reabrir</button>") {
		t.Error("o acervo ofereceu reabrir uma cena que já está aberta")
	}
}

// aLinhaDoAcervo recorta o `<li>` que contém uma marca, e FALHA quando não acha.
//
// Falhar em vez de devolver vazio é o que separa este helper de um instrumento
// mudo: uma busca que não acha nada faria toda asserção seguinte passar sobre
// uma string vazia — o `strings.Contains(vazio, x)` é falso, e "não contém" é
// exatamente o que a maioria dos guardas daqui afirma.
func aLinhaDoAcervo(t *testing.T, tela, marca string) string {
	t.Helper()
	pos := strings.Index(tela, marca)
	if pos < 0 {
		t.Fatalf("não achei %q na tela: a asserção seguinte mediria uma string vazia", marca)
	}
	inicio := strings.LastIndex(tela[:pos], "<li ")
	fim := strings.Index(tela[pos:], "</li>")
	if inicio < 0 || fim < 0 {
		t.Fatalf("a marca %q não está dentro de um <li> do acervo", marca)
	}
	return tela[inicio : pos+fim]
}

// APAGAR DO ACERVO O QUE ESTÁ NA MESA é recusado pelo SERVIDOR.
//
// Sem a recusa o gesto não faz o que diz: a linha some do acervo, a cena
// continua aberta, e no dia em que a aba fechar o `Archive` a grava de novo com
// o mesmo nome. O mestre veria a taverna que ele apagou ontem reaparecer
// sozinha — e desfazer sozinho o trabalho de quem estava limpando o acervo é a
// pior forma de um botão mentir.
func TestNaoSeApagaDoAcervoAcenaQueEstaNaMesa(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "taverna")
	id := f.guardaOLugar(t, "Taverna do Javali")
	f.abreSegunda(t, "Taverna do Javali")

	rec := f.pede(t, f.mestre, http.MethodPost,
		fmt.Sprintf("%s/tabuleiro/lugares/%d/remover", f.urlDaMesa(), id), "")

	// A recusa é 200 com a frase no rodapé do mestre: é o caminho do
	// `comandoDoMestreNoTabuleiro`, e o que se prende é o EFEITO.
	if rec.Code != http.StatusOK {
		t.Fatalf("remover deu %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "encerre a cena antes") {
		t.Error("a recusa não diz o que fazer para conseguir apagar")
	}
	achou := false
	for _, l := range f.s.boards.Places(context.Background(), f.campaignID) {
		if l.ID == id {
			achou = true
		}
	}
	if !achou {
		t.Error("o lugar aberto foi apagado do acervo: ele voltaria sozinho ao encerrar a aba")
	}
	// E a lixeira nem é oferecida — cortesia, não a trava.
	tela := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if strings.Contains(tela, "Apagar Taverna do Javali") {
		t.Error("a lista ofereceu a lixeira para a cena que está na mesa")
	}
}

// REABRIR RESPEITA O TETO, e a recusa diz o que fazer.
//
// É a porta que faltava: o teto vive no `Open`, e o acervo entrou por outro
// caminho. Sem ele, um mestre com oito cenas abertas passaria da nona pela lista
// de lugares — que é justamente onde há 148 botões para clicar.
func TestReabrirRespeitaOTetoDeAbertos(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "taverna")
	id := f.guardaOLugar(t, "Taverna do Javali")
	for i := 0; i < 8; i++ {
		f.abreSegunda(t, fmt.Sprintf("Cena %d", i))
	}

	rec := f.pede(t, f.mestre, http.MethodPost,
		fmt.Sprintf("%s/tabuleiro/lugares/%d/reabrir", f.urlDaMesa(), id), "")

	if rec.Code != http.StatusOK {
		t.Fatalf("reabrir deu %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "feche um antes") {
		t.Error("a recusa não diz o que fazer para caber")
	}
	if n := len(f.s.boards.Abertos(context.Background(), f.sessionID)); n != 8 {
		t.Errorf("a sessão passou do teto pela lista de lugares: %d cenas abertas", n)
	}
}
