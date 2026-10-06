package api

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"t20engine/domain/live"
)

// O BÔNUS CUMULATIVO DE CENA (ALE-423, p42).
//
//	"Enquanto está em fúria, quando faz um acerto crítico ou reduz um inimigo a
//	 0 PV, você recebe um bônus cumulativo de +1 em testes de ataque e rolagens
//	 de dano, limitado pelo seu nível, até o fim da cena." (p42)
//
// A Sangue dos Inimigos era LETRA MORTA: o catálogo a descrevia, a ficha a
// listava, e nenhum número se mexia — ela não tinha `modifiers`, e não podia
// ter: o `Modifier` descreve bônus estático ou condicional, e este SOBE a cada
// gatilho.
//
// INTEGRAÇÃO, e pela razão de sempre: o que se prende é a ligação entre o gesto
// do MESTRE (confirmar o ataque), o estado da FICHA (o efeito de cena) e o
// NÚMERO que a pessoa lê. Um teste de unidade do teto provaria aritmética e
// nada sobre o +1 chegar ao ataque.
//
// O D20 NÃO É SORTEADO AQUI. O provisório é semeado com `Critical: true` pelo
// mesmo `ProposeAttack` que a cena usa, porque o que este arquivo mede é o que
// acontece DEPOIS do crítico — rolar até sair um 20 seria um teste que falha
// por azar.

// attackValue acha o NÚMERO GRANDE da linha, pela classe que só ele tem.
//
// A primeira versão era `Atacar · …([+-]\d+)</span>` com `.*?` no meio, e ela
// casava com o `+5` de `1d6+5` — o DANO do detalhe, que não se mexe quando o
// bônus sobe. O caso teria passado verde sobre o número errado, que é a
// armadilha que o `CLAUDE.md` chama de instrumento com cara de resultado.
var attackValue = regexp.MustCompile(`tabular-nums text-foreground">([+-]\d+)</span>`)

// ragingBarbarianWithTheBloodPower é a bancada: o bárbaro de nível 6, com a
// Sangue dos Inimigos escolhida e EM FÚRIA, num combate com um Goblin.
//
// O poder é ESCOLHIDO pelo gesto da ficha e não por um `UPDATE`: ele é poder de
// vaga, e gravar a coluna à mão montaria uma ficha que o app não sabe produzir.
func ragingBarbarianWithTheBloodPower(t *testing.T) (sceneFixture, int64, string, string) {
	t.Helper()
	f, barbaro, goblin := barbarianOnTurn(t)
	if refused := powerCommand(t, f, barbaro, "escolhe/class.barbaro.sangue-dos-inimigos", ""); refused != "" {
		t.Fatalf("escolher a Sangue dos Inimigos: %s", refused)
	}
	if refused := powerCommand(t, f, barbaro, "postura/furia/entra", ""); refused != "" {
		t.Fatalf("entrar em fúria: %s", refused)
	}
	// O CONTROLE DA BANCADA: a fúria está MESMO acesa. Sem ele, um gesto que
	// falhasse em silêncio deixaria o caso "fora da fúria não acumula" medindo
	// um bárbaro que nunca entrou nela — verde sobre nada.
	if !strings.Contains(superficieDeAcoes(t, acoesDaMesa(t, f)), "Encerrar Fúria") {
		t.Fatal("a bancada não entrou em fúria: a superfície não oferece Encerrar Fúria")
	}
	// A VEZ PASSA PARA O GOBLIN, e isso é instrumento e não arranjo: confirmar
	// o ataque de quem está NA VEZ cobra a ação padrão (p233), então o segundo
	// crítico seria recusado por falta de ação — e a recusa volta num SINAL com
	// status 200, que é exatamente o silêncio que esconderia o defeito.
	// Confirmar fora da vez é caminho de produção: *"o mestre confirma fora de
	// hora o tempo todo"*.
	atacante := entryLabeled(t, stateOf(t, f.s.sessions, f.sessionID), "Furioso")
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/proxima-vez", ""); rec.Code != http.StatusOK {
		t.Fatalf("girar a vez deu %d", rec.Code)
	}
	return f, barbaro, goblin, atacante
}

// endsTheRage encerra a postura pela rota de verdade.
//
// A rota é a dos EFEITOS e não a dos poderes — `/efeitos/postura/{flag}` —, e
// escrevê-la por extenso aqui é deliberado: a primeira versão deste arquivo
// montava `"../efeitos/postura/furia"` em cima do helper dos poderes, o
// caminho não casava rota nenhuma, a fúria nunca terminava, e DOIS casos
// afirmavam o contrário do que mediam.
func endsTheRage(t *testing.T, f sceneFixture, barbaro int64) {
	t.Helper()
	rec := f.requests(t, f.player, http.MethodPost,
		fmt.Sprintf("/personagens/%d/efeitos/postura/furia?embutida=1", barbaro), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("encerrar a fúria deu %d", rec.Code)
	}
	if refusal := sceneRefusal(rec.Body.String()); refusal != "" {
		t.Fatalf("encerrar a fúria foi recusado: %s", refusal)
	}
	if strings.Contains(superficieDeAcoes(t, acoesDaMesa(t, f)), "Encerrar Fúria") {
		t.Fatal("a fúria continua acesa depois do gesto de encerrar")
	}
}

// attackOnTheActionsSurface é o número que o jogador LÊ na linha de Atacar.
//
// Pela tela e não pelo motor: o que esta fatia promete é que o +1 chega ao
// número que a pessoa usa para rolar, e ler o `ComputeSheet` aqui provaria a
// conta contra ela mesma.
func attackOnTheActionsSurface(t *testing.T, f sceneFixture) int {
	t.Helper()
	// RECORTADO À LINHA DE ATACAR: o grupo inteiro tem Manobra, Fintar e mais,
	// e o primeiro número grande da superfície pode não ser o do ataque.
	linha := entre(t, superficieDeAcoes(t, acoesDaMesa(t, f)), "Atacar · ", "</li>")
	found := attackValue.FindStringSubmatch(linha)
	if found == nil {
		t.Fatalf("não achei o número da linha de Atacar:\n%s", recorte(linha))
	}
	n, err := strconv.Atoi(strings.TrimPrefix(found[1], "+"))
	if err != nil {
		t.Fatalf("o número da linha de Atacar não é um número: %q", found[1])
	}
	return n
}

// theGameMasterConfirmsACriticalHit semeia o provisório crítico e confirma.
//
// PELO GESTO DO MESTRE, que é a divisa inteira: o bônus sobe quando o ataque é
// CONFIRMADO, não quando é proposto — um provisório cancelado não aconteceu.
func theGameMasterConfirmsACriticalHit(t *testing.T, f sceneFixture, attacker, target string) {
	t.Helper()
	if _, err := f.s.sessions.ProposeAttack(context.Background(), f.sessionID, live.PendingAttack{
		AttackerEntryID: attacker, TargetEntryID: target, TargetLabel: "Goblin",
		Weapon: "Machado de batalha", Roll: 20, Total: 28, Defense: 13,
		Hit: true, Critical: true, RawDamage: 1, Damage: 1,
	}); err != nil {
		t.Fatalf("semear o provisório crítico: %v", err)
	}
	rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/confirmar", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("confirmar o crítico deu %d", rec.Code)
	}
	// A RECUSA VOLTA COM 200, num sinal. Sem esta linha, um confirmar barrado
	// pela economia de ação passaria por "aconteceu" e o caso mediria o nada.
	if refusal := tableRefusal(t, rec.Body.String()); refusal != "" {
		t.Fatalf("confirmar o crítico foi recusado: %s", refusal)
	}
}

// CADA CRÍTICO CONFIRMADO SOBE UM, e é a fatia inteira numa frase.
func TestEveryConfirmedCriticalRaisesTheCumulativeBonus(t *testing.T) {
	f, _, goblin, attacker := ragingBarbarianWithTheBloodPower(t)
	before := attackOnTheActionsSurface(t, f)

	theGameMasterConfirmsACriticalHit(t, f, attacker, goblin)
	if got := attackOnTheActionsSurface(t, f); got != before+1 {
		t.Fatalf("o primeiro crítico levou o ataque de %+d para %+d, e o livro dá +1", before, got)
	}
	// O SEGUNDO é o que separa "cumulativo" de "um bônus que liga": com uma
	// gravação que SUBSTITUI em vez de somar, o caso de cima passa verde e este
	// reprova.
	theGameMasterConfirmsACriticalHit(t, f, attacker, goblin)
	if got := attackOnTheActionsSurface(t, f); got != before+2 {
		t.Errorf("o segundo crítico tinha de levar a %+d, e levou a %+d", before+2, got)
	}
}

// O TETO É O NÍVEL — *"limitado pelo seu nível"* (p42).
//
// O bárbaro da bancada é de nível 6, então o sétimo crítico não move nada.
func TestTheCumulativeBonusStopsAtTheCharacterLevel(t *testing.T) {
	f, _, goblin, attacker := ragingBarbarianWithTheBloodPower(t)
	before := attackOnTheActionsSurface(t, f)

	for range 6 {
		theGameMasterConfirmsACriticalHit(t, f, attacker, goblin)
	}
	noTeto := attackOnTheActionsSurface(t, f)
	if noTeto != before+6 {
		t.Fatalf("seis críticos tinham de dar +6 e deram %+d", noTeto-before)
	}
	theGameMasterConfirmsACriticalHit(t, f, attacker, goblin)
	if got := attackOnTheActionsSurface(t, f); got != noTeto {
		t.Errorf("o sétimo crítico passou do teto de nível: %+d virou %+d", noTeto, got)
	}
}

// FORA DA FÚRIA NÃO ACUMULA: *"enquanto está em fúria"* (p42).
//
// É a metade que separa este poder de um bônus que todo bárbaro teria sempre.
func TestNoCumulativeBonusIsEarnedOutsideTheStance(t *testing.T) {
	f, barbaro, goblin, attacker := ragingBarbarianWithTheBloodPower(t)
	endsTheRage(t, f, barbaro)
	before := attackOnTheActionsSurface(t, f)

	theGameMasterConfirmsACriticalHit(t, f, attacker, goblin)
	if got := attackOnTheActionsSurface(t, f); got != before {
		t.Errorf("fora da fúria o crítico acumulou assim mesmo: %+d virou %+d", before, got)
	}
}

// E O QUE FOI GANHO ACABA COM A FÚRIA (decisão do dono).
//
// O texto da p42 diz "até o fim da cena", e o dono leu a linha inteira como
// condicionada ao *"enquanto está em fúria"*: sair da fúria devolve o bárbaro
// aos números dele. O `scope: scene` do efeito fica como REDE, para a cena que
// termina com alguém ainda furioso.
func TestLeavingTheStanceTakesTheCumulativeBonusWithIt(t *testing.T) {
	f, barbaro, goblin, attacker := ragingBarbarianWithTheBloodPower(t)
	furiosoESemBonus := attackOnTheActionsSurface(t, f)
	theGameMasterConfirmsACriticalHit(t, f, attacker, goblin)
	// O CONTROLE: o bônus existia antes de a fúria acabar. Sem ele, "o número
	// voltou" passaria verde num caso em que ele nunca subiu.
	if got := attackOnTheActionsSurface(t, f); got != furiosoESemBonus+1 {
		t.Fatalf("o crítico não acumulou, então este caso não mede nada: %+d", got)
	}

	// SAIR E VOLTAR, e não só sair: comparar com o número de FORA da fúria
	// misturaria duas coisas num só delta — a fúria vale um bônus próprio, e o
	// caso afirmaria que ela some junto, que é outra regra. Voltando à fúria, o
	// único que pode ter sobrado é o cumulativo.
	endsTheRage(t, f, barbaro)
	if refused := powerCommand(t, f, barbaro, "postura/furia/entra", ""); refused != "" {
		t.Fatalf("voltar à fúria: %s", refused)
	}
	if got := attackOnTheActionsSurface(t, f); got != furiosoESemBonus {
		t.Errorf("a nova fúria começou com %+d em vez de %+d: o bônus sobreviveu à anterior",
			got, furiosoESemBonus)
	}
}

// QUEM NÃO TEM O PODER NÃO ACUMULA — o controle da família inteira.
//
// Sem ele, uma implementação que subisse o bônus de QUALQUER crítico passaria
// em todos os casos acima.
func TestNoCumulativeBonusReachesACharacterWithoutThePower(t *testing.T) {
	f, barbaro, goblin := barbarianOnTurn(t)
	if refused := powerCommand(t, f, barbaro, "postura/furia/entra", ""); refused != "" {
		t.Fatalf("entrar em fúria: %s", refused)
	}
	attacker := entryLabeled(t, stateOf(t, f.s.sessions, f.sessionID), "Furioso")
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/proxima-vez", ""); rec.Code != http.StatusOK {
		t.Fatalf("girar a vez deu %d", rec.Code)
	}
	before := attackOnTheActionsSurface(t, f)

	theGameMasterConfirmsACriticalHit(t, f, attacker, goblin)
	if got := attackOnTheActionsSurface(t, f); got != before {
		t.Errorf("um bárbaro SEM a Sangue dos Inimigos acumulou: %+d virou %+d", before, got)
	}
}

// theGameMasterConfirmsAPlainHit confirma um golpe COMUM — sem crítico — de
// `damage` de dano contra um alvo de PV conhecido.
func theGameMasterConfirmsAPlainHit(t *testing.T, f sceneFixture, attacker, target string, damage int) {
	t.Helper()
	if _, err := f.s.sessions.ProposeAttack(context.Background(), f.sessionID, live.PendingAttack{
		AttackerEntryID: attacker, TargetEntryID: target, TargetLabel: "Bicho",
		Weapon: "Machado de batalha", Roll: 11, Total: 19, Defense: 13,
		Hit: true, Critical: false, RawDamage: damage, Damage: damage,
	}); err != nil {
		t.Fatalf("semear o provisório: %v", err)
	}
	rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/confirmar", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("confirmar o golpe deu %d", rec.Code)
	}
	if refusal := tableRefusal(t, rec.Body.String()); refusal != "" {
		t.Fatalf("confirmar o golpe foi recusado: %s", refusal)
	}
}

// aBeastWithHitPoints põe na fila um alvo de PV RASTREADO.
//
// O Goblin da bancada do ataque entra sem PV, e sem PV não há como reduzir
// ninguém a zero: o caso mediria o gatilho do crítico de novo.
func aBeastWithHitPoints(t *testing.T, f sceneFixture, pv int64) string {
	t.Helper()
	hp := pv
	if _, err := f.s.sessions.AddInitiativeEntry(context.Background(), f.sessionID,
		live.InitiativeEntry{
			Label: "Bicho", Initiative: 1, Type: "npc", HpCurrent: &hp, HpMax: &hp,
		}); err != nil {
		t.Fatalf("pôr o Bicho na fila: %v", err)
	}
	return entryLabeled(t, stateOf(t, f.s.sessions, f.sessionID), "Bicho")
}

// DERRUBAR UM INIMIGO TAMBÉM ACUMULA — é a segunda metade da p42, e ela não
// depende de crítico nenhum.
//
// Sem este caso, uma implementação que lesse só o `Critical` passaria em tudo
// o que está acima.
func TestDroppingAnEnemyToZeroRaisesTheCumulativeBonus(t *testing.T) {
	f, _, _, attacker := ragingBarbarianWithTheBloodPower(t)
	bicho := aBeastWithHitPoints(t, f, 4)
	before := attackOnTheActionsSurface(t, f)

	// O CONTROLE VEM PRIMEIRO: uma pancada que NÃO derruba não acumula. Ele é o
	// que separa "derrubou" de "acertou", e sem ele o caso abaixo passaria
	// verde numa implementação que subisse o bônus em todo acerto.
	theGameMasterConfirmsAPlainHit(t, f, attacker, bicho, 1)
	if got := attackOnTheActionsSurface(t, f); got != before {
		t.Fatalf("um acerto que não derrubou acumulou: %+d virou %+d", before, got)
	}

	theGameMasterConfirmsAPlainHit(t, f, attacker, bicho, 99)
	if got := attackOnTheActionsSurface(t, f); got != before+1 {
		t.Errorf("derrubar o inimigo tinha de levar a %+d, e levou a %+d", before+1, got)
	}
}
