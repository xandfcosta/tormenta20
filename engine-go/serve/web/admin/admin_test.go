package admin

import (
	"strings"
	"t20engine/serve/web/ui"
	"testing"
	"time"
)

// OS GUARDAS DA REGRA DA TELA DE ADMINISTRAÇÃO.
//
// Aqui moram só os que exercitam função PURA — a frase de belongings, o custo de
// apagar, o prazo do convite — e os painéis desenhados a partir de uma view
// montada à mão. Os que precisam do servidor de verdade estão no `api`.

func TestExpiresIn(t *testing.T) {
	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	em := func(d time.Duration) string { return now.Add(d).Format(time.RFC3339) }

	cases := []struct {
		name     string
		deadline string
		want     string
	}{
		// Arredonda: sete dias menos alguns segundos ainda são 7, não 6.
		{"quase sete dias ainda são 7", em(7*24*time.Hour - 3*time.Second), "7 dias"},
		{"singular quando falta um dia", em(24 * time.Hour), "1 dia"},
		// Abaixo de um dia o dono precisa da escala de HORAS: "0 dias" não diz
		// se dá tempo de mandar a mensagem.
		{"menos de um dia vira horas", em(5 * time.Hour), "5 horas"},
		{"prestes a vencer não vira 0 horas", em(time.Minute), "1 hora"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := expiryLabel(c.deadline, now); got != c.want {
				t.Errorf("expiryLabel = %q, queria %q", got, c.want)
			}
		})
	}
}

// ── os três das frases da linha ──────────────────────────────────────────────
// "mostra o que cada conta tem", "marca quem administra" e as duas metades da
// frase do custo. Eram integração; viram unitário pelo mesmo motivo: são
// pluralização, e pluralização é função pura.

func TestHoldingsAndHowTheyRead(t *testing.T) {
	if got := belongings(false, 2, 1); got != "2 campanhas · 1 ficha" {
		t.Errorf("belongings = %q", got)
	}
	if got := belongings(true, 1, 3); got != "admin · 1 campanha · 3 fichas" {
		t.Errorf("belongings de admin = %q", got)
	}
}

// O aviso tem de dizer o preço DESTA conta: um texto genérico não distingue
// apagar uma conta vazia de apagar a do jogador que mestra duas campanhas.
func TestTheDeleteCostNamesThePriceOfTheAccount(t *testing.T) {
	withCampaigns := deletionCost(2, 3)
	if !strings.Contains(withCampaigns, "3 fichas") || !strings.Contains(withCampaigns, "2 campanhas passam") {
		t.Errorf("custo = %q — precisa dizer o que se perde E para onde vão as campanhas", withCampaigns)
	}
	noCampaigns := deletionCost(0, 1)
	if !strings.Contains(noCampaigns, "Não há campanhas para transferir") {
		t.Errorf("sem campanhas o aviso não pode prometer transferência: %q", noCampaigns)
	}
}

// ── o que virou teste de RENDER ──────────────────────────────────────────────
// "não oferece apagar a própria conta" não é regra pura: é o template decidindo
// o que desenhar. Renderizar o fragmento e olhar o HTML é o equivalente mais
// barato — não precisa de navegador, e afirma o mesmo resultado que o teste de
// integração afirmava.

func TestThePanelDoesNotOfferDeletingYourOwnAccount(t *testing.T) {
	view := adminView{Players: []playerRow{
		{ID: 1, Name: "Dono", Email: "dono@t.com", Belongings: "admin", Cost: "-", IsMe: true},
		{ID: 2, Name: "Outro", Email: "outro@t.com", Belongings: "-", Cost: "-", IsMe: false},
	}}

	html, err := ui.RenderFragment(t.Context(), playersPanel(view))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	text := html

	if strings.Contains(text, "Apagar a conta de Dono") {
		t.Error("o painel ofereceu apagar a PRÓPRIA conta — o servidor recusaria e o dono levaria um erro")
	}
	if !strings.Contains(text, "Apagar a conta de Outro") {
		t.Error("o painel deixou de oferecer apagar as outras contas")
	}
}

// O primeiro clique NÃO pode ser irreversível, e no Datastar isso é uma
// propriedade do MARCADOR: o botão da linha só abre o diálogo, e quem posta é o
// botão de dentro dele. Afirmar isso aqui é barato; o e2e irmão prova o
// comportamento no navegador.
func TestTheRowButtonOpensTheDialogInsteadOfDeleting(t *testing.T) {
	view := adminView{Players: []playerRow{{ID: 2, Name: "Outro", IsMe: false}}}
	row, err := ui.RenderFragment(t.Context(), playersPanel(view))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(row, "@post") {
		t.Error("o botão da linha posta direto — o primeiro clique virou irreversível")
	}
	if !strings.Contains(row, "showModal()") {
		t.Error("o botão da linha não abre o diálogo")
	}

	dialog, err := ui.RenderFragment(t.Context(), confirmDialog())
	if err != nil {
		t.Fatalf("render do diálogo: %v", err)
	}
	if !strings.Contains(dialog, "@post") {
		t.Error("quem apaga é o botão do diálogo, e ele não posta")
	}
}

// ── o link de redefinição ────────────────────────────────────────────────────

// Redefinir vale para TODA conta, inclusive a de quem está olhando — e é aí que
// ele se separa do Apagar, que tem a guarda do `IsMe`. O admin que esqueceu a
// própria senha usa esta mesma porta; sem isto ele fica de fora da única saída
// que o app oferece.
func TestResettingWorksForYourOwnAccountToo(t *testing.T) {
	view := adminView{Players: []playerRow{
		{ID: 1, Name: "Dono", Email: "dono@t.com", IsMe: true},
		{ID: 2, Name: "Outro", Email: "outro@t.com", IsMe: false},
	}}

	html, err := ui.RenderFragment(t.Context(), playersPanel(view))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, who := range []string{"Dono", "Outro"} {
		if !strings.Contains(html, "Redefinir a senha de "+who) {
			t.Errorf("o painel não oferece redefinir a senha de %q", who)
		}
	}
	// E o contraste com o Apagar continua valendo, senão este teste passaria
	// num painel que perdeu a guarda do `IsMe`.
	if strings.Contains(html, "Apagar a conta de Dono") {
		t.Error("o painel voltou a oferecer apagar a PRÓPRIA conta")
	}
}

// O prazo é 24h e não os 7 dias do convite, e a diferença é de RISCO: o convite
// abre uma conta que ainda não existe; este abre uma que já existe e tem fichas
// dentro. Um link esquecido numa conversa vale mais para um estranho.

// O botão da linha ABRE o diálogo; quem posta é o de dentro. Mesma propriedade
// do Apagar e pela mesma razão: gerar um link é um efeito no banco, e o
// primeiro clique não deve produzi-lo.
//
// E ele LIMPA o `#reset-link` ao abrir. Isso não é zelo: sem limpar, gerar o
// link da Ana, fechar, e abrir a caixa da Bia mostraria o link da ANA sob o
// nome da BIA — link de redefinição entregue à pessoa errada. O e2e irmão prova
// isso no navegador; aqui se afirma que a limpeza está no marcador.
func TestTheResetButtonOpensTheDialogAndClearsThePreviousLink(t *testing.T) {
	view := adminView{Players: []playerRow{{ID: 2, Name: "Outro", IsMe: false}}}
	row, err := ui.RenderFragment(t.Context(), playersPanel(view))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(row, "@post") {
		t.Error("o botão da linha posta direto — cunhar virou efeito do primeiro clique")
	}
	if !strings.Contains(row, "reset-link") {
		t.Error("abrir não limpa o link anterior — ele apareceria sob o nome errado")
	}

	dialog, err := ui.RenderFragment(t.Context(), resetDialog())
	if err != nil {
		t.Fatalf("render do diálogo: %v", err)
	}
	if !strings.Contains(dialog, "@post") {
		t.Error("quem cunha é o botão do diálogo, e ele não posta")
	}
}

// O remendo carrega o CAMINHO e nunca a URL inteira: quem prefixa a origem é o
// navegador. Com o `r.Host`, o link nasce apontando para o host que o proxy
// reescreveu — e link de redefinição existe para ser MANDADO, então host errado
// é link morto.
func TestTheResetPatchCarriesNoOrigin(t *testing.T) {
	html, err := ui.RenderFragment(t.Context(), mintedReset("/redefinir-senha?token=abc"))
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
