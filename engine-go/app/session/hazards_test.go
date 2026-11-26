package session

import (
	"context"
	"strings"
	"testing"

	"t20engine/domain/live"
)

// O PERIGO NA MESA (ver GLOSSARY, **perigo**): o mestre dispara a armadilha ou expõe ao ambiente, e a mesa rola o
// que a ficha do perigo manda.

// threes é o dado que sempre tira 3: a conta de cada caso sai à mão.
func threes(int) (int, error) { return 3, nil }

func firedOn(t *testing.T, hazardID, exposureID string) *live.SessionRuntimeState {
	t.Helper()
	store := storeWithDouble(&snapshotsDouble{stored: live.EmptyRuntimeState()})
	state, err := store.FireHazard(context.Background(), 7, hazardID, exposureID, threes)
	if err != nil {
		t.Fatalf("disparar %s %s: %v", hazardID, exposureID, err)
	}
	return state
}

func TestTheFiredTrapRollsItsDamageAndTellsTheSave(t *testing.T) {
	pit := firedOn(t, "fosso-com-estacas", "").LastHazard
	if pit == nil || pit.Hazard != "Fosso com Estacas" || pit.Kind != "armadilha" || pit.BookPage != 317 {
		t.Fatalf("o disparo do fosso ficou %+v", pit)
	}
	// 6d6 de impacto e 2d4+5 de perfuração (p317), com o dado que tira 3.
	if len(pit.Parts) != 2 || pit.Parts[0].Total != 18 || pit.Parts[0].Kind != "impacto" || pit.Parts[1].Total != 11 {
		t.Errorf("o dano do fosso saiu %+v; want 18 de impacto e 11 de perfuração", pit.Parts)
	}
	if pit.Save != "Reflexos evita" || pit.DC != 20 {
		t.Errorf("a resistência do fosso saiu %q CD %d; want Reflexos evita CD 20", pit.Save, pit.DC)
	}

	// O DANO FIXO da Agulha ("1 ponto de dano de perfuração") não é dado, e entra como total.
	needle := firedOn(t, "agulha-envenenada", "").LastHazard
	if len(needle.Parts) != 2 || needle.Parts[0].Total != 1 || len(needle.Parts[0].Rolls) != 0 || needle.Parts[1].Total != 3 {
		t.Errorf("o dano da agulha saiu %+v; want 1 fixo e o 1d12", needle.Parts)
	}
}

func TestThePoisonGasRollsHowLongItLastsAndLeavesTheMark(t *testing.T) {
	state := firedOn(t, "gas-venenoso", "")
	gas := state.LastHazard
	// 2d4 que tira 3 e 3: seis rodadas — a do disparo, que já rolou, e mais cinco na vez de quem está nele.
	if gas.Rounds != 6 || gas.RoundsDice != "2d4" || len(gas.RoundsRolls) != 2 {
		t.Fatalf("as rodadas do gás saíram %d (%s %v); want 6 do 2d4", gas.Rounds, gas.RoundsDice, gas.RoundsRolls)
	}
	mark := state.Markable
	if mark == nil || mark.Source != "Gás Venenoso" || mark.Who != "" || !mark.Counted || mark.TurnsLeft != 5 {
		t.Fatalf("o gás deixou a marca %+v; want a do Gás Venenoso, sem conjurador, por mais 5 turnos", mark)
	}
	if len(mark.Parts) != 1 || mark.Parts[0].Dice != "1d12" || mark.Parts[0].Kind != "perda de vida por veneno" {
		t.Errorf("a marca do gás rola %+v; want o 1d12 de perda de vida por veneno", mark.Parts)
	}

	// A ARMADILHA QUE NÃO DURA não deixa marca.
	if pit := firedOn(t, "fosso-camuflado", ""); pit.Markable != nil {
		t.Errorf("o fosso deixou a marca %+v", pit.Markable)
	}
}

func TestTheExposureRollsNothingNowAndLeavesTheMarkOfEachRound(t *testing.T) {
	state := firedOn(t, "lava", "imersao")
	lava := state.LastHazard
	if lava.Exposure != "imersão total" || len(lava.Parts) != 0 || !lava.Lingers {
		t.Fatalf("a imersão na lava ficou %+v; want sem dado rolado, e o dano que persiste depois de sair", lava)
	}
	mark := state.Markable
	if mark == nil || mark.Source != "Lava · imersão total" || mark.Counted || len(mark.Parts) != 1 ||
		mark.Parts[0].Dice != "20d6" || mark.Parts[0].Kind != "fogo" {
		t.Fatalf("a imersão deixou a marca %+v; want 20d6 de fogo por rodada, até o mestre tirar", mark)
	}
}

func TestWhatIsNotFiredIsRefusedSayingWhy(t *testing.T) {
	store := storeWithDouble(&snapshotsDouble{stored: live.EmptyRuntimeState()})
	for _, c := range []struct{ hazard, exposure, says string }{
		{"nao-existe", "", "não está no catálogo"},
		{"variola", "", "não se dispara"},
		{"acido", "", "exposição ou imersão total"},
		{"acido", "afogado", "exposição ou imersão total"},
		{"fosso-camuflado", "imersao", "não tem exposição"},
	} {
		_, err := store.FireHazard(context.Background(), 7, c.hazard, c.exposure, threes)
		if err == nil || !strings.Contains(err.Error(), c.says) {
			t.Errorf("disparar %q %q devolveu %v; want a recusa que diz %q", c.hazard, c.exposure, err, c.says)
		}
	}
}
