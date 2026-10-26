package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// O PM PAGO NO TESTE DE PERÍCIA, DO DIÁLOGO ATÉ A MESA (ALE-489).
//
// O gesto é o de sempre — rolar a perícia —, com os gastos marcados no sinal
// `expertise_spends`. O que se prende é a COMPOSIÇÃO: a oferta vem da ficha, o
// bônus entra na conta que a faixa lê, e o PM sai da ficha ao rolar (decisão do
// dono: o teste não tem confirmação do mestre).

func spenderAtTheTable(t *testing.T, class string, level, charisma int64) (sceneFixture, int64) {
	t.Helper()
	f := umaMesaAoVivo(t)
	id, err := f.s.queries.CreateCharacter(context.Background(), sqlcgen.CreateCharacterParams{
		OwnerId: f.player, Name: "Gastadora", Origin: "soldado", Level: level, Charisma: charisma,
		Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: "[]", ClassPowers: "[]", ClassChoices: "{}", PowerChoices: "{}",
		CreatedAt: dbvalue.NowISO(), UpdatedAt: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear a ficha: %v", err)
	}
	seedClasse(t, f.s, id, class, level)
	seedMember(t, f.s, f.campaignID, id)
	seedPericia(t, f.s, id, "Diplomacia", "charisma", false)
	seedPericia(t, f.s, id, "Misticismo", "intelligence", false)
	f.charID = id
	return f, id
}

func rollsWith(t *testing.T, f sceneFixture, id int64, skill, body string) string {
	t.Helper()
	rec := f.requests(t, f.player, http.MethodPost,
		"/personagens/"+strconv.FormatInt(id, 10)+"/pericias/rolar/"+skill, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("rolar %s deu %d", skill, rec.Code)
	}
	return sceneRefusal(rec.Body.String())
}

// p47 "Quando faz um teste de perícia, você pode gastar 2 PM para somar seu
// Carisma no teste." Carisma 3: o mesmo d20 dá +3, e a ficha perde 2 PM.
func TestAudacityAddsCharismaToTheTestAndChargesTwoPm(t *testing.T) {
	f, id := spenderAtTheTable(t, "Bucaneiro", 5, 3)

	if refusal := rollsWith(t, f, id, "Diplomacia", `{"expertise_d20":10}`); refusal != "" {
		t.Fatalf("o teste sem gasto foi recusado: %q", refusal)
	}
	plain := stateOf(t, f.s.sessions, f.sessionID).LastSkillTest
	before := pm(t, f, id)

	if refusal := rollsWith(t, f, id, "Diplomacia", `{"expertise_d20":10,"expertise_spends":{"audacia":2}}`); refusal != "" {
		t.Fatalf("o teste com Audácia foi recusado: %q", refusal)
	}
	paid := stateOf(t, f.s.sessions, f.sessionID).LastSkillTest
	if paid.Total-plain.Total != 3 {
		t.Errorf("com Audácia o total foi %d, sem ela %d — want +3 (Carisma 3)", paid.Total, plain.Total)
	}
	if len(paid.Spends) != 1 || paid.Spends[0].Name != "Audácia" || paid.Spends[0].PM != 2 {
		t.Errorf("a faixa não diz o que foi pago: %+v", paid.Spends)
	}
	if spent := before - pm(t, f, id); spent != 2 {
		t.Errorf("a Audácia cobrou %d PM, e a p47 diz 2", spent)
	}

	// O VALOR QUE NÃO É NÚMERO não derruba a leitura dos sinais: o d20 da mesa
	// continua chegando, e o gasto vira "não pedido".
	if refusal := rollsWith(t, f, id, "Diplomacia", `{"expertise_d20":10,"expertise_spends":{"audacia":""}}`); refusal != "" {
		t.Fatalf("um gasto vazio derrubou o teste: %q", refusal)
	}
	if got := stateOf(t, f.s.sessions, f.sessionID).LastSkillTest; got.Roll != 10 || len(got.Spends) != 0 {
		t.Errorf("com o gasto vazio a faixa ficou d20 %d e gastos %+v; want d20 10 e nada pago", got.Roll, got.Spends)
	}
}

// p45 Eclético: "gastar 1 PM para receber os benefícios de ser treinado nessa
// perícia para este teste". Misticismo é só para treinados (p115): sem o gasto
// a perícia não se rola, com ele rola.
func TestEclecticLetsTheBardRollATrainedOnlySkill(t *testing.T) {
	f, id := spenderAtTheTable(t, "Bardo", 3, 0)
	if refusal := rollsWith(t, f, id, "Misticismo", `{"expertise_d20":10}`); !strings.Contains(refusal, "treinamento") {
		t.Fatalf("Misticismo sem treino e sem gasto tinha de ser recusado pelo treino, e veio %q", refusal)
	}
	if refusal := rollsWith(t, f, id, "Misticismo", `{"expertise_d20":10,"expertise_spends":{"ecletico":1}}`); refusal != "" {
		t.Fatalf("Misticismo com Eclético foi recusado: %q", refusal)
	}
	if got := stateOf(t, f.s.sessions, f.sessionID).LastSkillTest; got == nil || got.Skill != "Misticismo" {
		t.Errorf("o teste com Eclético não chegou à mesa: %+v", got)
	}
}
