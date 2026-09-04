package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"t20engine/db/sqlcgen"
	"t20engine/plataforma"
	"t20engine/sheet"
	"testing"
)

func effectScreen(t *testing.T, f pilotoFixture, id int64) string {
	t.Helper()
	return f.pede(t, f.jogador, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=conditionals", id), "").Body.String()
}

func effect(t *testing.T, f pilotoFixture, id int64, caminho string) *responseRecorderLike {
	t.Helper()
	alvo := fmt.Sprintf("/personagens/%d/efeitos/%s?tab=conditionals", id, caminho)
	rec := f.pede(t, f.jogador, http.MethodPost, alvo, "")
	return &responseRecorderLike{Code: rec.Code, Body: rec.Body.String()}
}

func conditions(t *testing.T, f pilotoFixture, id int64) []string {
	t.Helper()
	row, err := f.s.Queries().GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler o personagem: %v", err)
	}
	return sheet.UnmarshalStrings(row.Activeconditions)
}

// A CONDIÇÃO ENTRA, SAI, E MOVE OS NÚMEROS.
//
// Mover os números é o caso inteiro: uma condição que fosse só crachá foi o
// defeito da ALE-28. O Caído (p394) NÃO mexe na Defesa geral — ele separa −5
// contra corpo a corpo e +5 contra à distância —, então medir o total mediria
// justamente onde a regra não aparece, e daria "22 → 22" com cara de "não
// funcionou". O guarda olha as linhas DIRECIONAIS.
func TestAConditionEntersLeavesAndMovesTheNumbers(t *testing.T) {
	f, id := fighterFixture(t)

	aba := combatScreen(t, f, id)
	if strings.Contains(aba, "Contra corpo a corpo") {
		t.Fatal("a Defesa já vinha com linhas direcionais: o caso não mede a chegada do Caído")
	}

	if rec := effect(t, f, id, "condicao/caido"); rec.Code != http.StatusOK {
		t.Fatalf("aplicar o Caído respondeu %d: %s", rec.Code, rec.Body)
	}
	if got := conditions(t, f, id); len(got) != 1 || got[0] != "caido" {
		t.Fatalf("as condições gravadas são %v, quer [caido]", got)
	}
	comCaido := combatScreen(t, f, id)
	for _, esperado := range []string{"Contra corpo a corpo", "Contra ataques à distância"} {
		if !strings.Contains(comCaido, esperado) {
			t.Errorf("com o Caído aplicado a Defesa não mostra %q: a condição virou crachá", esperado)
		}
	}

	// O MESMO comando desliga: o gesto manda a condição, não o estado desejado.
	if rec := effect(t, f, id, "condicao/caido"); rec.Code != http.StatusOK {
		t.Fatalf("remover o Caído respondeu %d", rec.Code)
	}
	if got := conditions(t, f, id); len(got) != 0 {
		t.Errorf("o segundo toque não removeu o Caído: %v", got)
	}
}

// UMA CONDIÇÃO INVENTADA É RECUSADA, e nada é gravado.
//
// A autoridade é o CATÁLOGO, e não uma lista escrita na tela: um blob com uma
// condição fantasma injetaria na ficha um efeito que o livro não tem.
func TestAnInventedConditionIsRefused(t *testing.T) {
	f, id := fighterFixture(t)
	if recusa := sceneRefusal(effect(t, f, id, "condicao/entediado").Body); recusa == "" {
		t.Error("uma condição que não existe foi aceita sem uma palavra na tela")
	}
	if got := conditions(t, f, id); len(got) != 0 {
		t.Errorf("a recusa gravou assim mesmo: %v", got)
	}
}

// O EFEITO APLICADO É DE QUEM O APLICOU, e a query não confere isso sozinha.
//
// O `DeleteEffectByID` apaga por id e mais nada. Sem a leitura de posse que o
// comando faz antes, um `@post` montado à mão encerraria o efeito de OUTRO
// personagem — e o 403 do `sheetCommand` não pega, porque a ficha do caminho é
// a do dono.
func TestAnEffectFromAnotherSheetCannotBeEnded(t *testing.T) {
	f, meu := fighterFixture(t)
	outro := seedCharacterAtLevel(t, f.s, f.jogador, "Vizinho", 1, 10, 10, 0, 0)
	alheio, err := f.s.Queries().CreateActiveEffect(context.Background(), sqlcgen.CreateActiveEffectParams{
		Characterid: outro, Catalogid: "armadura-arcana", Scope: "scene",
		Modifiers: "[]", Createdat: plataforma.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear o efeito alheio: %v", err)
	}

	rec := effect(t, f, meu, fmt.Sprintf("encerra/%d", alheio.ID))
	if recusa := sceneRefusal(rec.Body); recusa == "" {
		t.Error("encerrei o efeito de outro personagem pela minha ficha")
	}
	if _, err := f.s.Queries().GetActiveEffectMeta(context.Background(), alheio.ID); err != nil {
		t.Error("o efeito alheio foi apagado assim mesmo")
	}
}

// O PAINEL CHEGA NA TELA.
func TestTheEffectsPanelDrawsTheFourBlocks(t *testing.T) {
	f, id := fighterFixture(t)
	tela := effectScreen(t, f, id)

	for _, esperado := range []string{"Condições (p394)", "Efeitos ativos", "Aplicar condição", "Aplicar magia"} {
		if !strings.Contains(tela, esperado) {
			t.Errorf("a tela não tem %q", esperado)
		}
	}
}
