package api

import (
	"strings"
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

func TestExpiraEm(t *testing.T) {
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

func TestPossesEComoSeLeem(t *testing.T) {
	if got := posses(false, 2, 1); got != "2 campanhas · 1 ficha" {
		t.Errorf("posses = %q", got)
	}
	if got := posses(true, 1, 3); got != "admin · 1 campanha · 3 fichas" {
		t.Errorf("posses de admin = %q", got)
	}
}

// O aviso tem de dizer o preço DESTA conta: um texto genérico não distingue
// apagar uma conta vazia de apagar a do jogador que mestra duas campanhas.
func TestCustoDeApagarDizOPrecoDaConta(t *testing.T) {
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

func TestPainelNaoOfereceApagarAPropriaConta(t *testing.T) {
	view := adminView{Jogadores: []adminJogador{
		{ID: 1, Nome: "Dono", Email: "dono@t.com", Posses: "admin", Custo: "-", EhEu: true},
		{ID: 2, Nome: "Outro", Email: "outro@t.com", Posses: "-", Custo: "-", EhEu: false},
	}}

	html, err := renderFragmento("admin-jogadores", view)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	texto := string(html)

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
func TestOBotaoDaLinhaAbreODialogoEmVezDeApagar(t *testing.T) {
	view := adminView{Jogadores: []adminJogador{{ID: 2, Nome: "Outro", EhEu: false}}}
	linha, err := renderFragmento("admin-jogadores", view)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(string(linha), "@post") {
		t.Error("o botão da linha posta direto — o primeiro clique virou irreversível")
	}
	if !strings.Contains(string(linha), "showModal()") {
		t.Error("o botão da linha não abre o diálogo")
	}

	dialogo, err := renderFragmento("admin-confirmar", view)
	if err != nil {
		t.Fatalf("render do diálogo: %v", err)
	}
	if !strings.Contains(string(dialogo), "@post") {
		t.Error("quem apaga é o botão do diálogo, e ele não posta")
	}
}
