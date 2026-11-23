package sheet

import "testing"

// O CÍRCULO QUE A FICHA ALCANÇA É POR MAGIA (p170-171): o das classes que a lançam, e não o maior entre todas;
// sem nenhuma delas, o das classes que conjuram (um poder ensinou a magia de fora da lista, como o Aspecto da
// Primavera, p61); e, sem classe que conjure, o da própria magia, que veio de fora das classes.
func TestTheCircleReachIsTheOneOfTheClassesThatCastTheSpell(t *testing.T) {
	arcaneAndDivine := CharacterDTO{Classes: []ClassDTO{{ClassName: "Arcanista", Level: 9}, {ClassName: "Clérigo", Level: 1}}}
	for _, c := range []struct {
		why   string
		dto   CharacterDTO
		spell string
		want  int
	}{
		// O arcanista de 9º nível alcança o 3º círculo; o clérigo de 1º, só o 1º.
		{"a magia arcana pelo arcanista de 9º", arcaneAndDivine, "bola-de-fogo", 3},
		{"a magia divina pelo clérigo de 1º, e não pelo arcanista", arcaneAndDivine, "curar-ferimentos", 1},
		// O arcanista de 4º nível alcança só o 1º: o 2º círculo vem no 5º nível.
		{"o arcanista de 4º", CharacterDTO{Classes: []ClassDTO{{ClassName: "Arcanista", Level: 4}}}, "bola-de-fogo", 1},
		// A Metamorfose é só arcana: o druida não a lança pela lista, e vale o círculo que ele alcança.
		{"a magia de fora da lista, pelo druida", CharacterDTO{Classes: []ClassDTO{{ClassName: "Druida", Level: 1}}}, "metamorfose", 1},
		{"a magia de quem não conjura vale o círculo dela", CharacterDTO{Classes: []ClassDTO{{ClassName: "Bárbaro", Level: 2}}}, "bola-de-fogo", 2},
	} {
		if got := SpellCircleReach(c.dto, c.spell); got != c.want {
			t.Errorf("%s: alcança o %dº círculo para %s; want %dº", c.why, got, c.spell, c.want)
		}
	}
}
