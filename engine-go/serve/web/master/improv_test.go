package master

import (
	"net/http"
	"strings"
	"t20engine/domain/book"
	"t20engine/serve/web/ui"
	"testing"
)

// Os guardas do IMPROVISO.
//
// O dado em si é do `engine` e tem teste lá. O que se prende aqui é a tradução
// da linha do livro para a tela — que é onde um campo trocado passa por dado
// plausível.

// TODA face do dado cai numa linha, nas quatro tabelas.
//
// É a única forma honesta de testar tabela de rolagem: em vez de repetir a
// tabela num `expect` por linha — que é a transcrição que o guia proíbe —,
// percorre TODAS as faces e exige que cada uma caia em alguma linha. Tabela com
// buraco é o defeito real aqui, e ele é invisível até alguém rolar o número que
// falta no meio de uma sessão.
func TestEveryDieFaceHitsARow(t *testing.T) {
	tab, dungeon := book.ImprovTables()
	if len(tab.Ruin) == 0 || len(tab.ChaseEvents) == 0 || len(dungeon.Ideas) == 0 {
		t.Fatal("tabelas vazias: o catálogo não carregou, e verde aqui não valeria nada")
	}
	for face := 1; face <= 6; face++ {
		if _, err := linhaOuErro(tab.Ruin, face); err != nil {
			t.Errorf("ruína: %v", err)
		}
		if _, err := linhaOuErro(tab.RewardPunishment, face); err != nil {
			t.Errorf("consequências: %v", err)
		}
	}
	for face := 1; face <= 20; face++ {
		if _, err := linhaOuErro(tab.ChaseEvents, face); err != nil {
			t.Errorf("perseguição: %v", err)
		}
		if _, err := linhaOuErro(dungeon.Ideas, face); err != nil {
			t.Errorf("ideias: %v", err)
		}
	}
}

func linhaOuErro[T interface{ Covers(int) bool }](rows []T, face int) (T, error) {
	var empty T
	for _, l := range rows {
		if l.Covers(face) {
			return l, nil
		}
	}
	return empty, errFaceSemLinha(face)
}

type erroDeFace int

func (e erroDeFace) Error() string {
	return "nenhuma linha cobre a face " + ui.Int(int(e))
}
func errFaceSemLinha(f int) error { return erroDeFace(f) }

// A manchete da perseguição é o TIPO do evento, e não o exemplo.
//
// Com o exemplo em cima, a rolagem 4 sai como "4 —": na faixa "nenhum evento" o
// exemplo do livro é um travessão, e o tipo se perde inteiro.
func TestTheEventTypeAndNotTheExample(t *testing.T) {
	tab, _ := book.ImprovTables()
	seen := map[string]bool{}
	// 200 rolagens visitam as três faixas com folga; o que se mede é o FORMATO
	// da resposta, não o sorteio.
	for i := 0; i < 200; i++ {
		s, err := rollChase()
		if err != nil {
			t.Fatalf("rolar: %v", err)
		}
		if s.Text == "—" || s.Text == "" {
			t.Fatalf("rolagem %d saiu sem manchete: %+v", s.Roll, s)
		}
		seen[s.Text] = true
	}
	for _, want := range []string{"Nenhum evento", "Obstáculo", "Atalho"} {
		if !seen[want] {
			t.Errorf("em 200 rolagens nunca saiu %q — as faixas são %d", want, len(tab.ChaseEvents))
		}
	}
}

// O histórico guarda cinco e joga o sexto fora.
func TestTheHistoryKeepsFiveAndThrowsTheSixthAway(t *testing.T) {
	var h []roll
	for i := 1; i <= 8; i++ {
		h = push(h, roll{Roll: i, Text: "linha"})
	}
	if len(h) != historyDepth {
		t.Fatalf("%d entradas, quero %d", len(h), historyDepth)
	}
	// O mais NOVO fica na frente: a tela mostra o último grande e os anteriores
	// em voz baixa, então a ordem é parte do contrato.
	if h[0].Roll != 8 {
		t.Errorf("a frente é a rolagem %d, quero a última (8)", h[0].Roll)
	}
	if h[len(h)-1].Roll != 4 {
		t.Errorf("o fundo é a rolagem %d, quero 4", h[len(h)-1].Roll)
	}
}

// O esqueleto da masmorra segue o livro: uma ameaça a cada três salas (p263),
// arredondando PARA CIMA — sete salas dão três ameaças, não duas.
func TestTheDungeonSkeletonFollowsTheBook(t *testing.T) {
	cases := map[int]struct {
		threats int
		size    string
	}{
		3:  {1, "Pequena"},
		6:  {2, "Pequena"},
		7:  {3, "Média"},
		14: {5, "Média"},
	}
	for rooms, want := range cases {
		v := loadImprov(improvView{Rooms: rooms})
		if v.Threats != want.threats {
			t.Errorf("%d salas deram %d ameaças, quero %d", rooms, v.Threats, want.threats)
		}
		if v.Size == nil {
			t.Errorf("%d salas não casaram com tamanho nenhum", rooms)
			continue
		}
		if v.Size.Label != want.size {
			t.Errorf("%d salas viraram %q, quero %q", rooms, v.Size.Label, want.size)
		}
	}
}

// Acima do teto não é erro: o livro recomenda parar, e a tela diz isso em vez de
// esconder o campo ou fingir um tamanho.
func TestAboveTheCeilingIsNotAnError(t *testing.T) {
	v := loadImprov(improvView{Rooms: 120})
	if !v.OverCeiling {
		t.Fatal("120 salas não foram marcadas como acima do teto")
	}
	if v.Size != nil {
		t.Errorf("e ainda inventaram o tamanho %q", v.Size.Label)
	}
	// Salas absurdas caem no padrão, como as outras cenas: o número vem dos
	// sinais e alguém edita à mão.
	if got := loadImprov(improvView{Rooms: -4}).Rooms; got != salasPadrao {
		t.Errorf("-4 salas viraram %d, quero o padrão %d", got, salasPadrao)
	}
}

// ── pelo fio ─────────────────────────────────────────────────────────────────

// Os slugs do trilho são ÚNICOS — a rota resolve por eles.
//
// Slug repetido não quebra compilação nem teste nenhum: as duas entradas viram
// links para o mesmo endereço, e a segunda ferramenta fica inalcançável — com o
// trilho mostrando as duas, o que é pior que faltar uma.
func TestTheTrailSlugsAreUnique(t *testing.T) {
	seen := map[string]string{}
	for _, f := range railStops {
		if before, repeated := seen[f.Slug]; repeated {
			t.Errorf("o slug %q é de %q e de %q — a segunda fica inalcançável",
				f.Slug, before, f.Label)
		}
		seen[f.Slug] = f.Label
		if f.Slug == "" || f.Label == "" || f.Icon == "" {
			t.Errorf("a ferramenta %+v tem campo vazio", f)
		}
	}
}

// O trilho oferece TODA parada, e cada uma responde.
//
// "De pé" é responder 200, não existir no trilho: uma parada que perde a rota
// some do trilho sem erro nenhum, e por isso o número fica preso.
func TestTheRailOffersEveryStop(t *testing.T) {
	if len(railStops) != 13 {
		t.Fatalf("o trilho tem %d paradas", len(railStops))
	}
	// E as DUAS seções existem: sem elas o trilho volta a ser uma lista só, que
	// é o que o dono pediu para desfazer.
	if len(masterRail) != 2 {
		t.Fatalf("o trilho tem %d seções", len(masterRail))
	}
	for _, f := range railStops {
		t.Run(f.Slug, func(t *testing.T) {
			rec := pedeNaCena(t, "/mestre/"+f.Slug)
			if rec.Code != http.StatusOK {
				t.Fatalf("%s respondeu %d", f.Slug, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), f.Label) {
				t.Errorf("a cena de %s não desenha o próprio nome", f.Slug)
			}
		})
	}
}

// Limpar zera UMA tabela.
//
// O guarda mede o ISOLAMENTO e não só o zeramento: limpar a ruína não pode levar
// junto o evento de perseguição que o mestre acabou de tirar.
