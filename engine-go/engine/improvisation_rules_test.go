package engine

import "testing"

// As regras do improviso contra o livro (ALE-261).

// TestTheDieCoversEveryFaceAndNoneBeyondThem.
//
// Aleatoriedade se testa pela FAIXA e pela cobertura, não por um valor
// esperado: um gerador que devolvesse sempre 1 passaria num teste de "está
// entre 1 e 6", e um que devolvesse 0 ou 7 quebraria a tabela em silêncio.
func TestTheDieCoversEveryFaceAndNoneBeyondThem(t *testing.T) {
	for _, faces := range []int{6, 20} {
		vistas := map[int]int{}
		// 40× as faces dá margem folgada: a chance de uma face não sair em 240
		// rolagens de d6 é de ordem 10^-19.
		for i := 0; i < faces*40; i++ {
			d, err := RollDie(faces)
			if err != nil {
				t.Fatalf("d%d: %v", faces, err)
			}
			if d.Valor < 1 || d.Valor > faces {
				t.Fatalf("d%d saiu %d — fora da faixa", faces, d.Valor)
			}
			if d.Faces != faces {
				t.Errorf("d%d disse ter %d faces", faces, d.Faces)
			}
			vistas[d.Valor]++
		}
		for f := 1; f <= faces; f++ {
			if vistas[f] == 0 {
				t.Errorf("d%d: a face %d nunca saiu em %d rolagens", faces, f, faces*40)
			}
		}
	}
}

// TestADieWithoutFacesIsRefused: um `RolaDado(0)` viria de um catálogo torto, e
// `rand.Int` com máximo zero entra em pânico. Recusar é dizer o que houve.
func TestADieWithoutFacesIsRefused(t *testing.T) {
	for _, faces := range []int{0, 1, -3} {
		if _, err := RollDie(faces); err == nil {
			t.Errorf("d%d foi aceito", faces)
		}
	}
}

// TestOneThreatEveryThreeRooms, arredondando PARA CIMA (p263).
//
// O arredondamento é a regra e não um detalhe: sete salas com uma ameaça a cada
// três dão TRÊS, não duas. Duas deixariam a última salinha sem nada, e a regra
// do livro é cota mínima de tensão, não divisão exata.
func TestOneThreatEveryThreeRooms(t *testing.T) {
	casos := map[int]int{1: 1, 3: 1, 4: 2, 6: 2, 7: 3, 9: 3, 10: 4, 50: 17}
	for salas, quero := range casos {
		got, err := PlannedThreats(salas, 3)
		if err != nil {
			t.Fatalf("%d salas: %v", salas, err)
		}
		if got != quero {
			t.Errorf("%d salas deram %d ameaças, quero %d", salas, got, quero)
		}
	}
}

func TestADungeonWithoutRoomsIsRefused(t *testing.T) {
	if _, err := PlannedThreats(0, 3); err == nil {
		t.Error("masmorra de zero salas foi aceita")
	}
	// Razão zero viria de catálogo torto e daria divisão por zero.
	if _, err := PlannedThreats(6, 0); err == nil {
		t.Error("zero salas por ameaça foi aceito")
	}
}

// TestAnUncoveredRollIsAnError: tabela com buraco devolveria a linha errada ou
// nenhuma, e o mestre leria o resultado de outra faixa como se fosse o dele.
func TestAnUncoveredRollIsAnError(t *testing.T) {
	linhas := []faixaDeTeste{{1, 2}, {5, 6}}
	if _, err := RowForRoll(linhas, 1, "teste"); err != nil {
		t.Errorf("a face 1 está coberta e deu erro: %v", err)
	}
	_, err := RowForRoll(linhas, 3, "teste")
	if err == nil {
		t.Fatal("a face 3 não está coberta e passou")
	}
	// A mensagem tem de dizer a tabela E a rolagem: "nenhuma linha" sozinho
	// manda alguém procurar em quatro tabelas.
	if msg := err.Error(); msg == "" || !contem(msg, "teste") || !contem(msg, "3") {
		t.Errorf("a mensagem não localiza o buraco: %q", msg)
	}
}

type faixaDeTeste struct{ min, max int }

func (f faixaDeTeste) Covers(r int) bool { return r >= f.min && r <= f.max }

func contem(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
