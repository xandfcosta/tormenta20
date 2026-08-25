package tabuleiro

import (
	"encoding/json"
	"testing"

	"t20engine/engine"
)

// Os guardas das QUATRO espécies de terreno (T20 p238, Tabela 5-3).
//
// O que se prende aqui é a SEPARAÇÃO: cada espécie tem a própria lista, e pintar
// uma não mexe nas outras. É a garantia que um mapa com chave de string não
// daria — lá `"elevated"` contra `"elevado"` viraria lista vazia em silêncio.

// TestCadaEspecieTemAPROPRIAlista.
//
// Amostragem e não enumeração: percorre `EspeciesDeTerreno`, então a quinta
// espécie que alguém acrescentar já nasce medida, e não há uma entrada por caso
// para alguém esquecer de escrever.
func TestCadaEspecieTemAPropriaLista(t *testing.T) {
	casa := engine.Square{X: 3, Y: 4}
	for _, pincel := range EspeciesDeTerreno {
		b := &BoardState{}
		PaintTerrain(b, casa, pincel.ID, true)

		lista := listaDaEspecie(b, pincel.ID)
		if lista == nil || len(*lista) != 1 || (*lista)[0] != casa {
			t.Errorf("%s: pintar não pôs a casa na lista dela (%v)", pincel.ID, lista)
			continue
		}
		// E NENHUMA outra recebeu nada. Sem esta metade, quatro ponteiros para o
		// mesmo campo passariam verde no laço inteiro.
		for _, outra := range EspeciesDeTerreno {
			if outra.ID == pincel.ID {
				continue
			}
			if vizinha := listaDaEspecie(b, outra.ID); len(*vizinha) != 0 {
				t.Errorf("pintar %s sujou a lista de %s: %v", pincel.ID, outra.ID, *vizinha)
			}
		}
	}
}

// TestOPincelEIDEMPOTENTEemCadaEspecie: o arraste passa duas vezes pela mesma
// casa, e alternar faria ela piscar debaixo do dedo. Quem apaga é a borracha.
func TestOPincelEIdempotenteEmCadaEspecie(t *testing.T) {
	casa := engine.Square{X: -2, Y: 7}
	for _, pincel := range EspeciesDeTerreno {
		b := &BoardState{}
		PaintTerrain(b, casa, pincel.ID, true)
		versao := b.Version
		PaintTerrain(b, casa, pincel.ID, true)

		if lista := listaDaEspecie(b, pincel.ID); len(*lista) != 1 {
			t.Errorf("%s: pintar duas vezes deixou %d casas", pincel.ID, len(*lista))
		}
		if b.Version != versao {
			t.Errorf("%s: repintar a mesma casa subiu a versão de %d para %d — a mesa recebe um remendo sobre nada",
				pincel.ID, versao, b.Version)
		}

		PaintTerrain(b, casa, pincel.ID, false)
		if lista := listaDaEspecie(b, pincel.ID); len(*lista) != 0 {
			t.Errorf("%s: a borracha não apagou (%v)", pincel.ID, *lista)
		}
	}
}

// TestEspecieINVENTADAnaoPintaNadaEnaoDerruba.
//
// O id vem do CLIENTE. Uma espécie que a tela não oferece só chega por posse do
// fio, e a resposta não pode ser nem pânico nem pintar a lista errada.
func TestEspecieInventadaNaoPintaNadaENaoDerruba(t *testing.T) {
	b := &BoardState{}
	PaintTerrain(b, engine.Square{X: 1, Y: 1}, EspecieDeTerreno("lava"), true)

	if b.Version != 0 {
		t.Errorf("uma espécie inventada subiu a versão para %d", b.Version)
	}
	for _, pincel := range EspeciesDeTerreno {
		if lista := listaDaEspecie(b, pincel.ID); len(*lista) != 0 {
			t.Errorf("a espécie inventada foi parar em %s: %v", pincel.ID, *lista)
		}
	}
	// E o portão que a rota usa devolve o DIFÍCIL, que é o que o pincel sempre
	// pintou — a compatibilidade com quem manda só `x`, `y` e `difficult`.
	if e := EspecieConhecida("lava"); e != TerrenoDificil {
		t.Errorf("espécie inventada caiu em %q em vez do difícil", e)
	}
	if e := EspecieConhecida(""); e != TerrenoDificil {
		t.Errorf("espécie AUSENTE caiu em %q — a SPA manda o corpo antigo, sem `kind`", e)
	}
}

// TestOsTracosSOBREVIVEMaoAcervo.
//
// O Lugar guardado é `json.Marshal` do estado inteiro, e reabrir é `Unmarshal`.
// Uma espécie sem tag JSON, ou com tag repetida, sumiria na ida e volta — e o
// sintoma seria a taverna reabrindo parecendo certa, com o pântano virando chão
// liso. Perder dado sem estourar é o defeito que este guarda existe para pegar.
func TestOsTracosSobrevivemAoAcervo(t *testing.T) {
	casa := engine.Square{X: 5, Y: -3}
	original := &BoardState{Place: "Pântano"}
	for _, pincel := range EspeciesDeTerreno {
		PaintTerrain(original, casa, pincel.ID, true)
	}

	blob, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("guardar: %v", err)
	}
	var voltou BoardState
	if err := json.Unmarshal(blob, &voltou); err != nil {
		t.Fatalf("reabrir: %v", err)
	}

	for _, pincel := range EspeciesDeTerreno {
		lista := listaDaEspecie(&voltou, pincel.ID)
		if lista == nil || len(*lista) != 1 || (*lista)[0] != casa {
			t.Errorf("%s não sobreviveu ao acervo: %v", pincel.ID, lista)
		}
	}
}

// TestSOoDIFICILentraNaContaDoMovimento.
//
// A assimetria que a forma de quatro listas existe para deixar à vista: o
// difícil alimenta o MOTOR, os outros três alimentam o olho. Cobertura no
// caminho não pode encarecer o passo — seria inventar uma regra que o livro não
// tem, e ela apareceria como a peça andando menos sem explicação.
func TestSoODificilEntraNaContaDoMovimento(t *testing.T) {
	casa := engine.Square{X: 1, Y: 0}
	b := &BoardState{}
	for _, pincel := range EspeciesDeTerreno {
		if pincel.ID != TerrenoDificil {
			PaintTerrain(b, casa, pincel.ID, true)
		}
	}
	semDificil := moveTerrainOf(b)
	if len(semDificil.Difficult) != 0 {
		t.Errorf("cobertura, camuflagem ou elevado entraram na conta do movimento: %v", semDificil.Difficult)
	}

	// O CONTROLE: com o difícil pintado a casa APARECE. Sem ele, "nada entrou na
	// conta" seria verdade também sobre uma tradução quebrada que nunca devolve
	// casa nenhuma.
	PaintTerrain(b, casa, TerrenoDificil, true)
	if comDificil := moveTerrainOf(b); !comDificil.Difficult[casa] {
		t.Error("o terreno difícil não chegou ao motor — o guarda acima mediria um cano entupido")
	}
}
