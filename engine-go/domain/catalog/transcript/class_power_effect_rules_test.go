package transcript

import (
	"reflect"
	"testing"
)

// classPowerFieldsThatAreNotEffects são os campos do poder de classe que NÃO são efeito no motor, cada um com o
// porquê. Todo o resto tem de fazer o `HasEngineEffect` dizer sim.
var classPowerFieldsThatAreNotEffects = map[string]string{
	"ID":             "identidade",
	"UID":            "identidade",
	"ClassName":      "identidade",
	"Name":           "identidade",
	"Description":    "texto do livro",
	"BookPage":       "citação",
	"GrantedAtLevel": "quando o poder chega, e não o que ele faz",
	"MinLevel":       "quando o poder se pode escolher, e não o que ele faz",
	"Prerequisites":  "quem pode escolher, e não o que ele faz",
	"Modeled":        "a nota do que o motor faz: é a declaração, e o auditor a lê à parte",
	"Unmodeled":      "a nota do que fica com o mestre",
	"PowerUID":       "o ponteiro para o poder geral: o balde é do apontado (APONTA)",
	// A ESCOLHA não é efeito sozinha: ela diz O QUE o jogador escolhe, e o efeito, quando há, mora nos
	// modificadores ou no campo que lê a escolha. O Companheiro Animal tem escolha e nada no motor; contá-la
	// o poria em "em parte" sem nada aplicado.
	"Choice":          "a escolha, e não o que ela faz",
	"GrantedByChoice": "a escolha de outro poder que concede este, e não o que ele faz",
}

// O CAMPO NOVO DECIDE. O auditor de cobertura conta como efeito do poder de classe o que o `HasEngineEffect`
// diz, e a lista dos campos que o motor lê mora ali, ao lado do tipo. Este guarda preenche cada campo do tipo,
// um por vez, num poder vazio: ou o método passa a dizer "efeito", ou o campo está na lista acima. Um campo novo
// que ninguém classificou reprova com o nome dele — sem isso, o motor passaria a ler o campo e o auditor
// continuaria contando o verbete como "só nota", ou sustentado só pela nota escrita à mão.
func TestEveryClassPowerFieldSaysWhetherItIsAnEngineEffect(t *testing.T) {
	if (ClassPower{}).HasEngineEffect() {
		t.Fatal("o poder de classe VAZIO diz que tem efeito no motor")
	}
	kind := reflect.TypeOf(ClassPower{})
	for i := range kind.NumField() {
		field := kind.Field(i)
		var power ClassPower
		fillForTheEffectProbe(t, reflect.ValueOf(&power).Elem().Field(i), field.Name)
		_, notAnEffect := classPowerFieldsThatAreNotEffects[field.Name]
		switch effect := power.HasEngineEffect(); {
		case effect && notAnEffect:
			t.Errorf("o campo %s do poder de classe está na lista do que NÃO é efeito, e o HasEngineEffect o conta: tire-o de um dos dois", field.Name)
		case !effect && !notAnEffect:
			t.Errorf("o campo %s do poder de classe não diz se é efeito no motor: se o motor o lê, ponha-o no HasEngineEffect; se não, em classPowerFieldsThatAreNotEffects, com o porquê", field.Name)
		}
	}
	for name := range classPowerFieldsThatAreNotEffects {
		if _, found := kind.FieldByName(name); !found {
			t.Errorf("classPowerFieldsThatAreNotEffects cita o campo %s, que o poder de classe não tem mais", name)
		}
	}
}

// fillForTheEffectProbe põe no campo o menor valor que não é zero: o ponteiro aponta para um valor vazio, a lista
// ganha um elemento, o texto e o número deixam de ser zero.
func fillForTheEffectProbe(t *testing.T, value reflect.Value, name string) {
	t.Helper()
	switch value.Kind() {
	case reflect.Pointer:
		value.Set(reflect.New(value.Type().Elem()))
	case reflect.Slice:
		value.Set(reflect.MakeSlice(value.Type(), 1, 1))
	case reflect.String:
		value.SetString("x")
	case reflect.Int:
		value.SetInt(1)
	case reflect.Bool:
		value.SetBool(true)
	default:
		t.Fatalf("o campo %s do poder de classe é de um tipo (%s) que a sonda não sabe preencher", name, value.Kind())
	}
}
