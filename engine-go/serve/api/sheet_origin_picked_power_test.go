package api

import (
	"context"
	"strings"
	"testing"
)

// O PODER QUE A ORIGEM ESCOLHEU É DA FICHA, PARA TUDO. A Capanga dá "um poder de combate a
// sua escolha" (p91), e o motor soma os modificadores do escolhido; mas a lista de poderes da
// ficha só registrava o benefício "escolha um poder", e não o poder dentro dele. O Usar, que
// pergunta a essa lista, recusava com "o personagem não tem este poder".
func TestThePowerTheOriginPickedIsOwnedForEverything(t *testing.T) {
	f, id := barbaro(t, 5)
	if _, err := f.s.db.ExecContext(context.Background(),
		`UPDATE characters SET origin = 'capanga', originChoices = '["origin-capanga-poder-poder-de-combate-escolha"]' WHERE id = ?`, id); err != nil {
		t.Fatalf("semear a origem: %v", err)
	}
	seedPowerChoices(t, f.s, id, "[]", `{"origin-capanga-poder-poder-de-combate-escolha":["bloqueio-com-escudo"]}`)

	// NA LISTA DE AÇÕES da aba, e não em qualquer lugar da página: o catálogo do Escolher
	// também escreve o nome de todo poder de combate.
	_, actions, ok := strings.Cut(sheetTabWith(t, f, id, "abilities", `{}`), ">Ações<")
	if !ok {
		t.Fatal("a aba Poderes não tem a lista de Ações")
	}
	actions, _, _ = strings.Cut(actions, "Passivas")
	if !strings.Contains(actions, "Bloqueio com Escudo") {
		t.Error("a lista de Ações não tem o Bloqueio com Escudo que a origem escolheu")
	}
	before := mpOf(t, f.s, id)
	if refusal := powerCommand(t, f, id, "usa/bloqueio-com-escudo", ""); refusal != "" {
		t.Fatalf("o Usar recusou o poder que a origem escolheu: %q", refusal)
	}
	if spent := before - mpOf(t, f.s, id); spent != 1 {
		t.Errorf("o Bloqueio com Escudo cobrou %d PM, e custa 1", spent)
	}
}

// A ESCOLHA DO PODER DA ORIGEM TEM TELA. O Escolher ligava "Poder de Combate (escolha)" da
// Capanga sem perguntar qual poder, e nenhuma rota gravava a escolha: a validação estrita
// recusava a chave do benefício no `powerChoices` ("não pede escolha"). Agora o benefício
// marcado ganha um passo no Escolher com os poderes do tipo dele, e a rota grava o escolhido.

const capangaPick = "origin-capanga-poder-poder-de-combate-escolha"

func capangaBarbarian(t *testing.T) (sceneFixture, int64) {
	t.Helper()
	f, id := barbaro(t, 5)
	if _, err := f.s.db.ExecContext(context.Background(),
		`UPDATE characters SET origin = 'capanga', originChoices = '[]' WHERE id = ?`, id); err != nil {
		t.Fatalf("semear a origem: %v", err)
	}
	if refusal := powerCommand(t, f, id, "origem/"+capangaPick, ""); refusal != "" {
		t.Fatalf("marcar o benefício da Capanga: %q", refusal)
	}
	return f, id
}

func TestTheChoicesOfferThePowerTheOriginLetsPick(t *testing.T) {
	f, id := capangaBarbarian(t)
	page := sheetTabWith(t, f, id, "abilities", `{"sheet_dialog":"power-choices"}`)
	_, step, ok := strings.Cut(page, `id="choice-step-origin-pick-`+capangaPick+`"`)
	if !ok {
		t.Fatal("o Escolher não tem o passo do poder que a Capanga deixa escolher")
	}
	step, _, _ = strings.Cut(step, `id="choice-step-`)
	if !strings.Contains(step, "/poderes/origem/"+capangaPick+"/poder/bloqueio-com-escudo") {
		t.Error("o passo não oferece o Bloqueio com Escudo, que é poder de combate")
	}
	if strings.Contains(step, "/poder/acrobatico") {
		t.Error("o passo oferece o Acrobático, que é poder de destino e não de combate")
	}
}

func TestPickingThePowerTheOriginLetsPickStoresItAndTheSheetUsesIt(t *testing.T) {
	f, id := capangaBarbarian(t)
	if refusal := powerCommand(t, f, id, "origem/"+capangaPick+"/poder/bloqueio-com-escudo", ""); refusal != "" {
		t.Fatalf("escolher o Bloqueio com Escudo foi recusado: %q", refusal)
	}
	if _, picks := storedChoices(t, f.s, id); !strings.Contains(picks, `"`+capangaPick+`":["bloqueio-com-escudo"]`) {
		t.Fatalf("a escolha não foi gravada no benefício: %s", picks)
	}
	if refusal := powerCommand(t, f, id, "usa/bloqueio-com-escudo", ""); refusal != "" {
		t.Errorf("o Usar recusou o poder escolhido pela tela: %q", refusal)
	}

	// O TIPO É DO BENEFÍCIO: a Capanga dá poder de COMBATE.
	if refusal := powerCommand(t, f, id, "origem/"+capangaPick+"/poder/acrobatico", ""); refusal == "" {
		t.Error("a Capanga aceitou o Acrobático, que é poder de destino")
	}

	// DESMARCAR O BENEFÍCIO leva a escolha junto: sem ele, ela seria uma escolha órfã.
	if refusal := powerCommand(t, f, id, "origem/"+capangaPick, ""); refusal != "" {
		t.Fatalf("desmarcar o benefício foi recusado: %q", refusal)
	}
	if _, picks := storedChoices(t, f.s, id); strings.Contains(picks, capangaPick) {
		t.Errorf("o benefício saiu e a escolha dele ficou: %s", picks)
	}
}
