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
		fmt.Sprintf("/personagens/%d/atributos/strength/2", id),
		`aria-label="Valor de Força"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("a ficha não traz %q", want)
		}
	}

	// Mudar o livre não mexe no que veio de nível.
	if code := f.requests(t, f.player, http.MethodPost, fmt.Sprintf("/personagens/%d/atributos/strength/2", id), "").Code; code != http.StatusOK {
		t.Fatalf("mudar o valor livre: status %d", code)
	}
	page = html.UnescapeString(f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d", id), "").Body.String())
	if !strings.Contains(page, "+2 livre +2 Aumento de Atributo") {
		t.Error("mudar o valor livre não apareceu na soma, ou mexeu no que veio de nível")
	}
}

// O valor é escrito no próprio diálogo, e o que não é número volta como recusa dentro dele.
func TestTheSheetDialogRefusesAWrongValue(t *testing.T) {
	f, id := barbaro(t, 5)
	for _, written := range []string{"150", "abc"} {
		refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, fmt.Sprintf("/personagens/%d/atributos/strength/%s", id, written), "").Body.String())
		if !strings.Contains(refusal, written) {
			t.Errorf("a recusa de %q não cita o valor: %q", written, refusal)
		}
	}
	if refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, fmt.Sprintf("/personagens/%d/atributos/sorte/1", id), "").Body.String()); !strings.Contains(refusal, "sorte") {
		t.Errorf("a recusa do atributo que não existe não o cita: %q", refusal)
	}
	warned := html.UnescapeString(f.requests(t, f.player, http.MethodPost, fmt.Sprintf("/personagens/%d/atributos/strength/-6", id), "").Body.String())
	if !strings.Contains(warned, "paralisado (p17)") {
		t.Errorf("Força −6 não avisou o efeito do livro:\n%s", warned)
	}
}

// Só quem alcança a ficha muda: quem é de fora da mesa leva a recusa.
func TestOnlyTheOwnerChangesTheFreeValueAfterCreation(t *testing.T) {
	f, id := barbaro(t, 5)
	path := fmt.Sprintf("/personagens/%d/atributos/strength/1", id)
	if code := f.requests(t, seedUser(t, f.s, "estranho@t.com"), http.MethodPost, path, "").Code; code != http.StatusForbidden {
		t.Errorf("quem não é dono mudou o valor livre: status %d", code)
	}
}
