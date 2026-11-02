package api

import (
	"context"
	"slices"
	"testing"

	"t20engine/domain/engine"
	"t20engine/infra/db/sqlcgen"
)

func switchesOn(t *testing.T, s *Server, id int64) []string {
	t.Helper()
	on, err := s.queries.ListCharacterConditionals(context.Background(), id)
	if err != nil {
		t.Fatalf("ler os interruptores: %v", err)
	}
	return on
}

func turnSwitchOn(t *testing.T, s *Server, id int64, flag string) {
	t.Helper()
	if err := s.queries.AddCharacterConditional(context.Background(), sqlcgen.AddCharacterConditionalParams{
		Characterid: id, Conditionalid: engine.FlagGroupID(flag),
	}); err != nil {
		t.Fatalf("ligar %s: %v", flag, err)
	}
}

// A vez que passa desliga o que valia só nela. O que vale "até o próximo turno" só sai quando a vez volta
// para o dono (p125), e o "enquanto usa" fica.
func TestTheNextTurnEndsTheSwitchesAtTheTimeTheBookSays(t *testing.T) {
	s, sid, id := fallenCombatant(t)
	ctx := context.Background()
	for _, flag := range []string{"ataque-poderoso", "mira-apurada", "combate-defensivo", "estilo-de-duas-maos"} {
		turnSwitchOn(t, s, id, flag)
	}
	// A vez dele é a de agora: passá-la é o fim do turno.
	if _, err := s.sessions.NextTurn(ctx, sid); err != nil {
		t.Fatalf("girar: %v", err)
	}
	if got := switchesOn(t, s, id); !slices.Equal(got, []string{"flag:combate-defensivo", "flag:estilo-de-duas-maos"}) {
		t.Errorf("depois de a vez passar, ficaram %v", got)
	}
	// E ela volta: a defesa acaba, e o estilo continua.
	if _, err := s.sessions.NextTurn(ctx, sid); err != nil {
		t.Fatalf("girar: %v", err)
	}
	if got := switchesOn(t, s, id); !slices.Equal(got, []string{"flag:estilo-de-duas-maos"}) {
		t.Errorf("depois de a vez voltar, ficaram %v", got)
	}
}
