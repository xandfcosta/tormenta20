package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/starfederation/datastar-go/datastar"
)

// Os guardas do PUXÃO — "parem tudo e olhem isto" (ALE-205, fatia 2).
//
// Arquivo próprio e não junto dos guardas das abas porque o que eles prendem é
// outro: lá é "cada pessoa escolhe"; aqui é o mestre passando POR CIMA dessa
// escolha por um instante. As duas coisas se contradizem de propósito, e é a
// fronteira entre elas que estes casos seguram.

// O PUXÃO alcança quem NUNCA escolheu aba nenhuma.
//
// É o caso que decidiu o modelo. O mapa de escolhas só conhece quem já clicou;
// quem entrou na sessão e ficou na aba padrão não tem entrada nenhuma, e um laço
// sobre o mapa passaria por cima justamente de quem nunca mexeu em nada — que na
// mesa é a maioria. Por isso o puxão é um contador da SESSÃO, e o zero de quem
// nunca apareceu é menor que qualquer puxão.
func TestOPuxaoAlcancaQuemNuncaEscolheuAba(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra") // a padrão, onde o jogador está sem ter escolhido
	cripta := f.abreSegunda(t, "Cripta")

	// O mestre vai até a cripta e a mostra à mesa.
	f.pede(t, f.mestre, http.MethodPost, f.urlDaMesa()+"/tabuleiro/aba/"+cripta.ID, "")
	if rec := f.pede(t, f.mestre, http.MethodPost, f.urlDaMesa()+"/tabuleiro/aba/"+cripta.ID+"/mostrar", ""); rec.Code != http.StatusOK {
		t.Fatalf("mostrar à mesa deu %d", rec.Code)
	}

	doJogador := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()

	if !strings.Contains(doJogador, "Cripta</h2>") {
		t.Fatal("o jogador que nunca escolheu aba não foi trazido: o puxão não alcança quem está na padrão")
	}
	// E ele SABE que foi trazido, com a saída junto: um mapa que troca sozinho no
	// meio de um turno é lido como defeito da tela.
	if !strings.Contains(doJogador, "O mestre trouxe a mesa para Cripta") {
		t.Error("o mapa do jogador trocou sem nada dizer que foi o mestre")
	}
	if !strings.Contains(doJogador, "Voltar para Taverna do Javali") {
		t.Error("a tira do puxão não oferece o caminho de volta")
	}
	// O MESTRE não lê a tira: ele acabou de fazer o gesto.
	doMestre := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if strings.Contains(doMestre, "O mestre trouxe a mesa para") {
		t.Error("a cena contou ao mestre o que ele mesmo acabou de fazer")
	}
}

// PUXÃO É EMPURRÃO, NÃO TRAVA (decisão do dono).
//
// Depois de trazido, o jogador escolhe outra aba e FICA lá — o quadro seguinte
// do stream não pode devolvê-lo à cena do mestre. Uma trava disfarçada seria o
// pior dos dois mundos: a tela oferece o gesto de voltar e o servidor o desfaz,
// e a pessoa clica três vezes achando que o clique não pegou.
func TestDepoisDoPuxaoOJogadorVoltaAEscolher(t *testing.T) {
	f := novoPiloto(t)
	taverna := f.abreTabuleiro(t, "pedra")
	cripta := f.abreSegunda(t, "Cripta")
	f.pede(t, f.mestre, http.MethodPost, f.urlDaMesa()+"/tabuleiro/aba/"+cripta.ID, "")
	f.pede(t, f.mestre, http.MethodPost, f.urlDaMesa()+"/tabuleiro/aba/"+cripta.ID+"/mostrar", "")

	// Ele usa a saída da tira e volta para a taverna.
	f.pede(t, f.jogador, http.MethodPost, f.urlDaMesa()+"/tabuleiro/aba/"+taverna.ID, "")

	doJogador := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if !strings.Contains(doJogador, "Taverna do Javali</h2>") {
		t.Fatal("o puxão trouxe o jogador de volta depois de ele escolher: virou trava")
	}
	if strings.Contains(doJogador, "O mestre trouxe a mesa para") {
		t.Error("a tira do puxão continuou acesa depois de a pessoa escolher: é um modo sem gesto que o desfaça")
	}
}

// O JOGADOR NÃO PUXA A MESA — a trava é do servidor.
//
// Ela é o ponto desta fatia: "parem tudo e olhem isto" é do mestre. Um jogador
// que puxasse tiraria dos outros cinco exatamente o que a fatia 1 lhes deu.
func TestOJogadorNaoMostraNadaAMesa(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")
	cripta := f.abreSegunda(t, "Cripta")

	rec := f.pede(t, f.jogador, http.MethodPost, f.urlDaMesa()+"/tabuleiro/aba/"+cripta.ID+"/mostrar", "")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("o jogador puxou a mesa: %d", rec.Code)
	}
	// E a mesa não se moveu: o 403 não pode ter deixado rastro.
	doMestre := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if !strings.Contains(doMestre, "Taverna do Javali</h2>") {
		t.Error("a tela do mestre saiu do lugar apesar do 403")
	}
	// O gesto nem aparece para ele: cortesia, não a trava.
	doJogador := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if strings.Contains(doJogador, "à mesa") {
		t.Error("o jogador recebeu o botão de mostrar à mesa")
	}
}

// O PUXÃO PARA UMA CENA ENCERRADA devolve a pessoa PARA A DELA, não para a padrão.
//
// O mestre mostra a cripta, a mesa salta, ele encerra a cripta. Sem esta queda,
// quem foi trazido ficaria olhando uma tela sem tabuleiro — e o gesto que causou
// isso foi de outra pessoa, então ele não teria como ligar uma coisa à outra.
//
// SÃO TRÊS CENAS de propósito, e a primeira versão deste caso tinha duas: com o
// jogador na padrão, "voltar para a escolha dele" e "cair na padrão" dão a MESMA
// tela, e a sabotagem que arranca a queda passou VERDE. Duas quedas diferentes
// só se distinguem quando a escolha da pessoa não é a padrão.
func TestOPuxaoParaUmaCenaEncerradaDevolveAAbaDeQuemOlha(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra") // "Taverna do Javali", a PADRÃO
	ponte := f.abreSegunda(t, "Ponte de Corda")
	cripta := f.abreSegunda(t, "Cripta")
	// O jogador escolheu a ponte — é dela que ele foi tirado.
	f.pede(t, f.jogador, http.MethodPost, f.urlDaMesa()+"/tabuleiro/aba/"+ponte.ID, "")

	f.pede(t, f.mestre, http.MethodPost, f.urlDaMesa()+"/tabuleiro/aba/"+cripta.ID, "")
	f.pede(t, f.mestre, http.MethodPost, f.urlDaMesa()+"/tabuleiro/aba/"+cripta.ID+"/mostrar", "")
	if puxado := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String(); !strings.Contains(puxado, "Cripta</h2>") {
		t.Fatal("o jogador não chegou a ser puxado: o resto do caso não mede nada")
	}

	// O mestre está na cripta e a encerra.
	if rec := f.pede(t, f.mestre, http.MethodPost, f.urlDaMesa()+"/tabuleiro/encerrar", ""); rec.Code != http.StatusOK {
		t.Fatalf("encerrar deu %d", rec.Code)
	}

	doJogador := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if strings.Contains(doJogador, "O mestre ainda não abriu um tabuleiro") {
		t.Fatal("o jogador ficou sem mapa porque a cena para onde ele foi PUXADO acabou")
	}
	if !strings.Contains(doJogador, "Ponte de Corda</h2>") {
		t.Error("o jogador não voltou para a aba que ELE tinha escolhido quando o puxão morreu")
	}
	if strings.Contains(doJogador, "O mestre trouxe a mesa para") {
		t.Error("a tira do puxão sobreviveu à cena que o puxão mostrava")
	}
}

// QUEM JÁ ESTAVA NA ABA também é trazido — para a SUPERFÍCIE do tabuleiro.
//
// É o caso mais comum da mesa, e ele quase passou em branco: o jogador abre a
// sessão na superfície **Mesa**, que é a padrão, e fica na aba padrão. Um puxão
// para essa mesma aba não muda cena nenhuma para ele — e sem a tira e sem o
// empurrão de superfície, "parem tudo e olhem isto" não faria absolutamente nada
// na tela de quem nunca tocou em nada.
func TestOPuxaoParaAAbaEmQueOJogadorJaEstaAindaOAvisa(t *testing.T) {
	f := novoPiloto(t)
	taverna := f.abreTabuleiro(t, "pedra") // a padrão, onde o jogador já está
	f.abreSegunda(t, "Cripta")

	if rec := f.pede(t, f.mestre, http.MethodPost, f.urlDaMesa()+"/tabuleiro/aba/"+taverna.ID+"/mostrar", ""); rec.Code != http.StatusOK {
		t.Fatalf("mostrar à mesa deu %d", rec.Code)
	}

	doJogador := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if !strings.Contains(doJogador, "O mestre trouxe a mesa para Taverna do Javali") {
		t.Fatal("a tela do jogador trocou de superfície sem nada dizer por quê")
	}
	// A SAÍDA existe mesmo sem ter para onde voltar: a aba ativa é um `<h2>` e
	// não se clica, então sem este botão a tira seria um modo que a pessoa não
	// tem como desfazer.
	if !strings.Contains(doJogador, "Continuar aqui") {
		t.Error("a tira ficou sem gesto que a desfaça para quem já estava na aba")
	}
	if strings.Contains(doJogador, "Voltar para") {
		t.Error("a tira ofereceu voltar para a aba em que a pessoa já está")
	}
}

// O EMPURRÃO DE SUPERFÍCIE é UMA VEZ POR PUXÃO, e é ele que separa o empurrão da
// trava (decisão do dono).
//
// A superfície é sinal do NAVEGADOR, então o servidor só a alcança escrevendo
// nela pelo stream. Escrever a cada quadro seria uma trava disfarçada: a pessoa
// que tenta voltar para a Mesa é devolvida ao mapa 200ms depois, para sempre, e
// conclui que o botão está quebrado — sem erro em lugar nenhum.
//
// Mede o FIO, e não o estado: o que este caso prende é quantas vezes o remendo
// de sinal SAI, que é a coisa que a pessoa sente.
func TestASuperficieEEmpurradaUmaVezPorPuxao(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")
	cripta := f.abreSegunda(t, "Cripta")
	f.pede(t, f.mestre, http.MethodPost, f.urlDaMesa()+"/tabuleiro/aba/"+cripta.ID+"/mostrar", "")

	fio := httptest.NewRecorder()
	sse := datastar.NewSSE(fio, httptest.NewRequest(http.MethodGet, "/", nil))

	// O primeiro quadro empurra; os dois seguintes, não.
	empurrado := empurraParaOMapa(f.s, sse, f.sessionID, f.jogador, 0)
	empurrado = empurraParaOMapa(f.s, sse, f.sessionID, f.jogador, empurrado)
	empurraParaOMapa(f.s, sse, f.sessionID, f.jogador, empurrado)

	if n := strings.Count(fio.Body.String(), superficieDoTabuleiro); n != 1 {
		t.Fatalf("o stream empurrou a superfície %d vezes num puxão só: é uma trava, não um empurrão", n)
	}
	if empurrado == 0 {
		t.Error("o empurrão não ficou registrado na conexão, então o quadro seguinte o repetiria")
	}

	// E DEPOIS de a pessoa escolher, não há mais o que empurrar.
	f.pede(t, f.jogador, http.MethodPost, f.urlDaMesa()+"/tabuleiro/aba/"+cripta.ID, "")
	segundoFio := httptest.NewRecorder()
	empurraParaOMapa(f.s, datastar.NewSSE(segundoFio, httptest.NewRequest(http.MethodGet, "/", nil)), f.sessionID, f.jogador, 0)
	if strings.Contains(segundoFio.Body.String(), superficieDoTabuleiro) {
		t.Error("uma conexão nova levou o empurrão de um puxão que a pessoa já tinha consumido")
	}
}
