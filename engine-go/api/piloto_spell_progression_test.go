package api

import (
	"testing"

	"t20engine/catalog"
)

// Os guardas da PROGRESSÃO DE CÍRCULO e da fronteira que ela fechou
// (ALE-272, fatia 6).

// A PROGRESSÃO SAI DO CATÁLOGO, com as cinco classes que conjuram.
//
// Os números são os que estavam no front e foram MOVIDOS — nenhum foi lido do
// livro aqui, e o que garante a fidelidade da mudança é o
// `spell-progression-agree.test.ts`, que compara as duas cópias enquanto a SPA
// viver. O que este caso prende é que a leitura FUNCIONA: um `classes.json` sem
// o campo, ou um `Resource` que parasse de casar, daria um mapa vazio e o
// portão de círculo aceitaria tudo em silêncio.
func TestAProgressaoDeCirculoSaiDoCatalogo(t *testing.T) {
	prog := spellProgressions()
	if len(prog) != 5 {
		t.Fatalf("o catálogo ofereceu %d classes conjuradoras, quer 5", len(prog))
	}
	for _, nome := range []string{"Arcanista", "Bardo", "Clérigo", "Druida", "Paladino"} {
		if _, tem := prog[nome]; !tem {
			t.Errorf("a classe %q não trouxe progressão", nome)
		}
	}
	// O Bardo PARA no 4º círculo, e é o caso que separa "não alcançou ainda" de
	// "nunca alcança": o 5º dele é nulo, não um nível alto.
	bardo := prog["Bardo"]
	if bardo.MaxCircle != 4 {
		t.Errorf("o Bardo vai até o %dº círculo, quer o 4º", bardo.MaxCircle)
	}
	if bardo.UnlockLevel["5"] != nil {
		t.Error("o 5º círculo do Bardo veio com nível: ele nunca chega lá")
	}
}

// O CÍRCULO ALCANÇÁVEL sobe com o nível, e o piso é o da PRÓPRIA magia.
//
// O piso é a regra que não se adivinha: uma magia concedida por poder (Totem
// Espiritual, p42) é conjurável por quem não tem classe de conjurador nenhuma —
// no círculo dela, e só nele. Sem o piso, um bárbaro com Totem não conjuraria a
// magia que o poder acabou de lhe dar.
func TestOCirculoAlcancavelSobeComONivelETemPisoNaPropriaMagia(t *testing.T) {
	casos := []struct {
		nome    string
		classes []ClassDTO
		magia   int
		quer    int
	}{
		{"arcanista de 1º alcança o 1º", []ClassDTO{{ClassName: "Arcanista", Level: 1}}, 1, 1},
		{"arcanista de 4º ainda está no 1º", []ClassDTO{{ClassName: "Arcanista", Level: 4}}, 1, 1},
		{"arcanista de 5º abre o 2º", []ClassDTO{{ClassName: "Arcanista", Level: 5}}, 1, 2},
		{"arcanista de 17º chega ao 5º", []ClassDTO{{ClassName: "Arcanista", Level: 17}}, 1, 5},
		{"bardo de 20º para no 4º", []ClassDTO{{ClassName: "Bardo", Level: 20}}, 1, 4},
		{"guerreiro não conjura, e o piso é a magia", []ClassDTO{{ClassName: "Guerreiro", Level: 20}}, 2, 2},
		{"multiclasse fica com o MAIOR", []ClassDTO{
			{ClassName: "Guerreiro", Level: 10}, {ClassName: "Arcanista", Level: 9},
		}, 1, 3},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			if got := highestCastableCircle(caso.classes, caso.magia); got != caso.quer {
				t.Errorf("alcançou o %dº, quer o %dº", got, caso.quer)
			}
		})
	}
}

// A FRONTEIRA DO CÍRCULO É DO SERVIDOR, e não da tela.
//
// # O buraco que este guarda fecha
//
// 126 dos 486 aprimoramentos do catálogo exigem um círculo mínimo, e até a fatia
// 6 esse limite existia SÓ na interface: a tabela que o decide vivia no
// TypeScript, então o `validateAugments` nem tinha como perguntar. Um pedido
// montado à mão conjurava o que a regra não permite, e nada acusava.
func TestOAprimoramentoAcimaDoCirculoERecusadoPeloServidor(t *testing.T) {
	magia, conhecida := catalog.LookupSpell("invisibilidade")
	if !conhecida {
		t.Fatal("a magia `invisibilidade` sumiu do catálogo — o caso mediria o vazio")
	}
	alvo := -1
	for i, a := range magia.Augments {
		if a.RequiresCircle != nil && *a.RequiresCircle >= 3 {
			alvo = i
			break
		}
	}
	if alvo < 0 {
		t.Fatal("a Invisibilidade não tem aprimoramento com `requiresCircle` ≥ 3: " +
			"ou o catálogo mudou, ou o campo parou de ser lido — nos dois casos este " +
			"guarda mediria o vazio")
	}
	picks := []augmentPick{{AugmentIndex: alvo, Stacks: 1}}

	// QUEM NÃO ALCANÇA é recusado…
	if _, erro := validateAugments(magia, picks, 2); erro == "" {
		t.Error("um aprimoramento de círculo alto passou para quem alcança só o 2º")
	}
	// …e quem alcança, passa. Sem esta metade o guarda ficaria verde numa
	// validação que recusa TUDO.
	if _, erro := validateAugments(magia, picks, 5); erro != "" {
		t.Errorf("quem alcança o 5º foi recusado: %s", erro)
	}
}
