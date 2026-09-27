package engine

import (
	"path/filepath"
	"testing"
)

// Concessões permanentes de PV/PM máximos — o que poderes e habilidades somam
// aos poços da p35, e a regra da p226 que impede a soma dobrada:
//
//	"ATRIBUTOS. O valor de um mesmo atributo não se acumula em características
//	 do personagem. Ou seja, um clérigo/druida não soma duas vezes sua Sabedoria
//	 nos pontos de mana, assim como um bucaneiro/nobre não soma duas vezes seu
//	 Carisma na Defesa."
//
// As concessões vivem no catálogo, e a regra está em COMO elas são somadas.
//
// A metade da DEFESA da mesma regra (bucaneiro/nobre e o Carisma) NÃO está
// implementada — o `resolveStack` agrupa por `bonusType`, não por atributo.

func vitalCatalogs(t *testing.T) *Catalogs {
	t.Helper()
	return primeFromDump(t, filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity")))
}

// O exemplo trabalhado do próprio livro. O catálogo dá a `magias-1-circulo` ao
// Clérigo E ao Druida, as duas com `maxPm` escalando por Sabedoria — sem o
// dedupe, um multiclasse soma a Sabedoria duas vezes no PM.
func TestSameAttributeGrantedTwiceCountsOnce(t *testing.T) {
	c := vitalCatalogs(t)
	attrs := map[string]int{"wisdom": 4, "constitution": 0}

	clericDruid := VitalContext{
		Level: 8,
		Classes: []ClassEntry{
			{ClassName: "Clérigo", Level: 4},
			{ClassName: "Druida", Level: 4},
		},
		AttrTotals: attrs,
	}
	clericOnly := VitalContext{
		Level:      8,
		Classes:    []ClassEntry{{ClassName: "Clérigo", Level: 8}},
		AttrTotals: attrs,
	}

	_, doublePM := BookRuleset(c).sumVitalGrants(clericDruid)
	_, simplePM := BookRuleset(c).sumVitalGrants(clericOnly)

	if doublePM != simplePM {
		t.Errorf("Clérigo/Druida concedeu %d PM e o Clérigo puro %d — a Sabedoria entrou duas vezes (p226)", doublePM, simplePM)
	}
	// E entrou UMA vez, não zero: a regra é não dobrar, não anular.
	if simplePM != attrs["wisdom"] {
		t.Errorf("PM concedido = %d, want %d (a Sabedoria uma vez)", simplePM, attrs["wisdom"])
	}
}

// O dedupe é POR ATRIBUTO e por alvo, não um "só uma concessão de atributo":
// dois atributos diferentes no mesmo alvo continuam somando, e é isso que
// separa a regra do livro de um dedupe grosseiro. Bardo soma Carisma no PM,
// Caçador soma Sabedoria — as duas valem.
func TestDifferentAttributesStillStack(t *testing.T) {
	c := vitalCatalogs(t)
	attrs := map[string]int{"charisma": 3, "wisdom": 2}

	_, pm := BookRuleset(c).sumVitalGrants(VitalContext{
		Level: 4,
		Classes: []ClassEntry{
			{ClassName: "Bardo", Level: 2},
			{ClassName: "Caçador", Level: 2},
		},
		PowerIDs:   []string{"class.cacador.elo-com-a-natureza"},
		AttrTotals: attrs,
	})
	if pm != 5 {
		t.Errorf("PM concedido = %d, want 5 (Carisma 3 + Sabedoria 2 — atributos DIFERENTES acumulam)", pm)
	}
}

// As escalas com que uma concessão cresce. `levelStep` com arredondamento é a
// que mais erra: a Bênção do Mana de Wynna é "+1 PM a cada nível ímpar", modelada
// como passo 2 arredondando para CIMA — no 1º nível ela já vale 1, e não 0.
func TestModifierScaleEvaluation(t *testing.T) {
	attrs := map[string]int{"intelligence": 4}

	t.Run("flat: sem escala, o valor é o valor", func(t *testing.T) {
		if got := evalModifierScale(3, nil, 10, attrs); got != 3 {
			t.Errorf("= %d, want 3", got)
		}
	})

	t.Run("per level: multiplica pelo nível", func(t *testing.T) {
		if got := evalModifierScale(1, &VitalScale{Per: "level"}, 7, attrs); got != 7 {
			t.Errorf("= %d, want 7", got)
		}
	})

	t.Run("per attribute: multiplica pelo atributo FINAL", func(t *testing.T) {
		got := evalModifierScale(1, &VitalScale{Per: "attribute", Attribute: "intelligence"}, 10, attrs)
		if got != 4 {
			t.Errorf("= %d, want 4", got)
		}
	})

	// "a cada nível ímpar" = passo 2 para cima. As fronteiras são os ímpares.
	t.Run("levelStep para CIMA conta o nível em que se entra", func(t *testing.T) {
		scale := &VitalScale{Per: "levelStep", Step: 2, Round: "up"}
		for _, tt := range []struct{ level, want int }{
			{1, 1}, {2, 1}, {3, 2}, {4, 2}, {5, 3}, {20, 10},
		} {
			if got := evalModifierScale(1, scale, tt.level, attrs); got != tt.want {
				t.Errorf("nível %d: = %d, want %d", tt.level, got, tt.want)
			}
		}
	})

	// O PATAMAR, e o exemplo trabalhado é o Coração Heroico (p91): "Você recebe
	// +3 pontos de mana. Quando atinge um novo patamar (no 5º, 11º e 17º
	// níveis), recebe +3 PM."
	//
	// Os oito níveis abaixo são as duas pontas de cada patamar, e os números
	// estão escritos à mão a partir da frase do livro — 3 de saída, e mais 3 em
	// cada limiar. Derivá-los de `patamarOf` faria o caso e o defeito andarem
	// juntos.
	//
	// É a razão de a escala existir: nenhum `levelStep` produz esta coluna. Com
	// passo 6 para baixo o nível 5 dá 3 (devia dar 6); para cima, o nível 1 dá 6
	// (devia dar 3).
	t.Run("per patamar: o Coração Heroico dá 3, 6, 9 e 12 PM", func(t *testing.T) {
		scale := &VitalScale{Per: "patamar"}
		for _, tt := range []struct{ level, want int }{
			{1, 3}, {4, 3}, // iniciante
			{5, 6}, {10, 6}, // veterano
			{11, 9}, {16, 9}, // campeão
			{17, 12}, {20, 12}, // lenda
		} {
			if got := evalModifierScale(3, scale, tt.level, attrs); got != tt.want {
				t.Errorf("nível %d: = %d PM, want %d", tt.level, got, tt.want)
			}
		}
	})

	// Para BAIXO é a outra metade: no nível 1 ainda não vale nada.
	t.Run("levelStep para baixo só conta o passo completo", func(t *testing.T) {
		scale := &VitalScale{Per: "levelStep", Step: 2}
		for _, tt := range []struct{ level, want int }{
			{1, 0}, {2, 1}, {3, 1}, {4, 2}, {20, 10},
		} {
			if got := evalModifierScale(1, scale, tt.level, attrs); got != tt.want {
				t.Errorf("nível %d: = %d, want %d", tt.level, got, tt.want)
			}
		}
	})
}

// Os poços nunca ficam negativos, por mais que as penalidades somem — um
// personagem com PV máximo negativo não é um estado que a ficha saiba desenhar.
func TestVitalPoolsNeverGoNegative(t *testing.T) {
	c := vitalCatalogs(t)
	pools := BookRuleset(c).ComputeVitals(VitalContext{
		Level:      1,
		Classes:    []ClassEntry{{ClassName: "Arcanista", Level: 1}},
		AttrTotals: map[string]int{"constitution": -20},
	})
	if pools.PvMax < 0 || pools.PmMax < 0 {
		t.Errorf("poços = PV %d / PM %d, want nenhum negativo", pools.PvMax, pools.PmMax)
	}
}

// O PATAMAR CHEGA À FICHA, e não só à função que o calcula.
//
// O `TestModifierScaleEvaluation` prende a ARITMÉTICA da escala; este prende a
// COMPOSIÇÃO — que o modificador do poder único da origem, escolhido como
// benefício, atravessa o `vitalGrantMods` e soma no poço de PM pelo mesmo
// caminho que a ficha usa.
//
// As 18 fichas do oráculo não provam isto: nenhuma delas escolheu o Coração
// Heroico, então o diff delas sai vazio por mais certo ou errado que o número
// esteja. Um oráculo que não exercita o caminho não testemunha nada sobre ele.
//
// O CONTROLE é a mesma ficha SEM a escolha do benefício.
func TestTheHeroicHeartGrantsManaByPatamar(t *testing.T) {
	world := BookRuleset(vitalCatalogs(t))
	const benefício = "origin-heroi-campones-unique"

	// nível → o PM que o livro promete (p91): +3 de saída, e +3 a cada patamar
	// novo. Escrito à mão a partir da frase, não derivado da escala.
	for _, tt := range []struct{ level, want int }{
		{1, 3}, {4, 3}, {8, 6}, {12, 9}, {20, 12},
	} {
		semEscolha := VitalContext{Level: tt.level, Origin: "heroi-campones"}
		comEscolha := semEscolha
		comEscolha.OriginChoices = []string{benefício}

		if _, pm := world.sumVitalGrants(semEscolha); pm != 0 {
			t.Fatalf("nível %d: o controle já estava errado — sem escolher o "+
				"benefício a origem soma %d PM, e o caso abaixo mediria um número "+
				"que não é do poder", tt.level, pm)
		}
		if _, pm := world.sumVitalGrants(comEscolha); pm != tt.want {
			t.Errorf("nível %d: o Coração Heroico somou %d PM e o livro dá %d (p91)",
				tt.level, pm, tt.want)
		}
	}
}
