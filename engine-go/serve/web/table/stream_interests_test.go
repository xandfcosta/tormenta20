package table

import (
	"testing"

	"t20engine/infra/events"
)

// O FLUXO DA MESA ESCUTA TUDO O QUE ELA MOSTRA (ALE-639): a sessão, a campanha (as regras, os lugares, o
// elenco) e a ficha de cada um do Grupo — o PV do outro jogador muda pela ficha dele, por HTTP, e a Mesa
// de todos o mostra. E o aviso de "a sua ficha mudou" é só da ficha de quem olha.
func TestTheTableStreamListensToWhatTheTableShows(t *testing.T) {
	view := View{SessionID: 5, CampaignID: 3, Eu: &tableMe{CharacterID: 10},
		Group: []Member{{CharacterID: 10}, {CharacterID: 11}}}
	bus := &events.Bus{}
	listening, cancel := bus.Subscribe(readerInterests(view)...)
	defer cancel()
	for _, c := range []struct {
		why string
		ev  events.Event
	}{
		{"a fila da sessão", events.TurnAdvanced{SessionID: 5}},
		{"a regra da campanha", events.TableRuleChanged{CampaignID: 3}},
		{"a ficha de quem olha", events.CharacterChanged{CharacterID: 10}},
		{"a ficha de outro do Grupo", events.CharacterChanged{CharacterID: 11}},
	} {
		bus.Publish(c.ev)
		select {
		case <-listening.C:
		default:
			t.Errorf("o fluxo da Mesa não escuta %s", c.why)
		}
	}
	bus.Publish(events.CharacterChanged{CharacterID: 99})
	select {
	case <-listening.C:
		t.Error("o fluxo da Mesa escuta a ficha de quem não está no Grupo")
	default:
	}

	if !sheetChanged(events.CharacterChanged{CharacterID: 10}, view) {
		t.Error("a ficha de quem olha mudou, e o fluxo não avisa a ficha embutida")
	}
	if sheetChanged(events.CharacterChanged{CharacterID: 11}, view) {
		t.Error("a ficha de outro do Grupo mudou, e o fluxo avisou a ficha embutida de quem olha")
	}
}
