package ds

import "testing"

// A BARRA DO MEDIDOR NÃO TRANSBORDA (ALE-479).
//
// Com PV temporário, a reserva é parte da barra: medir a vida sobre o máximo
// sozinho daria 96% de vida mais 11% de reserva, e a barra passaria da caixa.
// Os números são escritos à mão, e não derivados do `share`.
func TestTheMeterBarNeverOverflows(t *testing.T) {
	cases := []struct {
		name         string
		m            Meter
		value, extra string
	}{
		{"vida quase cheia com reserva", Meter{Value: 50, Max: 52, Temp: 6}, "86%", "10%"},
		{"sem reserva", Meter{Value: 11, Max: 24}, "45%", "0%"},
		{"cura acima do máximo", Meter{Value: 70, Max: 52}, "100%", "0%"},
		{"caído", Meter{Value: -4, Max: 52}, "0%", "0%"},
		{"sem máximo", Meter{Value: 9}, "0%", "0%"},
	}
	for _, c := range cases {
		if got := c.m.share(c.m.Value); got != c.value {
			t.Errorf("%s: a vida ocupa %s, quero %s", c.name, got, c.value)
		}
		if got := c.m.share(c.m.Temp); got != c.extra {
			t.Errorf("%s: a reserva ocupa %s, quero %s", c.name, got, c.extra)
		}
	}
}
