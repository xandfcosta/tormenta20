package engine

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TODO CAMPO DO `Modifier` ATRAVESSA O JSON DO CATÁLOGO (ALE-415).
//
// # O defeito que ele repõe
//
// O `UnmarshalJSON` existe por UMA razão — o `amountInEngineUnits`, que leva o
// `amount` do catálogo para a unidade do motor. Ele era escrito com uma
// struct-sombra de campos à mão, e a sombra ENVELHECEU: a ALE-412 deu `dice` ao
// `Modifier` e não à sombra, e todo dado extra escrito no catálogo era
// descartado em silêncio na leitura.
//
// O que passou por cima disso é instrutivo: o
// `TestTheEnchantReachesTheWeaponCard` monta o `Modifier` em GO e nunca
// atravessa o JSON, então ele provava a composição por cima da fronteira que
// estava quebrada. Seis encantos e um material entraram no catálogo inertes,
// com a suíte verde.
//
// # Por que ele é REFLEXIVO
//
// Uma lista de campos escrita à mão seria a sombra de novo, num segundo lugar.
// Aqui o denominador é o TIPO: cada campo do `Modifier` tem de sair do JSON com
// o valor que entrou, e um campo novo que ninguém preencher reprova pelo NOME —
// que é o único jeito de esta armadilha não se repetir.
func TestEveryModifierFieldSurvivesTheCatalogJSON(t *testing.T) {
	// Todo campo com valor DISTINTO do zero, para que "não atravessou" e
	// "atravessou zerado" não se pareçam.
	cheio := Modifier{
		Target: ModifierTarget{
			K: "damage", Name: "Luta", Attribute: "strength",
			Scope: "this", DamageType: "fogo",
		},
		Amount:    3,
		Dice:      "1d6",
		BonusType: "enhancement",
		Condition: &ModifierCondition{C: "against", Trait: "mortos-vivos"},
		Note:      "uma nota",
		Scale:     &VitalScale{Per: "level", Step: 2},
		Factor:    &Ratio{Num: 1, Den: 2},
	}
	// O CONTROLE: o caso preenche TODOS os campos. Sem ele, um campo novo
	// nasceria zerado dos dois lados e o round-trip passaria sobre o nada.
	tipo := reflect.TypeOf(cheio)
	valor := reflect.ValueOf(cheio)
	for i := 0; i < tipo.NumField(); i++ {
		if valor.Field(i).IsZero() {
			t.Fatalf("o campo %s não foi preenchido neste caso: um campo no zero dos dois "+
				"lados atravessa o JSON sem provar nada.\nPreencha-o aqui junto de "+
				"acrescentá-lo ao `Modifier`.", tipo.Field(i).Name)
		}
	}

	bruto, err := json.Marshal(cheio)
	if err != nil {
		t.Fatalf("escrever o modificador: %v", err)
	}
	var voltou Modifier
	if err := json.Unmarshal(bruto, &voltou); err != nil {
		t.Fatalf("ler o modificador: %v", err)
	}

	for i := 0; i < tipo.NumField(); i++ {
		nome := tipo.Field(i).Name
		// O `Amount` é o único que MUDA de propósito, e só para deslocamento —
		// aqui o alvo é `damage`, então ele atravessa intocado.
		if !reflect.DeepEqual(valor.Field(i).Interface(), reflect.ValueOf(voltou).Field(i).Interface()) {
			t.Errorf("o campo %s não sobreviveu ao JSON: entrou %v e voltou %v.\n"+
				"O `UnmarshalJSON` do `Modifier` monta o valor campo a campo — "+
				"campo que ele não copiar é descartado em SILÊNCIO, e o catálogo "+
				"passa a carregar regra que o motor nunca lê.",
				nome, valor.Field(i).Interface(), reflect.ValueOf(voltou).Field(i).Interface())
		}
	}
}
