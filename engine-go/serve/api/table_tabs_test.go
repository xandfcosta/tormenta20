package api

import (
	"context"
	"net/http"
	"strings"
	"t20engine/domain/board"
	"testing"
)

func (f sceneFixture) openSecond(t *testing.T, name string) *board.BoardState {
	t.Helper()
	b, err := f.s.tableHost().Boards().Open(context.Background(), f.sessionID, name, "stone")
	if err != nil {
		t.Fatalf("abrir %q: %v", name, err)
	}
	return b
}

// A BARRA só nasce quando há o que trocar.
//
// Com uma cena aberta ela seria uma ficha só flutuando sobre o mapa. E a ATIVA é
// o `<h2>` da região: um `<h2>` por aba faria o leitor de tela anunciar três
// títulos para uma região que desenha uma cena.
func TestTheTabBarIsOnlyBornWithTwoScenes(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	one := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if strings.Contains(one, "board-tab") {
		t.Error("com uma cena aberta a barra de abas apareceu — é ficha só, sobre o mapa")
	}

	f.openSecond(t, "Cripta")
	two := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()

	if !strings.Contains(two, "Ver o tabuleiro Cripta") {
		t.Fatal("com duas cenas abertas não há como chegar à segunda")
	}
	if n := strings.Count(two, `class="board-tab board-tab-active"`); n != 1 {
		t.Errorf("%d abas ativas na barra, esperado exatamente 1", n)
	}
	// A ativa é o cabeçalho, e as outras são botões: UM `<h2>` na barra inteira.
	//
	// Contar `<h2` na PÁGINA responde outra pergunta: a Mesa tem quinze, uma por
	// região. O seletor tem de ser o da BARRA.
	if n := strings.Count(two, `<h2 class="board-tab`); n != 1 {
		t.Errorf("a barra tem %d abas como cabeçalho, esperado 1 (as outras são botões)", n)
	}
}

// TROCAR DE ABA É DE QUEM CLICOU, e de mais ninguém.
//
// O jogador que desceu na cripta abre a aba da cripta porque QUER, e o mestre
// continua montando a taverna. Uma troca que viajasse
// para a mesa faria cada clique de um jogador arrastar a tela dos outros cinco —
// e no meio de um combate ninguém entenderia por que o mapa mudou.
func TestSwitchingTabsChangesOnlyTheScreenOfWhoClicked(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone") // "Taverna do Javali", a primeira
	crypt := f.openSecond(t, "Cripta")

	rec := f.pede(t, f.player, http.MethodPost, f.tableUrl()+"/tabuleiro/aba/"+crypt.ID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("o jogador não conseguiu trocar de aba: %d", rec.Code)
	}

	forPlayer := f.pede(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(forPlayer, "Ver o tabuleiro Taverna do Javali") {
		t.Error("o jogador trocou para a cripta e a taverna deixou de ser alcançável")
	}
	if !strings.Contains(forPlayer, `aria-current="true"`) || !strings.Contains(forPlayer, "Cripta</h2>") {
		t.Error("a tela do jogador não seguiu a aba que ele escolheu")
	}

	forGM := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(forGM, "Taverna do Javali</h2>") {
		t.Error("o clique do jogador arrastou a tela do mestre junto")
	}
}

// O COMANDO AGE NA ABA DE QUEM CLICOU.
//
// É a afirmação que dispensou pôr o id do tabuleiro em vinte rotas: **não se
// pinta um tabuleiro que não se está olhando.** Sem ela, cada gesto do mestre
// cairia na aba padrão — ele pintaria a cripta e o terreno apareceria na taverna
// que a mesa está vendo, que é a emboscada vazando por outro caminho.
func TestTheGestureLandsOnTheTabTheGmIsLookingAt(t *testing.T) {
	f := newSceneFixture(t)
	tavern := f.seedOpenBoard(t, "stone")
	crypt := f.openSecond(t, "Cripta")
	ctx := context.Background()

	f.pede(t, f.gm, http.MethodPost, f.tableUrl()+"/tabuleiro/aba/"+crypt.ID, "")
	rec := f.pede(t, f.gm, http.MethodPost, f.tableUrl()+"/tabuleiro/terreno", stroke("dificil", 2, 3, 2, 3))
	if rec.Code != http.StatusOK {
		t.Fatalf("pintar deu %d", rec.Code)
	}

	if n := len(f.s.tableHost().Boards().Get(ctx, f.sessionID, crypt.ID).Difficult); n != 1 {
		t.Errorf("a cripta — a aba aberta — recebeu %d casas de terreno, esperado 1", n)
	}
	if n := len(f.s.tableHost().Boards().Get(ctx, f.sessionID, tavern.ID).Difficult); n != 0 {
		t.Errorf("a taverna, que ninguém estava olhando, recebeu %d casas de terreno", n)
	}
}

// A ABA FECHADA DEVOLVE A PESSOA À PADRÃO, e não a uma tela morta.
//
// O gesto que causa isto é de OUTRA pessoa: o mestre encerra a cripta enquanto
// um jogador a olha. Sem a conferência a cada leitura, a tela dele diria "esta
// sessão não tem tabuleiro" com a taverna aberta na mesa ao lado — e ele não
// teria como ligar uma coisa à outra.
func TestClosingATabSendsWhoeverWasOnItBackToTheDefault(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	crypt := f.openSecond(t, "Cripta")
	f.pede(t, f.player, http.MethodPost, f.tableUrl()+"/tabuleiro/aba/"+crypt.ID, "")

	// O mestre entra na cripta e a encerra.
	f.pede(t, f.gm, http.MethodPost, f.tableUrl()+"/tabuleiro/aba/"+crypt.ID, "")
	if rec := f.pede(t, f.gm, http.MethodPost, f.tableUrl()+"/tabuleiro/encerrar", ""); rec.Code != http.StatusOK {
		t.Fatalf("encerrar deu %d", rec.Code)
	}

	forPlayer := f.pede(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	// A frase é a DO JOGADOR, copiada do `.templ`: uma paráfrase, ou a frase do
	// MESTRE (que o jogador nunca lê), passa SEMPRE — verde sobre o defeito exato
	// que este guarda nomeia.
	if strings.Contains(forPlayer, "O mestre ainda não abriu um tabuleiro") {
		t.Fatal("o jogador ficou sem mapa porque a aba dele foi fechada, com outra cena aberta na mesa")
	}
	if !strings.Contains(forPlayer, "Taverna do Javali") {
		t.Error("o jogador não caiu na aba padrão depois de a dele ser fechada")
	}
}

// O NOME NÃO ATRAVESSA A CORTINA, nem na barra de abas.
//
// A aba sob cortina APARECE para o jogador (decisão do dono) — sumir e voltar
// trocaria a aba debaixo do dedo de quem estava olhando. O preço é este guarda:
// a ficha existe e não pode dizer COMO A CENA SE CHAMA. "Cripta do
// Rei Caolho" no HTML de quem não pode saber que há uma cripta é o vazamento que
// não aparece na tela — só no ver-código-fonte.
func TestATabUnderTheCurtainDoesNotTellThePlayerTheSceneName(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	ambush := f.openSecond(t, "Cripta do Rei Caolho")
	f.pede(t, f.gm, http.MethodPost, f.tableUrl()+"/tabuleiro/aba/"+ambush.ID, "")
	if rec := f.pede(t, f.gm, http.MethodPost, f.tableUrl()+"/tabuleiro/cortina/fechar", ""); rec.Code != http.StatusOK {
		t.Fatalf("fechar a cortina deu %d", rec.Code)
	}

	forPlayer := f.pede(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()

	if strings.Contains(forPlayer, "Rei Caolho") {
		t.Fatal("o nome da cena sob cortina saiu no HTML do jogador")
	}
	// E a aba continua LÁ: ela é como ele sabe que vem cena, sem ver qual.
	if !strings.Contains(forPlayer, "Cena 2") {
		t.Error("a aba sob cortina sumiu da barra do jogador em vez de se chamar pela posição")
	}
	// Para o mestre a cortina não é sobre ele: o nome continua na barra dele.
	forGM := f.pede(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(forGM, "Rei Caolho") {
		t.Error("o mestre perdeu o nome da própria cena por causa da cortina dele")
	}
}
