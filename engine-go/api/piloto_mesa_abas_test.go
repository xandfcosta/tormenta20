package api

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"t20engine/tabuleiro"
)

// Os guardas das ABAS DE TABULEIRO (ALE-205).
//
// A regra que eles prendem não é "existe um segundo tabuleiro" — isso o store
// prova sozinho. É a separação que a issue existe para criar: **o que o mestre
// olha e o que a mesa vê deixaram de ser a mesma coisa**, e cada pessoa escolhe.
// Toda a família de defeito desta issue mora aí: um gesto de uma pessoa mexendo
// na tela das outras, ou um comando pousando na cena errada.

// abreSegunda põe outra cena na mesa e devolve as duas, na ordem de abertura.
func (f pilotoFixture) abreSegunda(t *testing.T, nome string) *tabuleiro.BoardState {
	t.Helper()
	b, err := f.s.boards.Open(context.Background(), f.sessionID, nome, "pedra")
	if err != nil {
		t.Fatalf("abrir %q: %v", nome, err)
	}
	return b
}

// TestNoSceneCommandUsesTheDefaultTab — a varredura da convenção (ALE-205).
//
// A convenção que esta issue criou é uma frase: **o comando age na aba de quem
// clicou**, e é ela que dispensou pôr o id do tabuleiro em vinte rotas. O
// compilador obriga a passar o parâmetro; ele não obriga a passar o CERTO — um
// `aAbaPadrao` num arquivo do piloto compila, roda, e pinta a taverna que a mesa
// está vendo enquanto o mestre olha a cripta. Sem estourar nada.
//
// Por isso ele varre em vez de conferir um caso: uma revisão nomeia um arquivo,
// e a correção é TODO arquivo com a mesma forma. A `aAbaPadrao` é da tela ANTIGA
// (`board_commands.go`, `board_rules.go`), que não tem abas, e some junto com ela.
func TestNoSceneCommandUsesTheDefaultTab(t *testing.T) {
	arquivos, err := filepath.Glob("piloto_*.go")
	if err != nil {
		t.Fatalf("listar os arquivos do piloto: %v", err)
	}
	// CONTROLE: sem ele, um glob que não casa com nada passaria VERDE — o teste
	// diria "nenhum arquivo viola" sobre uma varredura que não visitou ninguém,
	// que é a forma de silêncio que esta casa já pagou caro.
	if len(arquivos) < 20 {
		t.Fatalf("a varredura achou só %d arquivos do piloto: o padrão parou de casar", len(arquivos))
	}
	visitados := 0
	for _, caminho := range arquivos {
		if strings.HasSuffix(caminho, "_test.go") || strings.HasSuffix(caminho, "_templ.go") {
			continue
		}
		fonte, err := os.ReadFile(caminho)
		if err != nil {
			t.Fatalf("ler %s: %v", caminho, err)
		}
		visitados++
		if strings.Contains(string(fonte), "aAbaPadrao") {
			t.Errorf("%s usa aAbaPadrao: o comando do piloto age na aba de QUEM CLICOU (c.TabuleiroID), "+
				"e a padrão é da tela antiga — este gesto mexeria na cena que outra pessoa está olhando", caminho)
		}
	}
	if visitados < 20 {
		t.Fatalf("a varredura leu só %d arquivos de produção do piloto", visitados)
	}
}

// A BARRA só nasce quando há o que trocar.
//
// Com uma cena aberta ela seria uma ficha só flutuando sobre o mapa — enfeite
// ocupando o que a ALE-203 acabou de ganhar de altura. E a ATIVA é o `<h2>` da
// região: um `<h2>` por aba faria o leitor de tela anunciar três títulos para
// uma região que desenha uma cena.
func TestTheTabBarIsOnlyBornWithTwoScenes(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")

	uma := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if strings.Contains(uma, "tabuleiro-aba") {
		t.Error("com uma cena aberta a barra de abas apareceu — é ficha só, sobre o mapa")
	}

	f.abreSegunda(t, "Cripta")
	duas := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()

	if !strings.Contains(duas, "Ver o tabuleiro Cripta") {
		t.Fatal("com duas cenas abertas não há como chegar à segunda")
	}
	if n := strings.Count(duas, `class="tabuleiro-aba tabuleiro-aba-ativa"`); n != 1 {
		t.Errorf("%d abas ativas na barra, esperado exatamente 1", n)
	}
	// A ativa é o cabeçalho, e as outras são botões: UM `<h2>` na barra inteira.
	//
	// A primeira versão desta asserção contava `<h2` na PÁGINA e esperava 1 — a
	// Mesa tem quinze, uma por região, e o que ela media era a página e não a
	// barra. Instrumento que responde outra pergunta.
	if n := strings.Count(duas, `<h2 class="tabuleiro-aba`); n != 1 {
		t.Errorf("a barra tem %d abas como cabeçalho, esperado 1 (as outras são botões)", n)
	}
}

// TROCAR DE ABA É DE QUEM CLICOU, e de mais ninguém.
//
// É o coração da issue: o jogador que desceu na cripta abre a aba da cripta
// porque QUER, e o mestre continua montando a taverna. Uma troca que viajasse
// para a mesa faria cada clique de um jogador arrastar a tela dos outros cinco —
// e no meio de um combate ninguém entenderia por que o mapa mudou.
func TestSwitchingTabsChangesOnlyTheScreenOfWhoClicked(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra") // "Taverna do Javali", a primeira
	cripta := f.abreSegunda(t, "Cripta")

	rec := f.pede(t, f.jogador, http.MethodPost, f.urlDaMesa()+"/tabuleiro/aba/"+cripta.ID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("o jogador não conseguiu trocar de aba: %d", rec.Code)
	}

	doJogador := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if !strings.Contains(doJogador, "Ver o tabuleiro Taverna do Javali") {
		t.Error("o jogador trocou para a cripta e a taverna deixou de ser alcançável")
	}
	if !strings.Contains(doJogador, `aria-current="true"`) || !strings.Contains(doJogador, "Cripta</h2>") {
		t.Error("a tela do jogador não seguiu a aba que ele escolheu")
	}

	doMestre := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if !strings.Contains(doMestre, "Taverna do Javali</h2>") {
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
	f := novoPiloto(t)
	taverna := f.abreTabuleiro(t, "pedra")
	cripta := f.abreSegunda(t, "Cripta")
	ctx := context.Background()

	f.pede(t, f.mestre, http.MethodPost, f.urlDaMesa()+"/tabuleiro/aba/"+cripta.ID, "")
	rec := f.pede(t, f.mestre, http.MethodPost, f.urlDaMesa()+"/tabuleiro/terreno/dificil/2/3/ate/2/3", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("pintar deu %d", rec.Code)
	}

	if n := len(f.s.boards.Get(ctx, f.sessionID, cripta.ID).Difficult); n != 1 {
		t.Errorf("a cripta — a aba aberta — recebeu %d casas de terreno, esperado 1", n)
	}
	if n := len(f.s.boards.Get(ctx, f.sessionID, taverna.ID).Difficult); n != 0 {
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
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")
	cripta := f.abreSegunda(t, "Cripta")
	f.pede(t, f.jogador, http.MethodPost, f.urlDaMesa()+"/tabuleiro/aba/"+cripta.ID, "")

	// O mestre entra na cripta e a encerra.
	f.pede(t, f.mestre, http.MethodPost, f.urlDaMesa()+"/tabuleiro/aba/"+cripta.ID, "")
	if rec := f.pede(t, f.mestre, http.MethodPost, f.urlDaMesa()+"/tabuleiro/encerrar", ""); rec.Code != http.StatusOK {
		t.Fatalf("encerrar deu %d", rec.Code)
	}

	doJogador := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()
	// A frase é a DO JOGADOR, copiada do `.templ`. Duas correções aqui, e as
	// duas eram o mesmo erro: eu tinha escrito uma paráfrase minha ("não há
	// tabuleiro"), que não existe na página; e depois a frase do MESTRE, que o
	// jogador nunca lê — as duas passariam SEMPRE, verdes sobre o defeito exato
	// que este guarda nomeia.
	if strings.Contains(doJogador, "O mestre ainda não abriu um tabuleiro") {
		t.Fatal("o jogador ficou sem mapa porque a aba dele foi fechada, com outra cena aberta na mesa")
	}
	if !strings.Contains(doJogador, "Taverna do Javali") {
		t.Error("o jogador não caiu na aba padrão depois de a dele ser fechada")
	}
}

// O NOME NÃO ATRAVESSA A CORTINA, nem na barra de abas.
//
// A decisão do dono foi que a aba sob cortina APARECE para o jogador — sumir e
// voltar trocaria a aba debaixo do dedo de quem estava olhando. O preço disso é
// este guarda: a ficha existe e não pode dizer COMO A CENA SE CHAMA. "Cripta do
// Rei Caolho" no HTML de quem não pode saber que há uma cripta é o vazamento que
// não aparece na tela — só no ver-código-fonte.
func TestATabUnderTheCurtainDoesNotTellThePlayerTheSceneName(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")
	emboscada := f.abreSegunda(t, "Cripta do Rei Caolho")
	f.pede(t, f.mestre, http.MethodPost, f.urlDaMesa()+"/tabuleiro/aba/"+emboscada.ID, "")
	if rec := f.pede(t, f.mestre, http.MethodPost, f.urlDaMesa()+"/tabuleiro/cortina/fechar", ""); rec.Code != http.StatusOK {
		t.Fatalf("fechar a cortina deu %d", rec.Code)
	}

	doJogador := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()

	if strings.Contains(doJogador, "Rei Caolho") {
		t.Fatal("o nome da cena sob cortina saiu no HTML do jogador")
	}
	// E a aba continua LÁ: ela é como ele sabe que vem cena, sem ver qual.
	if !strings.Contains(doJogador, "Cena 2") {
		t.Error("a aba sob cortina sumiu da barra do jogador em vez de se chamar pela posição")
	}
	// Para o mestre a cortina não é sobre ele: o nome continua na barra dele.
	doMestre := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if !strings.Contains(doMestre, "Rei Caolho") {
		t.Error("o mestre perdeu o nome da própria cena por causa da cortina dele")
	}
}
