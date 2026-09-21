package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"t20engine/serve/web/sheetui"
	"testing"
)

func TestTheSheetTabAddressSurvives(t *testing.T) {
	for _, tc := range []struct{ requested, want string }{
		{"abilities", "abilities"},
		{"", "expertises"},
		{"nao-existe", "expertises"},
	} {
		if found := sheetui.AskedTab(tc.requested); found != tc.want {
			t.Errorf("?tab=%q abriu %q, esperado %q", tc.requested, found, tc.want)
		}
	}
}

// Não há caso de "aba ainda não portada" nem placar de migração aqui de
// propósito: não existe mais aba sem painel, e quem cobra painel de TODA aba é o
// `TestEverySheetTabDrawsSomething`.

func sheetOf(t *testing.T, name string, level int64) (sceneFixture, int64) {
	t.Helper()
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, name, "Arcanista", level, 0, 0)
	return f, id
}

// O NÍVEL É DA CLASSE, e o do personagem é a SOMA — guarda de regressão.
//
// Escrever direto no nível do personagem falha em silêncio do pior jeito: a
// ficha passa a dizer 13 com as classes somando 12, e os pools de PV e PM — que
// derivam das CLASSES — não se mexem. O número sobe e o personagem não fica mais
// forte. Por isso o caso prende as DUAS metades: a soma bater, e o pool passar a
// dizer o que o livro diz.
func TestTheLevelStepRaisesTheClassAndNotOnlyTheTotal(t *testing.T) {
	f, id := sheetOf(t, "Arcanista Nv3", 3)
	ctx := context.Background()
	before := poolsOf(t, f.s, id)

	rec := f.pede(t, f.player, http.MethodPost,
		fmt.Sprintf("/personagens/%d/nivel/Arcanista/1", id), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("subir de nível deu %d: %s", rec.Code, rec.Body.String())
	}

	after, err := f.s.sceneCore().Queries().GetCharacter(ctx, id)
	if err != nil {
		t.Fatalf("reler: %v", err)
	}
	classes, err := f.s.sceneCore().Queries().ListClassesByCharacter(ctx, id)
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
	if after.Level != soma {
		t.Errorf("o nível do personagem (%d) não é a soma das classes (%d)", after.Level, soma)
	}
	// E OS POÇOS ACOMPANHAM. O número é do LIVRO e escrito à mão: *"Um arcanista
	// começa com 8 pontos de vida (+ Constituição) e ganha 2 PV (+ Constituição)
	// por nível"* (**p37**); a regra da soma dos níveis de classe está na
	// **p35**. Então no nível 4 com Constituição 0 são 8 + 3×2 = **14**.
	//
	// As duas páginas estavam erradas aqui — p36 e p34 — e foram conferidas no
	// livro, uma de cada vez (ALE-347). A p34 é uma ilustração de página inteira.
	//
	// O poço acompanha de GRAÇA desde a ALE-355: ele é derivado das classes a
	// cada leitura, e o degrau só precisa gravar o nível. As duas metades
	// continuam prendidas — o número do livro e o fato de ele ter MEXIDO —
	// porque um degrau que não gravasse a classe deixaria os dois parados.
	pool := poolsOf(t, f.s, id)
	if pool.HpMax != 14 {
		t.Errorf("o PV máximo do Arcanista 4 ficou em %d, e o livro dá 14 (8 inicial + 3×2, p36)", pool.HpMax)
	}
	if before.HpMax == pool.HpMax {
		t.Error("o PV máximo não se mexeu: o degrau não gravou o nível da classe")
	}
}

// DESCER uma classe de nível 1 é recusado: levá-la a zero apagaria a classe, e
// apagar classe é outra coisa — não tem gesto nesta tela e não pode acontecer
// por acidente num botão de menos.
func TestTheLevelStepDoesNotEraseALevelOneClass(t *testing.T) {
	f, id := sheetOf(t, "Aprendiz", 1)

	rec := f.pede(t, f.player, http.MethodPost,
		fmt.Sprintf("/personagens/%d/nivel/Arcanista/-1", id), "")

	refusal := sceneRefusal(rec.Body.String())
	if refusal == "" {
		t.Fatal("desceu uma classe de nível 1: a classe teria sumido da ficha")
	}
	if !strings.Contains(refusal, "apagaria a classe") {
		t.Errorf("a recusa não diz o que ia acontecer: %q", refusal)
	}
}

// O VITAL PRENDE na faixa em vez de recusar.
//
// É a diferença entre o gesto e a API: o `PATCH /vitals` manda o valor absoluto
// e recusa fora da faixa, o que está certo para um cliente que calculou. Aqui o
// gesto é "levou seis" — e uma recusa faria o mestre clicar de um em um para
// chegar no mesmo lugar.
//
// A FAIXA DO PV desce abaixo de zero até o limiar da morte (p236, ALE-366): com
// até 20 PV totais o limiar é –10, e é lá que o passo para.
func TestTheVitalClampsAtTheDeathThresholdAndAtTheMaximum(t *testing.T) {
	f, id := sheetOf(t, "Alvo", 3)
	url := fmt.Sprintf("/personagens/%d/vitais/pv/", id)
	if total := poolsOf(t, f.s, id).HpMax; total > 20 {
		t.Fatalf("o controle falhou: a ficha tem %d PV totais, e o caso precisa de até 20 para o limiar ser -10", total)
	}

	// Dez golpes de −5: bem abaixo do limiar, e o passo para nele.
	for i := 0; i < 10; i++ {
		if rec := f.pede(t, f.player, http.MethodPost, url+"-5", ""); rec.Code != http.StatusOK {
			t.Fatalf("ferir deu %d", rec.Code)
		}
	}
	if wounded := poolsOf(t, f.s, id); wounded.HpCurrent != -10 {
		t.Errorf("o PV foi para %d: o passo tinha de parar no limiar da morte, -10", wounded.HpCurrent)
	}

	// E curar além do máximo para NO máximo: passar dele seria PV temporário,
	// que é outra regra e tem dono no motor.
	for i := 0; i < 6; i++ {
		f.pede(t, f.player, http.MethodPost, url+"5", "")
	}
	if healed := poolsOf(t, f.s, id); healed.HpCurrent != healed.HpMax {
		t.Errorf("o PV parou em %d com máximo %d", healed.HpCurrent, healed.HpMax)
	}
}

// A FICHA É DO DONO. A trava é do servidor, e não da tela não oferecer o link:
// quem digitar o endereço de outro personagem leva 403.
func TestSomeoneElsesSheetDoesNotOpen(t *testing.T) {
	f, id := sheetOf(t, "Segredo", 3)

	rec := f.pede(t, f.gm, http.MethodGet, fmt.Sprintf("/personagens/%d", id), "")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("a ficha de outra pessoa abriu com %d", rec.Code)
	}
	// E o gesto também: barrar a leitura e deixar a escrita passar seria pior
	// que não barrar nada.
	write := f.pede(t, f.gm, http.MethodPost,
		fmt.Sprintf("/personagens/%d/vitais/pv/-5", id), "")
	if write.Code != http.StatusForbidden {
		t.Errorf("alguém feriu o personagem de outra pessoa: %d", write.Code)
	}
}

// A RECUSA VOLTA PELA CENA, e não por um status que o cliente descarta.
//
// Com `http.Error(400)` o cliente do Datastar não aplica o remendo, e a única
// marca da recusa é uma linha vermelha no CONSOLE: na tela o gesto simplesmente
// não acontece — gastar mais do que se tem fecha o diálogo e deixa o saldo
// igual, sem uma palavra.
//
// Este guarda prende as TRÊS coisas que fazem a recusa chegar: o status que o
// cliente aceita, a frase, e a cena INTEIRA junto — é ela que mostra o estado
// que não mudou.
func TestTheRefusalComesBackInTheSceneAndNotInAnErrorStatus(t *testing.T) {
	f, id := sheetOf(t, "Herói", 3)

	rec := f.pede(t, f.player, http.MethodPost,
		fmt.Sprintf("/personagens/%d/proficiencias/alterna/armas-de-laser?tab=proficiencies", id), "")

	if rec.Code != http.StatusOK {
		t.Fatalf("a recusa respondeu %d: o cliente do Datastar descarta o remendo e a tela não muda",
			rec.Code)
	}
	if refusal := sceneRefusal(rec.Body.String()); refusal == "" {
		t.Error("a recusa não escreveu nada na cena")
	}
	if !strings.Contains(rec.Body.String(), "Seções da ficha") {
		t.Error("a resposta não traz a cena inteira: o remendo apagaria a ficha em vez de avisar")
	}
}

// A COR da barra de PV diz "quão mal", e não só a largura.
//
// A escada tem três degraus — crítico até 25%, ferido até 50%, cheio acima — e a
// ficha já pintou `--hp-full` sempre, com o herói a 17,5% saindo verde enquanto
// a Mesa o pintava de vermelho. Nenhum guarda pegava: os casos de vital afirmam
// o NÚMERO, e o número sempre esteve certo. A largura também.
//
// Um caso só no crítico passaria verde sobre uma barra que pintasse crítico
// SEMPRE — o inverso exato do defeito, e igualmente invisível. Por isso os
// degraus são afirmados na descida, nas porcentagens de FRONTEIRA (75, 50, 25),
// com os tons escritos à mão: derivá-los de `ui.HpFillTone` faria a asserção
// andar junto com o defeito.
func TestTheSheetPaintsTheHpLadderAndNotOnlyTheWidth(t *testing.T) {
	f, id := sheetOf(t, "Ferido", 3)

	// O PASSO é um QUARTO do poço, e não um número escolhido: o poço vem do
	// livro, e um passo fixo de 5 atravessaria as fronteiras em outro lugar no
	// dia em que a tabela de classe mudasse — sem ninguém mexer no teste.
	pool := bookPools(t, f.s, "Arcanista", 3).PvMax
	url := fmt.Sprintf("/personagens/%d/vitais/pv/-%d", id, pool/4)

	// CHEIO, e este é o controle: sem ele, pintar crítico sempre passaria em
	// tudo que vem depois.
	screen := f.pede(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d", id), "").Body.String()
	if tom := hpTintOf(t, screen); tom != "full" {
		t.Errorf("com o poço cheio a faixa saiu %q, e vida cheia é `full`", tom)
	}

	// A descida, degrau por degrau, nas fronteiras da escada.
	for _, tc := range []struct {
		fraction string
		pct      int
		tom      string
	}{
		{"três quartos", 75, "full"},
		{"metade", 50, "hurt"},
		{"um quarto", 25, "critical"},
	} {
		body := f.pede(t, f.player, http.MethodPost, url, "").Body.String()
		if tom := hpTintOf(t, body); tom != tc.tom {
			t.Errorf("com %s do poço (%d%%) a faixa saiu %q, e o esperado é %q",
				tc.fraction, tc.pct, tom, tc.tom)
		}
	}
}

// O CRACHÁ MOSTRA A RESERVA DE PV TEMPORÁRIO, e ela é uma parcela à parte.
//
// O dado estava certo e a tela calava: o `PlanDamage` gasta a reserva antes do
// PV desde sempre, e o crachá dizia 137/137 com 30 de colchão. Defeito de
// APRESENTAÇÃO, que é a família da ALE-319.
//
// O CONTROLE vem primeiro e é obrigatório: um mostrador cujo REPOUSO é igual ao
// sucesso não testemunha nada. A ficha sem poça não pode ter a marca — senão
// este caso ficaria verde sobre uma tela que mostra a reserva o tempo todo.
func TestTheBadgeShowsTheTemporaryHpAsItsOwnParcel(t *testing.T) {
	f, id := barbaro(t, 5)
	const marker = "PV temporários — o dano gasta estes primeiro (p106)"

	if page := powerScreen(t, f, id); strings.Contains(page, marker) {
		t.Fatal("a ficha SEM poça já mostra a reserva — o caso mediria o repouso")
	}

	// A fração do PV é LIDA antes, e não escrita à mão: o poço do bárbaro vem do
	// catálogo (ALE-355), e o que este caso afirma é que ela não MUDA.
	fraction := pvFraction(t, f, id)

	if rec := effect(t, f, id, "aplica/campo-de-forca"); rec.Code != http.StatusOK {
		t.Fatalf("aplicar o Campo de Força devolveu %d", rec.Code)
	}

	screen := powerScreen(t, f, id)
	if !strings.Contains(screen, marker) {
		t.Error("a reserva não chegou ao crachá")
	}
	// O número é do LIVRO e escrito à mão: o Campo de Força dá 30 PV
	// temporários de cena.
	if !strings.Contains(screen, ">+30</span>") {
		t.Error("o crachá não diz QUANTO é a reserva")
	}
	// E o PV de verdade não se mexeu: a reserva é parcela à parte, não um
	// somando.
	if !strings.Contains(screen, fraction) {
		t.Errorf("a fração do PV saiu de %q — a reserva virou somando em vez de parcela", fraction)
	}
}

// pvFraction é o "atual/máximo" do PV como o crachá o escreve.
func pvFraction(t *testing.T, f sceneFixture, id int64) string {
	t.Helper()
	pool := poolsOf(t, f.s, id)
	return fmt.Sprintf("%d/%d", pool.HpCurrent, pool.HpMax)
}
