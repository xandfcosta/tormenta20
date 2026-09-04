package api

import (
	"bufio"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"t20engine/aovivo"
	"t20engine/events"
	"testing"
)

func TestTheTableDoesNotLeakHiddenHp(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)

	corpo := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()

	if !strings.Contains(corpo, "Ogro cansado") {
		t.Fatal("o ogro sumiu da fila do jogador — ele deve ver QUEM está lá")
	}
	if strings.Contains(corpo, "130") {
		t.Errorf("os PV ocultos do mestre vazaram para o HTML do jogador")
	}
	// A flag sobrevive à redação de propósito: "sem barra" e "escondido" são
	// coisas diferentes, e a segunda é informação (ALE-210).
	if !strings.Contains(corpo, "PV ocultos pelo mestre") {
		t.Errorf("a linha oculta não DISSE que está oculta — vira 'sem vida' na tela")
	}
}

// A outra metade da ALE-210: fora de cena o jogador não recebe fila NENHUMA.
// Não desenhar seria UX; não mandar é a trava.
//
// Provado VERMELHO com o mesmo desvio do teste acima: sem cena o HTML passou a
// listar os dois combatentes que o mestre está montando às escondidas.
func TestOffSceneTheTableSendsNoTracker(t *testing.T) {
	f := novoPiloto(t)
	// Fila CHEIA e cena DESLIGADA: é o mestre montando a briga antes de começar.
	if _, err := f.s.Sessions().AddInitiativeEntry(f.sessionID, aovivo.InitiativeEntry{
		Label: "Chefe secreto", Initiative: 22, Type: "npc",
	}); err != nil {
		t.Fatalf("semear chefe: %v", err)
	}

	corpo := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()

	if strings.Contains(corpo, "Chefe secreto") {
		t.Error("a fila de fora de cena vazou para o jogador")
	}
	if !strings.Contains(corpo, "Fora de cena") {
		t.Error("a tela não explicou o vazio")
	}
	// O mestre, na MESMA página, continua vendo o que montou — sem esta metade
	// o teste passaria com um `redactForPlayers` aplicado a todo mundo.
	corpoDoMestre := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(corpoDoMestre, "Chefe secreto") {
		t.Error("o mestre perdeu a própria fila — a redação está pegando o papel errado")
	}
}

// A recusa tem de CHEGAR NA TELA, e é isto que o piloto ganha de graça sobre o
// socket: a ALE-213 deixou anotado que o cliente não escuta o `exception`, então
// lá um d20 fora da faixa some em silêncio.
//
// Provado VERMELHO devolvendo `http.Error` no lugar do patch de sinal: o corpo
// virou texto solto que o Datastar descarta, e a tela não muda.
func TestTheTableRefusesAD20OutsideTheRangeAndSaysSo(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)

	corpo := f.posta(t, f.jogador, f.tableUrl()+"/iniciativa", `{"d20":47}`)

	if !strings.HasPrefix(corpo, "event: datastar-patch-signals") {
		t.Fatalf("a recusa não saiu como evento do Datastar, saiu como:\n%s", corpo)
	}
	// A mensagem carrega o valor ofendido, que é a regra da casa para exceção.
	if !strings.Contains(corpo, "47") {
		t.Errorf("a recusa não disse qual valor foi recusado:\n%s", corpo)
	}
	if !strings.Contains(corpo, `\"erro\"`) && !strings.Contains(corpo, `"erro"`) {
		t.Errorf("a recusa não veio no sinal `erro`, então nada acende na tela:\n%s", corpo)
	}
}

// O caminho feliz, e a metade que importa é a SEGUNDA: o total é do servidor.
//
// O bônus do fixture é 3 (nível 8, perícia Iniciativa treinada em Destreza),
// então mandar d20=14 tem de gravar 17. Sem a perícia semeada o bônus seria
// zero e 14 == 14 — o teste passaria verde sobre uma tela que somou sozinha,
// que é exatamente o defeito que ele mira (a armadilha da ALE-213).
func TestTheTableRecordsInitiativeWithTheServerTotal(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)
	bonus, err := f.s.InitiativeBonus(context.Background(), f.charID)
	if err != nil {
		t.Fatalf("bônus: %v", err)
	}
	if bonus == 0 {
		t.Fatal("bônus zero: o fixture nasceu vácuo e este teste não provaria nada")
	}

	// O corpo é conferido, e não só o código: com a ordem trocada o servidor
	// devolve 200 com um erro DENTRO do sinal, e "deu 200" não é resposta.
	if resposta := f.posta(t, f.jogador, f.tableUrl()+"/iniciativa", `{"d20":14}`); !strings.Contains(resposta, `{"erro":""}`) {
		t.Fatalf("a escrita não foi aceita, respondeu:\n%s", resposta)
	}

	estado := f.s.Sessions().GetState(f.sessionID)
	for i := range estado.Initiative {
		e := &estado.Initiative[i]
		if e.CharacterID != nil && *e.CharacterID == f.charID {
			if querido := 14 + int(bonus); e.Initiative != querido {
				t.Fatalf("iniciativa gravada = %d, queria %d (14 + bônus %d)", e.Initiative, querido, bonus)
			}
			return
		}
	}
	t.Fatal("a linha do jogador não entrou na fila")
}

// A COMPRESSÃO do stream — o passo (a) da ordem combinada, e o ganho que domina
// todos os outros: medido neste piloto, 52.332 bytes crus de três remendos viram
// 2.513 em gzip e 1.827 em brotli. Ela é invisível na tela, então quem apagar o
// `WithCompression()` multiplica a banda por vinte sem nada parecer errado — é
// exatamente por isso que ela precisa de guarda.
//
// Servidor de verdade e `Accept-Encoding` escrito à mão de propósito: o
// `http.Client` do Go põe o cabeçalho sozinho e descomprime por baixo do pano,
// escondendo o `Content-Encoding` que este teste existe para ver.
func TestTheTableStreamCompresses(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)
	srv := httptest.NewServer(f.s.WebRouter())
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL+f.tableUrl()+"/stream", nil)
	if err != nil {
		t.Fatalf("montar pedido: %v", err)
	}
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Authorization", "Bearer "+f.token(t, f.jogador))
	// O stream não termina sozinho; o contexto é o que devolve o controle depois
	// do primeiro quadro.
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		t.Fatalf("abrir stream: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if got := resp.Header.Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, queria gzip — o stream saiu cru", got)
	}
	zr, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatalf("o corpo não é gzip: %v", err)
	}
	// PROCURA a fila entre os quadros em vez de assumi-la no primeiro, e isso
	// mudou com as REGIÕES (ALE-264): a carga fria manda um quadro por região —
	// cabeçalho, registrar, grupo, tabuleiro, fila, comandos —, então a fila é o
	// quinto. Ler só o primeiro afirmaria a ordem interna do render, que não é
	// promessa nenhuma; o que o teste quer saber é que ela CHEGA comprimida.
	//
	// Ler um buffer de tamanho fixo continua não servindo: um buffer curto corta
	// o fragmento no meio, e um longo bloquearia esperando quadros que só o
	// batimento traria.
	leitor := bufio.NewScanner(zr)
	leitor.Buffer(make([]byte, 0, 64*1024), 1<<20)
	var tudo strings.Builder
	var achouPatch, achouFila bool
	for leitor.Scan() && !achouFila {
		linha := leitor.Text()
		tudo.WriteString(linha)
		tudo.WriteString("\n")
		if strings.Contains(linha, "datastar-patch-elements") {
			achouPatch = true
		}
		if strings.Contains(linha, "Ogro cansado") {
			achouFila = true
		}
	}

	// As duas metades: veio no evento do Datastar E carrega a fila de verdade.
	// Só a primeira passaria verde com um stream que comprime silêncio.
	if !achouPatch {
		t.Errorf("nenhum quadro é um patch do Datastar:\n%.600s", tudo.String())
	}
	if !achouFila {
		t.Errorf("a fila não chegou em quadro nenhum:\n%.600s", tudo.String())
	}
}

// O aviso do store — o passo (b). O que este teste afirma é a INVARIANTE que
// torna o aviso confiável: `apply` é o funil das treze mutações da fila, então
// nenhuma delas pode escapar sem virar notícia.
//
// Desde a ALE-279 a invariante é mais forte, e não é este teste que a segura: o
// `apply` recebe o EVENTO por parâmetro, então uma mutação sem notícia não
// compila. O que sobrou aqui para medir é que a notícia certa chega a quem
// escuta — que abrir a cena publique `SceneStarted`, e não um sino genérico que
// serviria igualmente para o encerramento.
func TestTheTableTellsSubscribersOnEveryMutation(t *testing.T) {
	f := novoPiloto(t)
	sub, parar := f.s.Bus().Subscribe(events.OfSession(f.sessionID))

	if _, err := f.s.Sessions().StartScene(f.sessionID); err != nil {
		t.Fatalf("iniciar cena: %v", err)
	}
	select {
	case ev := <-sub.C:
		if _, ok := ev.(events.SceneStarted); !ok {
			t.Fatalf("iniciar a cena publicou %T", ev)
		}
	default:
		t.Fatal("a mutação passou sem avisar quem escuta")
	}

	// Baixar a assinatura tem de PARAR a entrega: sem isto cada aba fechada deixa
	// um canal para sempre, e o `Publish` passa a percorrer uma lista que só
	// cresce escrevendo em canais que ninguém lê.
	parar()
	if _, err := f.s.Sessions().EndScene(f.sessionID); err != nil {
		t.Fatalf("encerrar cena: %v", err)
	}
	select {
	case ev := <-sub.C:
		t.Fatalf("o ouvinte baixado continuou recebendo %T — a lista vaza", ev)
	default:
	}
}
