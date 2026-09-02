package engine

import (
	"path/filepath"
	"testing"
)

// Limites de Carga — livro p141 (PDF 147).
//
//	"Você pode carregar 10 espaços +2 por ponto de Força (ou –1 por ponto de
//	 Força negativo). Se ultrapassar esse limite, fica sobrecarregado — sofre
//	 penalidade de armadura –5 e seu deslocamento é reduzido em –3m. Você não
//	 pode carregar mais do que o dobro do seu limite. Assim, um personagem com
//	 Força 2 pode carregar até 14 espaços sem penalidade, e até 28 espaços
//	 ficando sobrecarregado. Ele não pode carregar mais de 28 espaços de itens."
//
// O exemplo trabalhado do próprio livro é o caso deste teste: ele separa
// "ultrapassar" de "chegar em", que é a leitura que a frase deixa em aberto.
func TestSafeLoadTheBookExampleWithStrength2(t *testing.T) {
	limite := 14 // 10 + 2×2, o número que o livro escreve para Força 2.
	casos := []struct {
		espacos     float64
		sobrecarga  bool
		acimaDoTeto bool
		nota        string
	}{
		{13, false, false, "abaixo do limite"},
		{14, false, false, "EM 14 ainda não é sobrecarga — o livro diz 'até 14 sem penalidade'"},
		{14.5, true, false, "meio espaço acima já ultrapassa"},
		{28, true, false, "o dobro ainda é carregável, sobrecarregado"},
		{28.5, true, true, "acima do dobro o livro diz que ele NÃO pode carregar"},
	}
	for _, caso := range casos {
		ch := Character{Items: []CharacterItem{{Quantity: 1, Slots: caso.espacos}}}
		got := loadBreakdownOf(ch, limite)
		if got.Overloaded != caso.sobrecarga {
			t.Errorf("%v espaços: sobrecarregado=%v, want %v (%s)", caso.espacos, got.Overloaded, caso.sobrecarga, caso.nota)
		}
		if got.OverMax != caso.acimaDoTeto {
			t.Errorf("%v espaços: acima do teto=%v, want %v (%s)", caso.espacos, got.OverMax, caso.acimaDoTeto, caso.nota)
		}
	}
	if got := loadBreakdownOf(Character{}, limite).Max; got != 28 {
		t.Errorf("teto com Força 2 = %d, want 28 (o dobro, p141)", got)
	}
}

// p141: "Por padrão, um item ocupa 1 espaço […] Itens alquímicos, poções,
// pergaminhos e outros itens muitos leves ou pequenos ocupam meio espaço. Ou
// seja, dois desses itens ocupam 1 espaço."
//
// A linha de inventário guarda QUANTIDADE e ESPAÇO separados, então a soma é
// quantidade × espaço — e o caso de meio espaço é o único em que somar as
// linhas em vez de multiplicar passaria despercebido.
func TestLoadMultipliesQuantityByTheItemSlots(t *testing.T) {
	ch := Character{Items: []CharacterItem{
		{Name: "Poção de cura", Quantity: 2, Slots: 0.5},
		{Name: "Montante", Quantity: 1, Slots: 2},
	}}
	if got := loadBreakdownOf(ch, 14).Items; got != 3 {
		t.Errorf("duas poções (meio espaço cada) + um montante (2) = %v espaços, want 3", got)
	}
}

// p141: "Cada mil moedas, independentemente do tipo, ocupam 1 espaço."
//
// O milheiro é COMPLETO: 999 moedas não fecham o primeiro e não ocupam nada.
// Sem esta leitura o dinheiro ou não pesa nunca, ou pesa um espaço inteiro por
// um punhado de tibares.
func TestMoneyLoadCountsWholeThousands(t *testing.T) {
	casos := map[float64]float64{
		0:    0,
		999:  0,
		1000: 1,
		1999: 1,
		3000: 3,
	}
	for tibar, want := range casos {
		got := loadBreakdownOf(Character{Tibar: tibar}, 14)
		if got.Coins != want {
			t.Errorf("T$ %v ocupam %v espaços, want %v", tibar, got.Coins, want)
		}
		if got.Used != want {
			t.Errorf("T$ %v: carga total %v, want %v — o dinheiro entra na MESMA soma dos itens", tibar, got.Used, want)
		}
	}
}

// A consequência da sobrecarga, ponta a ponta pela ficha inteira: o motor não
// pode só ANUNCIAR "–5 e –3m" e deixar o resto da ficha com os números de quem
// anda leve. Roda por `ComputeSheetV2` de propósito — o defeito que este teste
// mira não é a conta da carga, é ela não chegar ao deslocamento e às perícias.
//
// p141: "sofre penalidade de armadura –5 e seu deslocamento é reduzido em –3m".
// p153: a penalidade de armadura vale em "testes de Acrobacia, Furtividade e
// Ladinagem".
func TestOverloadPenalizesDisplacementAndArmorExpertises(t *testing.T) {
	catalogs := primeFromDump(t, filepath.Clean(filepath.Join(mustWd(t), "..", "parity")))
	pericias := []CharacterExpertise{
		{Name: "Furtividade", Attribute: "dexterity"},
		{Name: "Diplomacia", Attribute: "charisma"},
	}
	// Força 0 ⇒ limite 10. Uma linha de 11 espaços ultrapassa; a de 10 não.
	comCarga := func(espacos float64) ComputedSheetV2 {
		ch := Character{
			Level: 1, Displacement: 9, Expertises: pericias,
			Items: []CharacterItem{{Name: "Barril", Quantity: 1, Slots: espacos}},
		}
		return catalogs.ComputeSheetV2(ch, map[string]bool{})
	}

	leve, pesado := comCarga(10), comCarga(11)
	if leve.Displacement.Total != 9 {
		t.Fatalf("dentro do limite o deslocamento é %d, want 9 — o caso de controle já estava errado", leve.Displacement.Total)
	}
	if pesado.Displacement.Total != 6 {
		t.Errorf("sobrecarregado: deslocamento %d, want 6 (9 − 3, p141)", pesado.Displacement.Total)
	}
	if got := periciaTotal(t, pesado, "Furtividade") - periciaTotal(t, leve, "Furtividade"); got != -5 {
		t.Errorf("sobrecarregado: Furtividade mudou %d, want −5 (p141 + p153)", got)
	}
	if got := periciaTotal(t, pesado, "Diplomacia") - periciaTotal(t, leve, "Diplomacia"); got != 0 {
		t.Errorf("sobrecarregado: Diplomacia mudou %d, want 0 — a penalidade de armadura não a alcança (p153)", got)
	}
}

func periciaTotal(t *testing.T, sheet ComputedSheetV2, name string) int {
	t.Helper()
	for _, ex := range sheet.Expertises {
		if ex.Name == name {
			return ex.Total
		}
	}
	t.Fatalf("perícia %q não saiu na ficha", name)
	return 0
}
