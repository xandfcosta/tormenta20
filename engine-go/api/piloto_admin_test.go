package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"t20engine/web/ui"
	"testing"
	"time"
)

// As RÉPLICAS dos dez testes de integração da tela de administração (ALE-219).
//
// Este arquivo é uma MEDIÇÃO, não só um guarda. A pergunta que a issue deixou
// parcial era "quanto custa perder a faixa de integração do vitest", e a
// resposta com um número exigia pegar dez testes reais e ver, um a um, onde
// cada garantia passaria a morar.
//
// O saldo está no comentário de cada bloco. O que a contagem escondia é o mais
// importante: quatro dos dez guardavam o `expiryLabel`, e a primeira versão
// desta tela renderizava o ISO cru — a migração não perdeu o teste, perdeu a
// REGRA, e o teste teria pegado.

// ── os quatro do prazo do convite ────────────────────────────────────────────
// Eram integração (montavam o painel) e viram UNITÁRIO: a regra é uma função
// pura, e montar um painel para afirmar arredondamento sempre foi mais caro do
// que a garantia pedia. Estes quatro ficam MAIS baratos depois da migração.

func TestExpiresIn(t *testing.T) {
	agora := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	em := func(d time.Duration) string { return agora.Add(d).Format(time.RFC3339) }

	casos := []struct {
		nome  string
		prazo string
		quer  string
	}{
		// Arredonda: sete dias menos alguns segundos ainda são 7, não 6.
		{"quase sete dias ainda são 7", em(7*24*time.Hour - 3*time.Second), "7 dias"},
		{"singular quando falta um dia", em(24 * time.Hour), "1 dia"},
		// Abaixo de um dia o dono precisa da escala de HORAS: "0 dias" não diz
		// se dá tempo de mandar a mensagem.
		{"menos de um dia vira horas", em(5 * time.Hour), "5 horas"},
		{"prestes a vencer não vira 0 horas", em(time.Minute), "1 hora"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := expiraEm(c.prazo, agora); got != c.quer {
				t.Errorf("expiraEm = %q, queria %q", got, c.quer)
			}
		})
	}
}

// ── os três das frases da linha ──────────────────────────────────────────────
// "mostra o que cada conta tem", "marca quem administra" e as duas metades da
// frase do custo. Eram integração; viram unitário pelo mesmo motivo: são
// pluralização, e pluralização é função pura.

func TestHoldingsAndHowTheyRead(t *testing.T) {
	if got := posses(false, 2, 1); got != "2 campanhas · 1 ficha" {
		t.Errorf("posses = %q", got)
	}
	if got := posses(true, 1, 3); got != "admin · 1 campanha · 3 fichas" {
		t.Errorf("posses de admin = %q", got)
	}
}

// O aviso tem de dizer o preço DESTA conta: um texto genérico não distingue
// apagar uma conta vazia de apagar a do jogador que mestra duas campanhas.
func TestTheDeleteCostNamesThePriceOfTheAccount(t *testing.T) {
	comCampanhas := custoDeApagar(2, 3)
	if !strings.Contains(comCampanhas, "3 fichas") || !strings.Contains(comCampanhas, "2 campanhas passam") {
		t.Errorf("custo = %q — precisa dizer o que se perde E para onde vão as campanhas", comCampanhas)
	}
	semCampanhas := custoDeApagar(0, 1)
	if !strings.Contains(semCampanhas, "Não há campanhas para transferir") {
		t.Errorf("sem campanhas o aviso não pode prometer transferência: %q", semCampanhas)
	}
}

// ── o que virou teste de RENDER ──────────────────────────────────────────────
// "não oferece apagar a própria conta" não é regra pura: é o template decidindo
// o que desenhar. Renderizar o fragmento e olhar o HTML é o equivalente mais
// barato — não precisa de navegador, e afirma o mesmo resultado que o teste de
// integração afirmava.

func TestThePanelDoesNotOfferDeletingYourOwnAccount(t *testing.T) {
	view := adminView{Jogadores: []adminJogador{
		{ID: 1, Nome: "Dono", Email: "dono@t.com", Posses: "admin", Custo: "-", EhEu: true},
		{ID: 2, Nome: "Outro", Email: "outro@t.com", Posses: "-", Custo: "-", EhEu: false},
	}}

	html, err := ui.RenderFragment(t.Context(), painelJogadores(view))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	texto := html

	if strings.Contains(texto, "Apagar a conta de Dono") {
		t.Error("o painel ofereceu apagar a PRÓPRIA conta — o servidor recusaria e o dono levaria um erro")
	}
	if !strings.Contains(texto, "Apagar a conta de Outro") {
		t.Error("o painel deixou de oferecer apagar as outras contas")
	}
}

// O primeiro clique NÃO pode ser irreversível, e no Datastar isso é uma
// propriedade do MARCADOR: o botão da linha só abre o diálogo, e quem posta é o
// botão de dentro dele. Afirmar isso aqui é barato; o e2e irmão prova o
// comportamento no navegador.
func TestTheRowButtonOpensTheDialogInsteadOfDeleting(t *testing.T) {
	view := adminView{Jogadores: []adminJogador{{ID: 2, Nome: "Outro", EhEu: false}}}
	linha, err := ui.RenderFragment(t.Context(), painelJogadores(view))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(linha, "@post") {
		t.Error("o botão da linha posta direto — o primeiro clique virou irreversível")
	}
	if !strings.Contains(linha, "showModal()") {
		t.Error("o botão da linha não abre o diálogo")
	}

	dialogo, err := ui.RenderFragment(t.Context(), dialogoConfirmar())
	if err != nil {
		t.Fatalf("render do diálogo: %v", err)
	}
	if !strings.Contains(dialogo, "@post") {
		t.Error("quem apaga é o botão do diálogo, e ele não posta")
	}
}

// ── o link de redefinição (ALE-242) ──────────────────────────────────────────

// Redefinir vale para TODA conta, inclusive a de quem está olhando — e é aí que
// ele se separa do Apagar, que tem a guarda do `EhEu`. O admin que esqueceu a
// própria senha usa esta mesma porta; sem isto ele fica de fora da única saída
// que o app oferece.
func TestResettingWorksForYourOwnAccountToo(t *testing.T) {
	view := adminView{Jogadores: []adminJogador{
		{ID: 1, Nome: "Dono", Email: "dono@t.com", EhEu: true},
		{ID: 2, Nome: "Outro", Email: "outro@t.com", EhEu: false},
	}}

	html, err := ui.RenderFragment(t.Context(), painelJogadores(view))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, quem := range []string{"Dono", "Outro"} {
		if !strings.Contains(html, "Redefinir a senha de "+quem) {
			t.Errorf("o painel não oferece redefinir a senha de %q", quem)
		}
	}
	// E o contraste com o Apagar continua valendo, senão este teste passaria
	// num painel que perdeu a guarda do `EhEu`.
	if strings.Contains(html, "Apagar a conta de Dono") {
		t.Error("o painel voltou a oferecer apagar a PRÓPRIA conta")
	}
}

// O prazo é 24h e não os 7 dias do convite, e a diferença é de RISCO: o convite
// abre uma conta que ainda não existe; este abre uma que já existe e tem fichas
// dentro. Um link esquecido numa conversa vale mais para um estranho.
func TestTheResetLinkLastsTwentyFourHours(t *testing.T) {
	s := newTestServer(t)
	dono := seedUser(t, s, "dono@t20.local")

	antes := time.Now()
	reset, err := s.mintPasswordReset(context.Background(), dono, dono)
	if err != nil {
		t.Fatalf("cunhar: %v", err)
	}
	expira, err := time.Parse(time.RFC3339, reset.Expiresat)
	if err != nil {
		t.Fatalf("expiresat %q não é RFC3339: %v", reset.Expiresat, err)
	}
	vida := expira.Sub(antes)
	if vida < 23*time.Hour || vida > 25*time.Hour {
		t.Errorf("o link vale %v, queria ~24h — o prazo do CONVITE é 7 dias e são coisas diferentes", vida)
	}
}

// Conta inexistente devolve um erro NOMEADO, e não um erro qualquer: é o que
// deixa quem chama escolher a resposta. A rota JSON traduz para 404, a cena do
// piloto para um aviso na tela — e nenhuma das duas repete a consulta.
func TestMintingForAMissingAccountSaysItIsMissing(t *testing.T) {
	s := newTestServer(t)
	dono := seedUser(t, s, "dono@t20.local")

	_, err := s.mintPasswordReset(context.Background(), 999999, dono)
	if !errors.Is(err, errUserNotFound) {
		t.Errorf("erro = %v, queria errUsuarioInexistente", err)
	}
}

// O botão da linha ABRE o diálogo; quem posta é o de dentro. Mesma propriedade
// do Apagar e pela mesma razão: gerar um link é um efeito no banco, e o
// primeiro clique não deve produzi-lo.
//
// E ele LIMPA o `#reset-link` ao abrir. Isso não é zelo: sem limpar, gerar o
// link da Ana, fechar, e abrir a caixa da Bia mostraria o link da ANA sob o
// nome da BIA — link de redefinição entregue à pessoa errada. O e2e irmão prova
// isso no navegador; aqui se afirma que a limpeza está no marcador.
func TestTheResetButtonOpensTheDialogAndClearsThePreviousLink(t *testing.T) {
	view := adminView{Jogadores: []adminJogador{{ID: 2, Nome: "Outro", EhEu: false}}}
	linha, err := ui.RenderFragment(t.Context(), painelJogadores(view))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(linha, "@post") {
		t.Error("o botão da linha posta direto — cunhar virou efeito do primeiro clique")
	}
	if !strings.Contains(linha, "reset-link") {
		t.Error("abrir não limpa o link anterior — ele apareceria sob o nome errado")
	}

	dialogo, err := ui.RenderFragment(t.Context(), dialogoRedefinir())
	if err != nil {
		t.Fatalf("render do diálogo: %v", err)
	}
	if !strings.Contains(dialogo, "@post") {
		t.Error("quem cunha é o botão do diálogo, e ele não posta")
	}
}

// O remendo carrega o CAMINHO e nunca a URL inteira: quem prefixa a origem é o
// navegador. Com o `r.Host`, o link nasce apontando para a porta da API porque
// o proxy do Vite reescreve o `Host` em desenvolvimento — e link de redefinição
// existe para ser MANDADO, então host errado é link morto.
func TestTheResetPatchCarriesNoOrigin(t *testing.T) {
	html, err := ui.RenderFragment(t.Context(), resetGerado("/redefinir-senha?token=abc"))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(html, "http://") || strings.Contains(html, "https://") {
		t.Error("o remendo trouxe origem — o link nasceria apontando para a máquina errada")
	}
	if !strings.Contains(html, "location.origin") {
		t.Error("ninguém prefixa a origem no navegador — o campo ficaria com um caminho solto")
	}
}

// Cunhar convite pela ADMINISTRAÇÃO remenda DUAS coisas: o link e o painel.
//
// O segundo é a diferença entre esta rota e a do Hub, e ele não é enfeite:
// aqui a lista de convites está a três centímetros do botão, e sem remendá-la a
// tela diz "Convites abertos (0)" logo depois de a pessoa abrir um. No Hub não
// existe essa lista, e por isso lá basta o link.
//
// O guarda é em Go e não no navegador de propósito: cunhar grava uma linha que
// a TELA não sabe revogar, então um e2e desta garantia deixaria lixo permanente
// no banco de desenvolvimento a cada corrida — que é a família de problema da
// ALE-238. Aqui o banco é descartável.
func TestMintingFromAdminPatchesThePanelToo(t *testing.T) {
	s := newTestServer(t, "chefe@t20.local")
	chefe := seedUser(t, s, "chefe@t20.local")

	rec := pedeNoPiloto(t, s, chefe, http.MethodPost, "/admin/convites")
	corpo := rec.Body.String()

	if !strings.Contains(corpo, "convite-url") {
		t.Errorf("o remendo do link não veio:\n%s", corpo)
	}
	if !strings.Contains(corpo, `id="painel-convites"`) {
		t.Error("o painel de convites não foi remendado — a contagem ao lado do botão fica velha")
	}
	if !strings.Contains(corpo, "Convites abertos (1)") {
		t.Errorf("o painel voltou sem contar o convite recém-cunhado:\n%s", corpo)
	}
}

// A trava é do SERVIDOR: quem não administra não cunha, mesmo postando na mão.
// A tela nem oferece o botão, mas isso é UX — a fronteira é aqui.
func TestANonAdminDoesNotReachTheInviteRoute(t *testing.T) {
	s := newTestServer(t, "chefe@t20.local")
	seedUser(t, s, "chefe@t20.local")
	qualquerUm := seedUser(t, s, "outro@t20.local")

	rec := pedeNoPiloto(t, s, qualquerUm, http.MethodPost, "/admin/convites")
	if rec.Code == http.StatusOK {
		t.Errorf("a rota respondeu %d para quem não é admin", rec.Code)
	}
}

// pedeNoPiloto bate no roteador do piloto, que é OUTRO que o `Router()` da API.
func pedeNoPiloto(t *testing.T, s *Server, userID int64, metodo, caminho string) *httptest.ResponseRecorder {
	t.Helper()
	u, err := s.queries.GetUserByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("usuário: %v", err)
	}
	token, err := s.signToken(u)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	req := httptest.NewRequest(metodo, caminho, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	s.WebRouter().ServeHTTP(rec, req)
	return rec
}
