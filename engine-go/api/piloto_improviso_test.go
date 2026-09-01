package api

import (
	"net/http"
	"strings"
	"testing"
)

// Os guardas do IMPROVISO (ALE-261).
//
// O dado em si é do `engine` e tem teste lá. O que se prende aqui é a tradução
// da linha do livro para a tela — que é onde um campo trocado passa por dado
// plausível.

// TestTodaFaceDoDadoAcertaUmaLinha, nas quatro tabelas.
//
// É a única forma honesta de testar tabela de rolagem: em vez de repetir a
// tabela num `expect` por linha — que é a transcrição que o guia proíbe —,
// percorre TODAS as faces e exige que cada uma caia em alguma linha. Tabela com
// buraco é o defeito real aqui, e ele é invisível até alguém rolar o número que
// falta no meio de uma sessão.
func TestTodaFaceDoDadoAcertaUmaLinha(t *testing.T) {
	tab, masmorra := tabelasDoImproviso()
	if len(tab.Ruina) == 0 || len(tab.ChaseEvents) == 0 || len(masmorra.Ideas) == 0 {
		t.Fatal("tabelas vazias: o catálogo não carregou, e verde aqui não valeria nada")
	}
	for face := 1; face <= 6; face++ {
		if _, err := linhaOuErro(tab.Ruina, face); err != nil {
			t.Errorf("ruína: %v", err)
		}
		if _, err := linhaOuErro(tab.RewardCastigo, face); err != nil {
			t.Errorf("consequências: %v", err)
		}
	}
	for face := 1; face <= 20; face++ {
		if _, err := linhaOuErro(tab.ChaseEvents, face); err != nil {
			t.Errorf("perseguição: %v", err)
		}
		if _, err := linhaOuErro(masmorra.Ideas, face); err != nil {
			t.Errorf("ideias: %v", err)
		}
	}
}

func linhaOuErro[T interface{ Cobre(int) bool }](linhas []T, face int) (T, error) {
	var vazio T
	for _, l := range linhas {
		if l.Cobre(face) {
			return l, nil
		}
	}
	return vazio, errFaceSemLinha(face)
}

type erroDeFace int

func (e erroDeFace) Error() string {
	return "nenhuma linha cobre a face " + inteiro(int(e))
}
func errFaceSemLinha(f int) error { return erroDeFace(f) }

// TestOTipoDoEventoENAOOExemplo prende a manchete da perseguição.
//
// A primeira versão punha o EXEMPLO em cima e perdia o tipo inteiro; a rolagem 4
// saía como "4 —", porque na faixa "nenhum evento" o exemplo do livro é um
// travessão. Só apareceu ao olhar a captura de tela.
func TestOTipoDoEventoENaoOExemplo(t *testing.T) {
	tab, _ := tabelasDoImproviso()
	vistos := map[string]bool{}
	// 200 rolagens visitam as três faixas com folga; o que se mede é o FORMATO
	// da resposta, não o sorteio.
	for i := 0; i < 200; i++ {
		s, err := rolaPerseguicao()
		if err != nil {
			t.Fatalf("rolar: %v", err)
		}
		if s.Texto == "—" || s.Texto == "" {
			t.Fatalf("rolagem %d saiu sem manchete: %+v", s.Rolagem, s)
		}
		vistos[s.Texto] = true
	}
	for _, quero := range []string{"Nenhum evento", "Obstáculo", "Atalho"} {
		if !vistos[quero] {
			t.Errorf("em 200 rolagens nunca saiu %q — as faixas são %d", quero, len(tab.ChaseEvents))
		}
	}
}

// TestOHistoricoGuardaCincoEJogaOSextoFora.
func TestOHistoricoGuardaCincoEJogaOSextoFora(t *testing.T) {
	var h []sorteio
	for i := 1; i <= 8; i++ {
		h = empilha(h, sorteio{Rolagem: i, Texto: "linha"})
	}
	if len(h) != fundoDoHistorico {
		t.Fatalf("%d entradas, quero %d", len(h), fundoDoHistorico)
	}
	// O mais NOVO fica na frente: a tela mostra o último grande e os anteriores
	// em voz baixa, então a ordem é parte do contrato.
	if h[0].Rolagem != 8 {
		t.Errorf("a frente é a rolagem %d, quero a última (8)", h[0].Rolagem)
	}
	if h[len(h)-1].Rolagem != 4 {
		t.Errorf("o fundo é a rolagem %d, quero 4", h[len(h)-1].Rolagem)
	}
}

// TestOEsqueletoDaMasmorraSegueOLivro: uma ameaça a cada três salas (p263),
// arredondando PARA CIMA — sete salas dão três ameaças, não duas.
func TestOEsqueletoDaMasmorraSegueOLivro(t *testing.T) {
	casos := map[int]struct {
		ameacas int
		tamanho string
	}{
		3:  {1, "Pequena"},
		6:  {2, "Pequena"},
		7:  {3, "Média"},
		14: {5, "Média"},
	}
	for salas, quero := range casos {
		v := carregaImproviso(improvisoView{Salas: salas})
		if v.Ameacas != quero.ameacas {
			t.Errorf("%d salas deram %d ameaças, quero %d", salas, v.Ameacas, quero.ameacas)
		}
		if v.Tamanho == nil {
			t.Errorf("%d salas não casaram com tamanho nenhum", salas)
			continue
		}
		if v.Tamanho.Label != quero.tamanho {
			t.Errorf("%d salas viraram %q, quero %q", salas, v.Tamanho.Label, quero.tamanho)
		}
	}
}

// TestAcimaDoTetoNaoEErro: o livro recomenda parar, e a tela diz isso em vez de
// esconder o campo ou fingir um tamanho.
func TestAcimaDoTetoNaoEErro(t *testing.T) {
	v := carregaImproviso(improvisoView{Salas: 120})
	if !v.AcimaDoTeto {
		t.Fatal("120 salas não foram marcadas como acima do teto")
	}
	if v.Tamanho != nil {
		t.Errorf("e ainda inventaram o tamanho %q", v.Tamanho.Label)
	}
	// Salas absurdas caem no padrão, como as outras cenas: o número vem dos
	// sinais e alguém edita à mão.
	if got := carregaImproviso(improvisoView{Salas: -4}).Salas; got != salasPadrao {
		t.Errorf("-4 salas viraram %d, quero o padrão %d", got, salasPadrao)
	}
}

// ── pelo fio ─────────────────────────────────────────────────────────────────

// TestRolarDevolveRemendoEEmpilha.
func TestRolarDevolveRemendoEEmpilha(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, eu, "POST", "/mestre/improviso/ruina",
		`{"ruina":[{"r":2,"t":"Vazia"}]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type %q — a rolagem navegou em vez de remendar", ct)
	}
	// A anterior tem de sobreviver: o histórico é o que separa esta tabela de um
	// botão que só mostra o último.
	if !strings.Contains(rec.Body.String(), "Vazia") {
		t.Error("o remendo perdeu a rolagem anterior")
	}
}

// TestTabelaInventadaERecusada: a rota é montada a partir da própria lista,
// então nome errado só chega por URL digitada à mão — e devolver a cena intacta
// faria parecer que o botão não funciona.
func TestTabelaInventadaERecusada(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeNoMestre(t, s, eu, "POST", "/mestre/improviso/tarot", `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d, quero 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "tarot") {
		t.Error("a recusa não diz qual tabela foi recusada")
	}
}

// TestOTrilhoOfereceTodasAsParadas — e cada uma responde.
//
// É o guarda da VIRADA: a `/gm` só pode ser apagada quando as ferramentas
// estiverem de pé, e "de pé" é responder 200, não existir no trilho.
//
// Eram QUATRO até a ALE-264, quando o dono viu que "o bestiário conta como
// catálogo": o trilho virou duas seções e cada catálogo ganhou parada e cena
// próprias. São 13 — duas ferramentas e ONZE catálogos, os últimos a chegar
// sendo as escolas de magia e as perícias. O número fica preso porque uma parada
// que perde a rota some do trilho sem erro nenhum.
func TestOTrilhoOfereceTodasAsParadas(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	if len(ferramentasDoMestre) != 13 {
		t.Fatalf("o trilho tem %d paradas", len(ferramentasDoMestre))
	}
	// E as DUAS seções existem: sem elas o trilho volta a ser uma lista só, que
	// é o que o dono pediu para desfazer.
	if len(trilhoDoMestre) != 2 {
		t.Fatalf("o trilho tem %d seções", len(trilhoDoMestre))
	}
	for _, f := range ferramentasDoMestre {
		t.Run(f.Slug, func(t *testing.T) {
			rec := pedeNoMestre(t, s, eu, "GET", "/mestre/"+f.Slug, "")
			if rec.Code != http.StatusOK {
				t.Fatalf("%s respondeu %d", f.Slug, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), f.Rotulo) {
				t.Errorf("a cena de %s não desenha o próprio nome", f.Slug)
			}
		})
	}
}

// TestOsSlugsDaTrilhaSaoUnicos — a rota resolve por eles.
//
// Herdado do `gm-tools.test.ts` da SPA, apagado na virada da ALE-264 — o
// original se lê com
// `git show 7956b59:frontend/src/features/gm-tools/gm-tools.test.ts`. Slug repetido não
// quebra compilação nem teste nenhum: as duas entradas viram links para o mesmo
// endereço, e a segunda ferramenta fica inalcançável — com o trilho mostrando
// as duas, o que é pior que faltar uma.
func TestOsSlugsDaTrilhaSaoUnicos(t *testing.T) {
	vistos := map[string]string{}
	for _, f := range ferramentasDoMestre {
		if antes, repetido := vistos[f.Slug]; repetido {
			t.Errorf("o slug %q é de %q e de %q — a segunda fica inalcançável",
				f.Slug, antes, f.Rotulo)
		}
		vistos[f.Slug] = f.Rotulo
		if f.Slug == "" || f.Rotulo == "" || f.Icone == "" {
			t.Errorf("a ferramenta %+v tem campo vazio", f)
		}
	}
}

// TestLimparZeraSOAquelaTabela.
//
// Esta feature quase se perdeu no porte: a SPA tem um botão "Limpar" por tabela
// e a minha primeira versão rolava e acumulava sem como zerar. O que a
// denunciou foi comparar o teste ÓRFÃO da SPA (`roll-history.test.ts` em
// 7956b59, "limpar
// esvazia o histórico") com o substituto em Go ANTES de apagá-lo — apagar
// primeiro teria levado a testemunha junto, que é o que o checklist da virada
// existe para impedir.
//
// O guarda mede o ISOLAMENTO e não só o zeramento: limpar a ruína não pode
// levar junto o evento de perseguição que o mestre acabou de tirar.
func TestLimparZeraSoAquelaTabela(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	sinais := `{"ruina":[{"r":4,"t":"Vazia"}],"perseguicao":[{"r":9,"t":"Obstáculo"}],` +
		`"recompensa":[{"r":2,"t":"Favor"}],"ideias":[{"r":7,"t":"Cripta"}]}`
	rec := pedeNoMestre(t, s, eu, "POST", "/mestre/improviso/ruina/limpar", sinais)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	corpo := rec.Body.String()
	if strings.Contains(corpo, "Vazia") {
		t.Error("a ruína não foi limpa")
	}
	for _, sobrevivente := range []string{"Obstáculo", "Favor", "Cripta"} {
		if !strings.Contains(corpo, sobrevivente) {
			t.Errorf("limpar a ruína levou junto %q — as tabelas são independentes", sobrevivente)
		}
	}
}

// TestLimparTabelaInventadaERecusado, como a rolagem.
func TestLimparTabelaInventadaERecusado(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")
	rec := pedeNoMestre(t, s, eu, "POST", "/mestre/improviso/tarot/limpar", `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status %d, quero 400", rec.Code)
	}
}
