package api

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"strings"
	"testing"
)

// O toque no atributo mostra o valor livre separado do que veio de nível, com as parcelas
// iguais juntas, e leva à tela que muda só o livre.
func TestTheAttributeDialogSeparatesTheFreeValueFromTheLevelOne(t *testing.T) {
	f, id := barbaro(t, 5)
	if _, err := f.s.db.ExecContext(context.Background(),
		"UPDATE characters SET strength = 1 WHERE id = ?", id,
	); err != nil {
		t.Fatalf("gravar o valor livre: %v", err)
	}
	for i := 0; i < 2; i++ {
		if refusal := powerCommand(t, f, id, barbarianIncrease+"strength", ""); refusal != "" {
			t.Fatalf("Força: %q", refusal)
		}
	}

	page := html.UnescapeString(f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d", id), "").Body.String())
	for _, want := range []string{
		"+1 livre +2 Aumento de Atributo",
		"Força +3",
		fmt.Sprintf("/personagens/%d/atributos", id),
		"Mudar o valor livre",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("a ficha não traz %q", want)
		}
	}

	// Mudar o livre não mexe no que veio de nível.
	if code := f.requests(t, f.player, http.MethodPost, fmt.Sprintf("/personagens/%d/atributos/strength/1", id), "").Code; code != http.StatusOK {
		t.Fatalf("mudar o valor livre: status %d", code)
	}
	page = html.UnescapeString(f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d", id), "").Body.String())
	if !strings.Contains(page, "+2 livre +2 Aumento de Atributo") {
		t.Error("mudar o valor livre não apareceu na soma, ou mexeu no que veio de nível")
	}
}

// Só o dono muda, e a tela é a mesma da forja: o outro jogador leva a recusa.
func TestOnlyTheOwnerChangesTheFreeValueAfterCreation(t *testing.T) {
	f, id := barbaro(t, 5)
	path := fmt.Sprintf("/personagens/%d/atributos/strength/1", id)
	if code := f.requests(t, f.gm, http.MethodPost, path, "").Code; code != http.StatusForbidden {
		t.Errorf("quem não é dono mudou o valor livre: status %d", code)
	}
}
