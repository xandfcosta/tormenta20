package book

import "testing"

// A CONCESSÃO ENXERGA A REGRA DO VERBETE (ALE-403).
//
// O `class-powers.json` passou a ter duas espécies de linha: o verbete, com a
// regra, e a concessão, com a classe e o nível mais um `powerUid`. Quem junta
// as duas é o `classPowersResolved`, e o risco desta fatia é ele não juntar —
// a aba de Poderes mostraria o nome vazio e a descrição sumiria, sem erro
// nenhum.
//
// O caso mede pelo CONSUMIDOR e não pela função: é o `ClassPowers()` que
// a tela usa, e é ele que tem de devolver a regra.
func TestTheGrantSeesTheRuleOfItsEntry(t *testing.T) {
	powers := ClassPowers()
	if len(powers) < 400 {
		t.Fatalf("só %d poderes de classe — eram 462 quando isto foi escrito, e o "+
			"caso não estaria medindo nada", len(powers))
	}

	// O VERBETE não é um poder que alguém possui: ele não tem classe, e a aba
	// de Poderes pergunta sempre "o que esta classe me deu".
	if _, tem := powers["poder.aumento-de-atributo"]; tem {
		t.Error("o verbete `poder.aumento-de-atributo` apareceu na lista de poderes " +
			"possuíveis — ele é a REGRA, não uma concessão de classe")
	}

	// A última coluna da tabela diz se aquele poder é destravado por nível ou
	// escolha de classe. O Aumento de Atributo não é nenhum dos dois: ele é
	// ESCOLHÍVEL, e
	// quem descreve como escolhê-lo é o bloco `choice` do verbete. Exigir posse
	// dele seria inventar uma regra que o livro não tem.
	for _, caso := range []struct {
		id       string
		classe   string
		temPosse bool
	}{
		{"class.barbaro.aumento-de-atributo", "Bárbaro", false},
		{"class.arcanista.aumento-de-atributo", "Arcanista", false},
		{"class.ladino.evasao", "Ladino", true},
		{"class.bardo.magias-2-circulo", "Bardo", true},
	} {
		p, tem := powers[caso.id]
		if !tem {
			t.Errorf("%s sumiu do catálogo — a concessão tem de sobreviver à divisão",
				caso.id)
			continue
		}
		if p.Name == "" {
			t.Errorf("%s ficou sem NOME: o `powerUid` não resolveu, e a aba de Poderes "+
				"desenharia uma linha em branco", caso.id)
		}
		if p.Description == "" {
			t.Errorf("%s ficou sem DESCRIÇÃO pelo mesmo motivo", caso.id)
		}
		if p.ClassName != caso.classe {
			t.Errorf("%s diz que é da classe %q e devia ser %q — o verbete não pode "+
				"sobrescrever a concessão", caso.id, p.ClassName, caso.classe)
		}
		if caso.temPosse && p.GrantedAtLevel == nil && p.GrantedByChoice == nil {
			t.Errorf("%s perdeu a regra de posse: sem nível nem escolha, ninguém o "+
				"recebe nunca", caso.id)
		}
	}

	// O MESMO poder por duas classes tem a MESMA regra — que é o ponto inteiro
	// da divisão. Antes eram duas cópias que podiam divergir; agora é uma.
	barbaro := powers["class.barbaro.aumento-de-atributo"]
	arcanista := powers["class.arcanista.aumento-de-atributo"]
	if barbaro.Description != arcanista.Description {
		t.Errorf("o Aumento de Atributo do Bárbaro e o do Arcanista têm descrições "+
			"diferentes, e eles apontam para o MESMO verbete:\n  bárbaro:   %q\n"+
			"  arcanista: %q", barbaro.Description, arcanista.Description)
	}
}
