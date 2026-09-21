package live

import (
	"strings"
	"testing"
)

// A DECISÃO do ciclo, sem banco, sem servidor e sem fixture.
//
// É a faixa que o `CLAUDE.md` reserva ao que carrega REGRA: os quatro caminhos,
// as duas recusas e o clique repetido cabem numa tabela, e cada linha nomeia o
// caso que ela protege — a mensagem de falha diz o caso, e não "esperava X".
func TestAskingForAStatusSaysWhichWriteToMake(t *testing.T) {
	for _, tc := range []struct {
		tc        string
		current   string
		requested string
		want      StatusChange
		refusal   bool
	}{
		{"planejada e o mestre inicia: carimba o começo", "planned", "active", StatusStartsFresh, false},
		{"ENCERRADA e o mestre inicia: REABRE, com a fila e o tabuleiro", "ended", "active", StatusReopens, false},
		{"já ativa e o mestre inicia de novo: nada, e não é erro", "active", "active", StatusUnchanged, false},
		{"ativa e o mestre encerra: tira do ar", "active", "ended", StatusEnds, false},
		{"já encerrada e o mestre encerra de novo: nada, e não é erro", "ended", "ended", StatusUnchanged, false},
		{"PLANEJADA e o mestre encerra: recusa, senão o histórico mente", "planned", "ended", "", true},
		{"status que ninguém pode pedir", "active", "planned", "", true},
		{"status que não existe", "active", "pausada", "", true},
	} {
		t.Run(tc.tc, func(t *testing.T) {
			change, err := ChangeToStatus(tc.current, tc.requested)
			if tc.refusal {
				if err == nil {
					t.Fatalf("de %q para %q passou, e este caso existe para ser RECUSADO", tc.current, tc.requested)
				}
				return
			}
			if err != nil {
				t.Fatalf("de %q para %q recusou (%v), e este caso tem de passar", tc.current, tc.requested, err)
			}
			if change != tc.want {
				t.Errorf("de %q para %q deu %q, e o caso pede %q", tc.current, tc.requested, change, tc.want)
			}
		})
	}
}

// A MENSAGEM da recusa carrega o valor ofensor, que é o que o `CLAUDE.md` pede
// de toda exceção: "a sessão está X" resolve na hora, "pedido inválido" não.
func TestRefusingToEndAPlannedSessionNamesTheStatus(t *testing.T) {
	_, err := ChangeToStatus("planned", "ended")
	if err == nil {
		t.Fatal("encerrar uma sessão planejada passou")
	}
	if !strings.Contains(err.Error(), "planned") {
		t.Errorf("a mensagem %q não diz em que status a sessão está", err)
	}
}
