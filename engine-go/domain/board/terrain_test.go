package board

import (
	"encoding/json"
	"testing"

	"t20engine/domain/engine"
)

// Os guardas das QUATRO espécies de terreno (T20 p238, Tabela 5-3).
//
// O que se prende aqui é a SEPARAÇÃO: cada espécie tem a própria lista, e pintar
// uma não mexe nas outras. É a garantia que um mapa com chave de string não
// daria — lá `"elevated"` contra `"elevated"` viraria lista vazia em silêncio.

// Amostragem e não enumeração: percorre `TerrainKinds`, então a quinta
// espécie que alguém acrescentar já nasce medida, e não há uma entrada por caso
// para alguém esquecer de escrever.
func TestEachKindHasItsOwnList(t *testing.T) {
	square := engine.Square{X: 3, Y: 4}
	for _, brush := range TerrainKinds {
		b := &BoardState{}
		PaintTerrain(b, square, brush.ID, true)

		list := listForKind(b, brush.ID)
		if list == nil || len(*list) != 1 || (*list)[0] != square {
			t.Errorf("%s: pintar não pôs a casa na lista dela (%v)", brush.ID, list)
			continue
		}
		// E NENHUMA outra recebeu nada. Sem esta metade, quatro ponteiros para o
		// mesmo campo passariam verde no laço inteiro.
		for _, other := range TerrainKinds {
			if other.ID == brush.ID {
				continue
			}
			if neighbor := listForKind(b, other.ID); len(*neighbor) != 0 {
				t.Errorf("pintar %s sujou a lista de %s: %v", brush.ID, other.ID, *neighbor)
			}
		}
	}
}

// O pincel é IDEMPOTENTE em cada espécie: o arraste passa duas vezes pela mesma
// casa, e alternar faria ela piscar debaixo do dedo. Quem apaga é a borracha.
func TestTheBrushIsIdempotentForEachKind(t *testing.T) {
	square := engine.Square{X: -2, Y: 7}
	for _, brush := range TerrainKinds {
		b := &BoardState{}
		PaintTerrain(b, square, brush.ID, true)
		version := b.Version
		PaintTerrain(b, square, brush.ID, true)

		if list := listForKind(b, brush.ID); len(*list) != 1 {
			t.Errorf("%s: pintar duas vezes deixou %d casas", brush.ID, len(*list))
		}
		if b.Version != version {
			t.Errorf("%s: repintar a mesma casa subiu a versão de %d para %d — a mesa recebe um remendo sobre nada",
				brush.ID, version, b.Version)
		}

		PaintTerrain(b, square, brush.ID, false)
		if list := listForKind(b, brush.ID); len(*list) != 0 {
			t.Errorf("%s: a borracha não apagou (%v)", brush.ID, *list)
		}
	}
}

// O id vem do CLIENTE. Uma espécie que a tela não oferece só chega por posse do
// fio, e a resposta não pode ser nem pânico nem pintar a lista errada.
func TestAnInventedKindPaintsNothingAndDoesNotCrash(t *testing.T) {
	b := &BoardState{}
	PaintTerrain(b, engine.Square{X: 1, Y: 1}, TerrainKind("lava"), true)

	if b.Version != 0 {
		t.Errorf("uma espécie inventada subiu a versão para %d", b.Version)
	}
	for _, brush := range TerrainKinds {
		if list := listForKind(b, brush.ID); len(*list) != 0 {
			t.Errorf("a espécie inventada foi parar em %s: %v", brush.ID, *list)
		}
	}
	// E o portão que a rota usa devolve o DIFÍCIL, que é o que o pincel sempre
	// pintou — a compatibilidade com quem manda só `x`, `y` e `difficult`.
	if e := KnownTerrainKind("lava"); e != TerrenoDificil {
		t.Errorf("espécie inventada caiu em %q em vez do difícil", e)
	}
	if e := KnownTerrainKind(""); e != TerrenoDificil {
		t.Errorf("espécie AUSENTE caiu em %q — a SPA manda o corpo antigo, sem `kind`", e)
	}
}

// O Lugar guardado é `json.Marshal` do estado inteiro, e reabrir é `Unmarshal`.
// Uma espécie sem tag JSON, ou com tag repetida, sumiria na ida e volta — e o
// sintoma seria a taverna reabrindo parecendo certa, com o pântano virando chão
// liso. Perder dado sem estourar é o defeito que este guarda existe para pegar.
func TestTheStrokesSurviveTheArchive(t *testing.T) {
	square := engine.Square{X: 5, Y: -3}
	original := &BoardState{Place: "Pântano"}
	for _, brush := range TerrainKinds {
		PaintTerrain(original, square, brush.ID, true)
	}

	blob, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("guardar: %v", err)
	}
	var returned BoardState
	if err := json.Unmarshal(blob, &returned); err != nil {
		t.Fatalf("reabrir: %v", err)
	}

	for _, brush := range TerrainKinds {
		list := listForKind(&returned, brush.ID)
		if list == nil || len(*list) != 1 || (*list)[0] != square {
			t.Errorf("%s não sobreviveu ao acervo: %v", brush.ID, list)
		}
	}
}

// A assimetria que a forma de quatro listas existe para deixar à vista: o
// difícil alimenta o MOTOR, os outros três alimentam o olho. Cobertura no
// caminho não pode encarecer o passo — seria inventar uma regra que o livro não
// tem, e ela apareceria como a peça andando menos sem explicação.
func TestOnlyDifficultTerrainCountsForMovement(t *testing.T) {
	square := engine.Square{X: 1, Y: 0}
	b := &BoardState{}
	for _, brush := range TerrainKinds {
		if brush.ID != TerrenoDificil {
			PaintTerrain(b, square, brush.ID, true)
		}
	}
	noHard := moveTerrainOf(b)
	if len(noHard.Difficult) != 0 {
		t.Errorf("cobertura, camuflagem ou elevado entraram na conta do movimento: %v", noHard.Difficult)
	}

	// O CONTROLE: com o difícil pintado a casa APARECE. Sem ele, "nada entrou na
	// conta" seria verdade também sobre uma tradução quebrada que nunca devolve
	// casa nenhuma.
	PaintTerrain(b, square, TerrenoDificil, true)
	if withHard := moveTerrainOf(b); !withHard.Difficult[square] {
		t.Error("o terreno difícil não chegou ao motor — o guarda acima mediria um cano entupido")
	}
}
