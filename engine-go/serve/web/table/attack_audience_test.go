package table

import (
	"reflect"
	"testing"

	"t20engine/domain/live"
)

// TODO CAMPO DO CARTÃO DO ATAQUE DIZ QUEM O VÊ.
//
// O recorte do jogador (`forPlayer`) copiava o cartão do mestre campo a campo, e cada campo
// novo do mestre era esquecido no do jogador até alguém notar: o botão do Destruidor, a
// manobra declarada, as situações do lado dele, o "não letal". Agora o recorte de quem rolou
// parte do cartão INTEIRO e apaga o que é só do mestre — e este guarda exige que cada campo
// esteja numa das quatro listas, e confere o recorte contra ela.
//
// Campo novo reprova aqui com o nome dele até alguém decidir quem o vê.
var attackCardAudience = map[string]string{
	// A cabeça do cartão: todo jogador vê quem ataca quem.
	"Stage": everyone, "Attacker": everyone, "Target": everyone, "TargetEntryID": everyone,
	"Weapon": everyone, "Action": everyone, "Maneuver": everyone, "Mine": everyone,
	// O que é de quem rolou, e só dele.
	"Boost": roller, "Dose": roller, "FollowUp": roller, "Guard": roller, "Rerolled": roller,
	"Offers": roller, "ToRoll": roller, "DamageLines": roller, "Hand": roller, "HandBonus": roller, "Said": roller,
	"DamageTotal": roller, "Destroyer": roller, "DamageRerolled": roller,
	// O que quem rolou vê ESCRITO DE OUTRO JEITO: a conta sem a Defesa, as situações do lado
	// dele, o dano com o "não letal" da arma dele.
	"Tally": rewritten, "Situations": rewritten, "DamageLine": rewritten,
	// O que é só do mestre: a sugestão do livro, a Defesa, a RD, o tipo da criatura (p230).
	"Against": gmOnly, "Duel": gmOnly, "Verdicts": gmOnly, "Hint": gmOnly, "Rows": gmOnly, "Notes": gmOnly,
}

const (
	everyone  = "todos"
	roller    = "quem rolou"
	rewritten = "quem rolou, reescrito"
	gmOnly    = "só o mestre"
)

func TestEveryAttackCardFieldSaysWhoSeesIt(t *testing.T) {
	kind := reflect.TypeOf(attackProposal{})
	for i := range kind.NumField() {
		if _, ok := attackCardAudience[kind.Field(i).Name]; !ok {
			t.Errorf("o campo %s do cartão do ataque não diz quem o vê: ponha-o no `attackCardAudience` "+
				"(%s, %s, %s ou %s) — sem isso, ele nasce visível para quem rolou sem ninguém ter decidido",
				kind.Field(i).Name, everyone, roller, rewritten, gmOnly)
		}
	}
	for name := range attackCardAudience {
		if _, ok := kind.FieldByName(name); !ok {
			t.Errorf("o `attackCardAudience` classifica %s, que o cartão não tem mais: tire a linha", name)
		}
	}

	// A proposta tem o que cada campo REESCRITO lê: a rolagem, as situações do lado de quem
	// ataca e o dano não letal.
	pa := live.PendingAttack{Roll: 12, Total: 19, AttackerSituations: []string{"flanqueando"}, Damage: 9, NonLethal: 9}
	for _, mine := range []bool{true, false} {
		full := filledProposal(t)
		full.Mine = mine
		cut := reflect.ValueOf(*full.forPlayer(pa))
		source := reflect.ValueOf(full)
		for i := range kind.NumField() {
			name := kind.Field(i).Name
			got, was := cut.Field(i), source.Field(i)
			same := reflect.DeepEqual(got.Interface(), was.Interface())
			switch audience := attackCardAudience[name]; {
			case audience == everyone && !same:
				t.Errorf("%s é de todos, e o recorte (de quem rolou: %v) o mudou", name, mine)
			case audience == roller && mine && !same:
				t.Errorf("%s é de quem rolou, e o recorte dele o perdeu", name)
			case audience == rewritten && mine && (same || got.IsZero()):
				t.Errorf("%s é reescrito para quem rolou, e o recorte dele o %s",
					name, map[bool]string{true: "copiou do mestre", false: "apagou"}[same])
			case (audience == roller || audience == rewritten) && !mine && !got.IsZero():
				t.Errorf("%s é de quem rolou, e o recorte de outro jogador o mostra", name)
			case audience == gmOnly && !got.IsZero():
				t.Errorf("%s é só do mestre, e o recorte (de quem rolou: %v) o mostra", name, mine)
			}
		}
	}
}

// filledProposal é o cartão do mestre com TODO campo preenchido, para o recorte ter o que
// apagar e o que copiar em cada um.
func filledProposal(t *testing.T) attackProposal {
	t.Helper()
	var a attackProposal
	value := reflect.ValueOf(&a).Elem()
	for i := range value.NumField() {
		fill(t, value.Field(i), value.Type().Field(i).Name)
	}
	return a
}

func fill(t *testing.T, field reflect.Value, name string) {
	t.Helper()
	switch field.Kind() {
	case reflect.String:
		field.SetString("do mestre: " + name)
	case reflect.Bool:
		field.SetBool(true)
	case reflect.Int:
		field.SetInt(7)
	case reflect.Slice:
		one := reflect.New(field.Type().Elem()).Elem()
		fillStruct(t, one, name)
		field.Set(reflect.Append(reflect.MakeSlice(field.Type(), 0, 1), one))
	case reflect.Struct:
		fillStruct(t, field, name)
	default:
		t.Fatalf("o campo %s é de um tipo (%s) que o guarda não sabe preencher: ensine o `fill`", name, field.Kind())
	}
}

func fillStruct(t *testing.T, v reflect.Value, name string) {
	t.Helper()
	if v.Kind() != reflect.Struct {
		fill(t, v, name)
		return
	}
	for i := range v.NumField() {
		if v.Field(i).CanSet() {
			fill(t, v.Field(i), name+"."+v.Type().Field(i).Name)
		}
	}
}
