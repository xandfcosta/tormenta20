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
