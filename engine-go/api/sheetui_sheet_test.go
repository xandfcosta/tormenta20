package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"t20engine/web/sheetui"
	"testing"
)

func TestTheSheetTabAddressSurvives(t *testing.T) {
	for _, caso := range []struct{ pedido, esperado string }{
		{"abilities", "abilities"},
		{"inventory", "bag"},
		{"equipment", "bag"},
		{"", "expertises"},
		{"nao-existe", "expertises"},
	} {
		if achou := sheetui.AskedTab(caso.pedido); achou != caso.esperado {
			t.Errorf("?tab=%q abriu %q, esperado %q", caso.pedido, achou, caso.esperado)
		}
	}
}

// A ABA NÃO PORTADA DIZ ISSO, e leva para a ficha antiga.
//
// Enquanto os painéis não chegam, a casca não pode fingir: uma seção vazia é
// lida como defeito, e mandar a pessoa procurar sozinha o endereço velho é pior
// do que dar o link. Este guarda morre junto com a última fatia — quando não

// Aqui moravam DOIS testes que a fatia 10 aposentou.
//
// O `TestAAbaAindaNaoPortadaLevaParaAFichaAntiga` abria uma aba sem painel e
// afirmava que ela mandava para a ficha velha; ele perdeu o alvo na fatia 8,
// quando deixou de existir aba sem painel.
//
// O `TestTodaAbaDaFichaEstaPortada` era o placar da migração — ele autorizava
// esta fatia a apagar a ficha antiga, e cumpriu isso. O `oPainelJaPortado` que
// ele lia não existe mais, porque "portada" deixou de ser uma pergunta. O que
// segue valendo é `TestEverySheetTabDrawsSomething`, que cobra painel de TODA
// aba — a mesma garantia, sem o placar.

func sheetOf(t *testing.T, nome string, nivel int64) (pilotoFixture, int64) {
	t.Helper()
	f := novoPiloto(t)
	id := seedCharacterAtLevel(t, f.s, f.jogador, nome, nivel, 20, 20, 10, 10)
	seedClasse(t, f.s, id, "Arcanista", nivel)
	return f, id
}

// O NÍVEL É DA CLASSE, e o do personagem é a SOMA — guarda de regressão.
//
// A primeira versão deste comando escrevia direto no nível do personagem, e o
// defeito é silencioso do pior jeito: a ficha passa a dizer 13 com as classes
// somando 12, e os pools de PV e PM — que derivam das CLASSES — não se mexem. O
// número sobe e o personagem não fica mais forte.
//
// Eu não achei isso lendo o código: achei comparando com a SPA no navegador,
// onde o degrau chama `PATCH /classes/level`. Por isso o caso prende as DUAS
// metades: a soma bater, e o pool passar a dizer o que o livro diz.
func TestTheLevelStepRaisesTheClassAndNotOnlyTheTotal(t *testing.T) {
	f, id := sheetOf(t, "Arcanista Nv3", 3)
	ctx := context.Background()
	antes, err := f.s.Queries().GetCharacter(ctx, id)
	if err != nil {
		t.Fatalf("ler o personagem: %v", err)
	}

	rec := f.pede(t, f.jogador, http.MethodPost,
		fmt.Sprintf("/personagens/%d/nivel/Arcanista/1", id), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("subir de nível deu %d: %s", rec.Code, rec.Body.String())
	}

	depois, err := f.s.Queries().GetCharacter(ctx, id)
	if err != nil {
		t.Fatalf("reler: %v", err)
	}
	classes, err := f.s.Queries().ListClassesByCharacter(ctx, id)
	if err != nil {
		t.Fatalf("ler as classes: %v", err)
	}
	var soma int64
	for _, cl := range classes {
		soma += cl.Level
	}
	if soma != 4 {
		t.Errorf("a CLASSE não subiu: as classes somam %d, esperado 4", soma)
	}
	if depois.Level != soma {
		t.Errorf("o nível do personagem (%d) não é a soma das classes (%d)", depois.Level, soma)
	}
	// E OS POOLS ACOMPANHAM. O número é do LIVRO e escrito à mão: o Arcanista tem
	// PV inicial 8 e +2 por nível (T20 p36, a tabela da classe; a regra da soma
	// está em p34), então no nível 4 com Constituição 0 são 8 + 3×2 = **14**.
	//
	// A primeira versão deste caso afirmava "o PV máximo CRESCEU", e ela estava
	// errada de um jeito instrutivo: o personagem semeado tinha 20 gravados, que
	// não é um número do motor — sincronizar o BAIXOU para 14, e "cresceu" falhou
	// sobre um app correto. O que a sincronização garante não é crescer: é a
	// ficha passar a dizer o que o livro diz.
	if depois.Hpmax != 14 {
		t.Errorf("o PV máximo do Arcanista 4 ficou em %d, e o livro dá 14 (8 inicial + 3×2, p36)", depois.Hpmax)
	}
	if antes.Hpmax == depois.Hpmax {
		t.Error("o PV máximo não se mexeu: o degrau gravou o nível sem sincronizar os pools")
	}
}

// DESCER uma classe de nível 1 é recusado: levá-la a zero apagaria a classe, e
// apagar classe é outra coisa — não tem gesto nesta tela e não pode acontecer
// por acidente num botão de menos.
func TestTheLevelStepDoesNotEraseALevelOneClass(t *testing.T) {
	f, id := sheetOf(t, "Aprendiz", 1)

	rec := f.pede(t, f.jogador, http.MethodPost,
		fmt.Sprintf("/personagens/%d/nivel/Arcanista/-1", id), "")

	recusa := sceneRefusal(rec.Body.String())
	if recusa == "" {
		t.Fatal("desceu uma classe de nível 1: a classe teria sumido da ficha")
	}
	if !strings.Contains(recusa, "apagaria a classe") {
		t.Errorf("a recusa não diz o que ia acontecer: %q", recusa)
	}
}

// O VITAL PRENDE na faixa em vez de recusar.
//
// É a diferença entre o gesto e a API: o `PATCH /vitals` manda o valor absoluto
// e recusa fora da faixa, o que está certo para um cliente que calculou. Aqui o
// gesto é "levou seis" — com 4 de PV o resultado é zero, e uma recusa faria o
// mestre clicar quatro vezes de um em um para chegar no mesmo lugar.
func TestTheVitalClampsAtZeroAndAtTheMaximum(t *testing.T) {
	f, id := sheetOf(t, "Alvo", 3)
	ctx := context.Background()
	url := fmt.Sprintf("/personagens/%d/vitais/pv/", id)

	// Cinco golpes de −5 sobre 20 de PV: para em zero e não vira negativo.
	for i := 0; i < 5; i++ {
		if rec := f.pede(t, f.jogador, http.MethodPost, url+"-5", ""); rec.Code != http.StatusOK {
			t.Fatalf("ferir deu %d", rec.Code)
		}
	}
	ferido, _ := f.s.Queries().GetCharacter(ctx, id)
	if ferido.Hpcurrent != 0 {
		t.Errorf("o PV foi para %d: o passo tinha de prender em zero", ferido.Hpcurrent)
	}

	// E curar além do máximo para NO máximo: passar dele seria PV temporário,
	// que é outra regra e tem dono no motor.
	for i := 0; i < 6; i++ {
		f.pede(t, f.jogador, http.MethodPost, url+"5", "")
	}
	curado, _ := f.s.Queries().GetCharacter(ctx, id)
	if curado.Hpcurrent != curado.Hpmax {
		t.Errorf("o PV parou em %d com máximo %d", curado.Hpcurrent, curado.Hpmax)
	}
}

// A FICHA É DO DONO. A trava é do servidor, e não da tela não oferecer o link:
// quem digitar o endereço de outro personagem leva 403.
func TestSomeoneElsesSheetDoesNotOpen(t *testing.T) {
	f, id := sheetOf(t, "Segredo", 3)

	rec := f.pede(t, f.mestre, http.MethodGet, fmt.Sprintf("/personagens/%d", id), "")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("a ficha de outra pessoa abriu com %d", rec.Code)
	}
	// E o gesto também: barrar a leitura e deixar a escrita passar seria pior
	// que não barrar nada.
	escrita := f.pede(t, f.mestre, http.MethodPost,
		fmt.Sprintf("/personagens/%d/vitais/pv/-5", id), "")
	if escrita.Code != http.StatusForbidden {
		t.Errorf("alguém feriu o personagem de outra pessoa: %d", escrita.Code)
	}
}

// A RECUSA VOLTA PELA CENA, e não por um status que o cliente descarta.
//
// Medido no navegador (ALE-272, fatia 7): com `http.Error(400)` o cliente do
// Datastar não aplicava o remendo, e a única marca da recusa era uma linha
// vermelha no CONSOLE. Na tela o gesto simplesmente não acontecia — gastar mais
// do que se tem fechava o diálogo e deixava o saldo igual, sem uma palavra.
//
// Este guarda prende as TRÊS coisas que fazem a recusa chegar: o status que o
// cliente aceita, a frase, e a cena INTEIRA junto — é ela que mostra o estado
// que não mudou.
func TestTheRefusalComesBackInTheSceneAndNotInAnErrorStatus(t *testing.T) {
	f, id := sheetOf(t, "Herói", 3)

	rec := f.pede(t, f.jogador, http.MethodPost,
		fmt.Sprintf("/personagens/%d/proficiencias/alterna/armas-de-laser?tab=proficiencies", id), "")

	if rec.Code != http.StatusOK {
		t.Fatalf("a recusa respondeu %d: o cliente do Datastar descarta o remendo e a tela não muda",
			rec.Code)
	}
	if recusa := sceneRefusal(rec.Body.String()); recusa == "" {
		t.Error("a recusa não escreveu nada na cena")
	}
	if !strings.Contains(rec.Body.String(), "Seções da ficha") {
		t.Error("a resposta não traz a cena inteira: o remendo apagaria a ficha em vez de avisar")
	}
}
