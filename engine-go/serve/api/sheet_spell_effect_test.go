package api

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// O EFEITO DA MAGIA NA FICHA (ver GLOSSARY, **efeito da magia**): uma linha por magia, a duração que a conjuração
// deu, e a manutenção só de quem conjurou. A escrita em si é provada no `app/character`; aqui ficam os gestos.

// spellRowsOf são as linhas em vigor da magia na ficha, pela duração gravada.
func spellRowsOf(t *testing.T, s *Server, characterID int64, spellID string) []string {
	t.Helper()
	rows, err := s.queries.ListActiveEffectsByCharacter(context.Background(), characterID)
	if err != nil {
		t.Fatalf("listar efeitos da ficha %d: %v", characterID, err)
	}
	var scopes []string
	for _, r := range rows {
		if r.Catalogid == spellID {
			scopes = append(scopes, r.Scope)
		}
	}
	return scopes
}

// O APRIMORAMENTO QUE MUDA A DURAÇÃO chega à linha do Aplicar: a Velocidade conjurada de cena e aplicada de cena
// fica de cena, sem manutenção; a Camuflagem Ilusória aplicada antes e conjurada sustentada fica UMA linha, e
// sustentada.
func TestTheAugmentedDurationReachesTheAppliedSpell(t *testing.T) {
	s := newCastServer(t)
	owner := seedUser(t, s, "duracao@t20.local")
	arcanist := seedCaster(t, s, owner, "Arcanista", 9, 40, "velocidade")
	learnPrepared(t, s, arcanist, "camuflagem-ilusoria")
	plays := s.characterPlays()
	ctx := context.Background()

	// O aprimoramento 0 da Velocidade muda a duração para cena.
	if err := castSpell(t, s, owner, arcanist, "velocidade", `{"augments":[{"augmentIndex":0,"stacks":1}]}`); err != nil {
		t.Fatalf("conjurar a Velocidade de cena: %v", err)
	}
	if err := plays.ApplySpellBuff(ctx, arcanist, "velocidade", "cena"); err != nil {
		t.Fatalf("aplicar a Velocidade de cena: %v", err)
	}
	if got := spellRowsOf(t, s, arcanist, "velocidade"); len(got) != 1 || got[0] != "scene" {
		t.Errorf("a Velocidade de cena ficou nas linhas %v; want uma, de cena", got)
	}
	if upkeep, err := (sheetVitals{q: s.queries}).SustainedOf(ctx, arcanist); err != nil || len(upkeep) != 0 {
		t.Errorf("a Velocidade de cena cobra manutenção: %v (%v)", upkeep, err)
	}

	if err := plays.ApplySpellBuff(ctx, arcanist, "camuflagem-ilusoria", ""); err != nil {
		t.Fatalf("aplicar a Camuflagem Ilusória: %v", err)
	}
	// O aprimoramento 0 da Camuflagem Ilusória muda a duração para sustentada.
	if err := castSpell(t, s, owner, arcanist, "camuflagem-ilusoria", `{"augments":[{"augmentIndex":0,"stacks":1}]}`); err != nil {
		t.Fatalf("conjurar a Camuflagem Ilusória sustentada: %v", err)
	}
	if got := spellRowsOf(t, s, arcanist, "camuflagem-ilusoria"); len(got) != 1 || got[0] != "sustained" {
		t.Errorf("a Camuflagem Ilusória ficou nas linhas %v; want uma, sustentada", got)
	}

	t.Run("a duração que a magia não tem é recusada", func(t *testing.T) {
		err := plays.ApplySpellBuff(ctx, arcanist, "velocidade", "dia")
		if err == nil || !strings.Contains(err.Error(), "Velocidade") {
			t.Errorf("aplicar a Velocidade de um dia devolveu %v; want a recusa nomeando a magia", err)
		}
	})
}

// QUEM RECEBE NÃO SUSTENTA: o aliado que aplica a Velocidade na própria ficha não paga a manutenção, que é de quem
// conjurou, nem fica impedido de receber a Oração de outro conjurador (decisão do dono).
func TestTheAllyWhoReceivesASustainedSpellDoesNotSustainIt(t *testing.T) {
	f, ally := fighterFixture(t)
	for _, path := range []string{"aplica/velocidade", "aplica/oracao"} {
		if rec := effect(t, f, ally, path); rec.Code != http.StatusOK || sceneRefusal(rec.Body) != "" {
			t.Fatalf("o aliado aplicar %s respondeu %d: %s", path, rec.Code, sceneRefusal(rec.Body))
		}
	}
	upkeep, err := (sheetVitals{q: f.s.queries}).SustainedOf(context.Background(), ally)
	if err != nil || len(upkeep) != 0 {
		t.Errorf("o aliado paga a manutenção de %v (%v); quem paga é quem conjurou", upkeep, err)
	}
}

// O APLICAR OFERECE AS DURAÇÕES DA MAGIA: a do verbete e a que o aprimoramento dá. A esfera do Campo de Força não
// é duração do bônus, porque ela não dá os PV temporários (p183).
func TestTheApplyOffersTheDurationsTheAugmentGives(t *testing.T) {
	f, id := fighterFixture(t)
	// O diálogo do Aplicar mora fora da seção da aba, então a página inteira é o recorte.
	page := effectScreen(t, f, id)
	for _, want := range []string{
		`aria-label="Aplicar magia: Velocidade"`,
		`aria-label="Aplicar magia: Velocidade · cena"`,
		`/efeitos/aplica/velocidade?tab=conditionals&amp;duracao=cena`,
		`aria-label="Aplicar magia: Camuflagem Ilusória · sustentada"`,
		`aria-label="Aplicar magia: Oração · cena"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("o Aplicar magia não oferece %s", want)
		}
	}
	if strings.Contains(page, `Aplicar magia: Campo de Força · sustentada`) {
		t.Error("o Aplicar oferece o Campo de Força sustentado, e a esfera não dá os PV temporários")
	}
}
