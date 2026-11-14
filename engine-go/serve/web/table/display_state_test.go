package table

import (
	"fmt"
	"testing"
	"time"

	"t20engine/infra/events"
)

// CADA ATO DA TELA DA MESA PUBLICA O SEU: ela só existe em memória, e o fluxo que a desenha só acorda no
// evento. E o segundo Congelar não publica nada, porque não tira outro retrato — a tela continua a mesma.
func TestEveryDisplayActPublishesItsEvent(t *testing.T) {
	bus := &events.Bus{}
	sub, stop := bus.Subscribe(events.OfSession(7))
	defer stop()
	d := newTableDisplays(bus)

	d.Join(7)
	d.Frame(7, displayFraming{X: 1, Y: 2, Width: 10, Height: 6})
	d.Freeze(7, displayView{}, time.Now())
	d.Freeze(7, displayView{}, time.Now())
	d.Thaw(7)
	d.Leave(7)

	want := []string{"TableDisplayOpened", "TableDisplayFramed", "TableDisplayFrozen", "TableDisplayThawed", "TableDisplayClosed"}
	var got []string
	for len(sub.C) > 0 {
		got = append(got, fmt.Sprintf("%T", <-sub.C)[len("events."):])
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("os atos publicaram %v, e cada um devia publicar o seu, nesta ordem: %v", got, want)
	}
}
