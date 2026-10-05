package api

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// O TESTE DE PERÍCIA, DA FICHA ATÉ A MESA (ALE-423, p220-221).
//
//	"Um teste é uma rolagem de 1d20 + um modificador." (p220)
//	"Um 20 natural sempre é um sucesso, e um 1 natural sempre é uma falha, não
//	 importando o valor a ser alcançado." (p221)
//
// INTEGRAÇÃO porque o que pode quebrar é a COMPOSIÇÃO, e ela atravessa duas
// cenas que não se conhecem: o gesto sai da FICHA, o modificador vem da ficha
// COMPUTADA, a regra é do motor, e o resultado tem de aparecer na MESA — por uma
// PORTA, porque a ficha não conhece a mesa e não deve conhecer.
//
// Um teste do `ResolveSkillTest` prova a aritmética e nada sobre o número chegar
// na faixa que a mesa lê.

// umaMesaAoVivo é a bancada com a sessão em curso.
//
// A SESSÃO TEM DE ESTAR ATIVA, e isto não é detalhe de bancada: `active` é a
// única situação que conta como mesa EM CURSO — o `HasLiveSessionForCharacter`
// diz isso por escrito, e `planned` e `ended` não são mesa acontecendo. O
// `seedSession` cria planejada, então quem quer mesa viva a começa.
func umaMesaAoVivo(t *testing.T) sceneFixture {
	t.Helper()
	f := newSceneFixture(t)
	if _, err := f.s.queries.StartSessionFresh(context.Background(), sqlcgen.StartSessionFreshParams{
		StartedAt: sql.NullString{String: dbvalue.NowISO(), Valid: true},
		UpdatedAt: dbvalue.NowISO(), ID: f.sessionID,
	}); err != nil {
		t.Fatalf("começar a sessão: %v", err)
	}
	return f
}

// rolaPericia bate no gesto da ficha. `d20` zero quer dizer "role por mim", que
// é a convenção do sinal.
func rolaPericia(t *testing.T, f sceneFixture, pericia string, d20 int) string {
	t.Helper()
	rec := f.requests(t, f.player, http.MethodPost,
		"/personagens/"+strconv.FormatInt(f.charID, 10)+"/pericias/rolar/"+pericia,
		`{"expertise_d20":`+strconv.Itoa(d20)+`}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("rolar %q deu %d", pericia, rec.Code)
	}
	return rec.Body.String()
}

// O D20 QUE A MESA ROLOU chega inteiro à faixa, e o TOTAL é ele mais o valor da
// perícia — que vem da ficha computada e não de uma conta refeita aqui.
//
// O 14 é escolhido para NÃO ser natural de nada: é o caso comum, e é o que
// separa "a conta funciona" de "o selo está sempre aceso".
func TestTheDieTheTableRolledReachesTheBandWithTheSheetModifier(t *testing.T) {
	f := umaMesaAoVivo(t)
	rolaPericia(t, f, "Iniciativa", 14)

	estado := stateOf(t, f.s.sessions, f.sessionID)
	teste := estado.LastSkillTest
	if teste == nil {
		t.Fatalf("o teste não chegou à mesa")
	}
	if teste.Roll != 14 {
		t.Errorf("a faixa diz d20 %d, e a mesa rolou 14", teste.Roll)
	}
	if teste.Total != 14+teste.Modifier {
		t.Errorf("o total veio %d, e 14 + %d é %d — a conta não fecha",
			teste.Total, teste.Modifier, 14+teste.Modifier)
	}
	if teste.Skill != "Iniciativa" {
		t.Errorf("a faixa nomeia a perícia %q", teste.Skill)
	}
	// DE ONDE VEIO O DADO é o que separa confiar no app de confiar no dado da
	// mesa, e a faixa tem de dizer.
	if !teste.ByHand {
		t.Errorf("o d20 veio da mesa e a faixa não marcou")
	}
	if teste.Natural20 || teste.Natural1 {
		t.Errorf("um d20 14 veio marcado como natural: %+v", teste)
	}

	// O MODIFICADOR VEM DA FICHA, e esta é a metade que a asserção acima NÃO
	// prende: `Total == 14 + Modifier` é auto-consistente e passaria com um
	// modificador cravado. O que prova a procedência é a ficha MUDAR e o número
	// seguir — treinar a perícia sobe o valor, e o teste seguinte tem de subir
	// junto.
	//
	// Pelo DELTA e não por um número escrito à mão: o bônus de treino depende do
	// nível, e escrevê-lo aqui seria reimplementar a regra que a ficha já computa
	// — o erro que o `CLAUDE.md` chama de derivar o esperado do código sob teste.
	antes := teste.Modifier
	if rec := f.requests(t, f.player, http.MethodPost,
		"/personagens/"+strconv.FormatInt(f.charID, 10)+"/pericias/treino/Iniciativa", ""); rec.Code != http.StatusOK {
		t.Fatalf("treinar a perícia deu %d", rec.Code)
	}
	rolaPericia(t, f, "Iniciativa", 14)
	depois := stateOf(t, f.s.sessions, f.sessionID).LastSkillTest
	if depois.Modifier <= antes {
		t.Errorf("a perícia foi TREINADA e o modificador do teste ficou em %d, vindo de %d "+
			"— o número não está saindo da ficha", depois.Modifier, antes)
	}
	if depois.Total != depois.Roll+depois.Modifier {
		t.Errorf("a conta parou de fechar depois do treino: %+v", depois)
	}
}

// O ZERO É "ROLE POR MIM", e é a convenção dos sinais numéricos da ficha.
//
// O que se afirma aqui é que o servidor rolou um dado VÁLIDO e que a faixa NÃO
// diz "rolado na mesa" — um caminho que caísse no zero seria recusado pelo
// motor, e um que marcasse `ByHand` mentiria sobre a procedência.
func TestTheZeroMeansTheServerRollsAndTheBandDoesNotClaimTheTable(t *testing.T) {
	f := umaMesaAoVivo(t)
	corpo := rolaPericia(t, f, "Iniciativa", 0)

	estado := stateOf(t, f.s.sessions, f.sessionID)
	teste := estado.LastSkillTest
	if teste == nil {
		t.Fatalf("o teste não chegou à mesa; a resposta foi:\n%s", corpo[:min(len(corpo), 400)])
	}
	if teste.Roll < 1 || teste.Roll > 20 {
		t.Errorf("o servidor rolou %d, e um d20 vai de 1 a 20", teste.Roll)
	}
	if teste.ByHand {
		t.Errorf("o servidor rolou e a faixa disse que o dado veio da mesa")
	}
}

// O D20 INVÁLIDO É RECUSADO, e a frase fala do dado.
//
// Esta porta é mais larga que a do ataque — ela ACEITA o que a mesa digitou —, e
// por isso a conferência importa mais: um 40 digitado viraria um total que
// ninguém consegue explicar.
func TestADieTheTableCouldNotHaveRolledIsRefused(t *testing.T) {
	f := umaMesaAoVivo(t)
	corpo := rolaPericia(t, f, "Iniciativa", 40)

	// A RECUSA DA FICHA volta DENTRO da cena redesenhada e não num sinal: o
	// cliente do Datastar não aplica remendo que não é 2xx, então um erro HTTP
	// deixaria o gesto sem acontecer e sem dizer por quê. Procurar `command_error`
	// aqui seria ler o canal da MESA, que é outra cena.
	if !strings.Contains(corpo, "d20") {
		t.Errorf("o d20 40 não foi recusado com uma frase sobre o dado")
	}
	if estado := stateOf(t, f.s.sessions, f.sessionID); estado.LastSkillTest != nil {
		t.Errorf("a recusa deixou um teste na mesa: %+v", estado.LastSkillTest)
	}
}

// A PERÍCIA QUE A FICHA NÃO TEM é recusada pelo nome — e este caso existe porque
// o nome vem do CAMINHO, que é cliente.
func TestRollingASkillTheSheetDoesNotHaveIsRefusedByName(t *testing.T) {
	f := umaMesaAoVivo(t)
	corpo := rolaPericia(t, f, "Pilotagem", 10)

	if !strings.Contains(corpo, "Pilotagem") {
		t.Errorf("a recusa não citou a perícia pedida")
	}
	if estado := stateOf(t, f.s.sessions, f.sessionID); estado.LastSkillTest != nil {
		t.Errorf("a recusa deixou um teste na mesa: %+v", estado.LastSkillTest)
	}
}

// A MESA VÊ A FAIXA, e é por isso que o gesto existe: o teste é de quem rola e a
// mesa inteira lê.
func TestTheWholeTableSeesTheRolledTest(t *testing.T) {
	f := umaMesaAoVivo(t)
	rolaPericia(t, f, "Iniciativa", 20)

	para := func(quem int64) string {
		return f.requests(t, quem, http.MethodGet, f.tableUrl(), "").Body.String()
	}
	for _, caso := range []struct {
		nome string
		html string
	}{
		{"o mestre", para(f.gm)},
		{"o jogador", para(f.player)},
	} {
		if !strings.Contains(caso.html, "20 natural") {
			t.Errorf("%s não viu o selo do 20 natural na faixa", caso.nome)
		}
		if !strings.Contains(caso.html, "Iniciativa") {
			t.Errorf("%s não viu a perícia na faixa", caso.nome)
		}
	}
}

// A MESA É ACHADA PELA CAMPANHA, e o id da campanha NÃO é o id da linha de
// elenco (ALE-423).
//
// O `ListCampaignsForCharacter` devolve a linha de `campaign_members` com o
// `m.id` no campo `ID` e a campanha no `Campaignid`. Quem lesse `ID` pediria as
// sessões de uma campanha que é outra — ou de nenhuma —, e o teste rolado
// simplesmente não chegaria à mesa: sem erro, sem recusa, sem faixa.
//
// POR QUE A BANCADA COMUM NÃO PEGA ISSO: nela a campanha é a 1 e a linha de
// elenco também é a 1, então ler o campo errado dá o número certo. O defeito
// apareceu no banco de desenvolvimento, onde um jogador está em DUAS campanhas —
// e foi OLHANDO a tela depois de mesclar, não na suíte.
//
// A bancada daqui força a divergência: um segundo personagem entra na primeira
// campanha, e só então o nosso entra na segunda. A sessão VIVA é a da segunda.
func TestTheRolledTestReachesTheTableWhenTheMembershipIdIsNotTheCampaignId(t *testing.T) {
	f := newSceneFixture(t)

	// Outro personagem na campanha 1 empurra o número da linha de elenco.
	outro := seedCharacterAtLevel(t, f.s, f.player, "Figurante", "Guerreiro", 1, 0, 0)
	seedMember(t, f.s, f.campaignID, outro)

	segunda := seedCampaign(t, f.s, f.gm)
	aoVivo := seedSession(t, f.s, segunda)
	seedMember(t, f.s, segunda, f.charID)
	if _, err := f.s.queries.StartSessionFresh(context.Background(), sqlcgen.StartSessionFreshParams{
		StartedAt: sql.NullString{String: dbvalue.NowISO(), Valid: true},
		UpdatedAt: dbvalue.NowISO(), ID: aoVivo,
	}); err != nil {
		t.Fatalf("começar a segunda sessão: %v", err)
	}

	// O CONTROLE: se os dois ids coincidirem, ler o campo errado dá o número
	// certo e o caso não mede nada — que é exatamente o que acontecia antes.
	linhas, err := f.s.queries.ListCampaignsForCharacter(context.Background(), f.charID)
	if err != nil {
		t.Fatalf("listar as campanhas do personagem: %v", err)
	}
	divergiu := false
	for _, linha := range linhas {
		if linha.Campaignid == segunda && linha.ID != linha.Campaignid {
			divergiu = true
		}
	}
	if !divergiu {
		t.Fatalf("a bancada não fez os ids divergirem, e o caso não mede nada: %+v", linhas)
	}

	rolaPericia(t, f, "Iniciativa", 14)

	estado := stateOf(t, f.s.sessions, aoVivo)
	if estado.LastSkillTest == nil {
		t.Fatalf("o teste não chegou à mesa VIVA (sessão %d da campanha %d) — a busca "+
			"pela sessão leu o id da linha de elenco no lugar do id da campanha",
			aoVivo, segunda)
	}
	if estado.LastSkillTest.Roll != 14 {
		t.Errorf("a faixa da mesa viva diz d20 %d, e a mesa rolou 14", estado.LastSkillTest.Roll)
	}
}
