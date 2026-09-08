package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// O SEED QUE MENTE É CARO DE UM JEITO ESPECÍFICO (ALE-226).
//
// Ele não quebra nada: ele produz um personagem QUASE certo. O relato que abriu
// a issue foi `machado-de-batalha` no lugar de `machado-batalha` — o personagem
// nasceu SEM a arma, a ficha abriu normal, e não houve erro em lugar nenhum. O
// e2e roda contra a seed, então um combatente sem a arma dele vira um teste
// medindo o ambiente e não o app.
//
// Só `create.items[].catalogId` era conferido. Tudo o mais que aponta para o
// catálogo — raça, origem, classe, deus, poder concedido, magia — entrava cru.
func TestTheSeedRefusesEveryUnknownCatalogReference(t *testing.T) {
	casos := []struct {
		campo   string
		errado  string
		esperar string
	}{
		{"races", "Anãoo", "races"},
		{"origin", "Acóito", "origin"},
		{"className", "Arcanistaa", "classes"},
		{"god", "Alihanna", "god"},
		{"godPower", "Benção do Mana", "godPower"},
		{"spell", "bola-de-fog", "magia"},
		{"item", "machado-de-batalha", "item"},
	}
	for _, caso := range casos {
		t.Run(caso.campo, func(t *testing.T) {
			sf := sujaOCampo(t, caso.campo, caso.errado)
			err := validateCatalogRefs(sf)
			if err == nil {
				t.Fatalf("%s = %q passou em silêncio: o personagem nasceria sem isso e a ficha abriria normal",
					caso.campo, caso.errado)
			}
			// A MENSAGEM carrega o valor ofensor, que é a regra da casa — e aqui
			// ela vale duplo, porque quem a lê está a meio caminho de escrever um
			// teste sobre esse personagem.
			if !strings.Contains(err.Error(), caso.errado) {
				t.Errorf("a mensagem não diz qual valor está errado: %v", err)
			}
			if !strings.Contains(err.Error(), caso.esperar) {
				t.Errorf("a mensagem não diz de que catálogo se trata (esperava %q): %v", caso.esperar, err)
			}
		})
	}
}

// O VIZINHO, porque errar id é ERRO DE DIGITAÇÃO, e digitação tem vizinho.
//
// `machado-de-batalha` está a distância 3 de `machado-batalha` num catálogo que
// também tem `machado-guerra`, `machado-anao` e `machado-taurico` — sem a
// sugestão, quem lê a mensagem vai procurar o id certo na mão.
func TestTheRefusalSuggestsTheNeighbourWhenThereIsOne(t *testing.T) {
	err := validateCatalogRefs(sujaOCampo(t, "item", "machado-de-batalha"))
	if err == nil {
		t.Fatal("o item errado passou: o resto deste caso não mediria nada")
	}
	if !strings.Contains(err.Error(), "machado-batalha") {
		t.Errorf("a mensagem não sugere o vizinho óbvio: %v", err)
	}
}

// O SEED DE VERDADE PASSA, e sem isto os casos acima passariam com um validador
// que recusa tudo — que é o modo de falhar mais fácil de escrever.
func TestTheRealSeedHasNoUnknownCatalogReference(t *testing.T) {
	var sf seedFile
	if err := json.Unmarshal(seedData, &sf); err != nil {
		t.Fatalf("seed-data.json: %v", err)
	}
	if len(sf.Users) == 0 {
		t.Fatal("o seed veio sem usuários: o que vem abaixo não mediria nada")
	}
	if err := validateCatalogRefs(sf); err != nil {
		t.Errorf("o seed do repositório tem referência quebrada: %v", err)
	}
}

// sujaOCampo devolve o seed de verdade com UM valor trocado por um errado.
//
// Do seed de verdade e não de um fixture à mão: um fixture teria a forma que eu
// imagino, e o que precisa ser conferido é a forma que o arquivo TEM.
func sujaOCampo(t *testing.T, campo, errado string) seedFile {
	t.Helper()
	var sf seedFile
	if err := json.Unmarshal(seedData, &sf); err != nil {
		t.Fatalf("seed-data.json: %v", err)
	}
	for iu := range sf.Users {
		for ic := range sf.Users[iu].Characters {
			ch := &sf.Users[iu].Characters[ic]
			if campo == "spell" {
				if len(ch.Spells) == 0 {
					continue
				}
				ch.Spells[0].ID = errado
				return sf
			}
			var criar map[string]json.RawMessage
			if err := json.Unmarshal(ch.Create, &criar); err != nil {
				t.Fatalf("create do personagem %d: %v", ic, err)
			}
			if !sujaOCriar(t, criar, campo, errado) {
				continue
			}
			bruto, err := json.Marshal(criar)
			if err != nil {
				t.Fatalf("remontar o create: %v", err)
			}
			ch.Create = bruto
			return sf
		}
	}
	t.Fatalf("nenhum personagem do seed tem o campo %q: o caso mediria um seed que não existe", campo)
	return sf
}

func sujaOCriar(t *testing.T, criar map[string]json.RawMessage, campo, errado string) bool {
	t.Helper()
	switch campo {
	case "races":
		return trocaNaLista(t, criar, "races", errado)
	case "item":
		return trocaNoItem(t, criar, errado)
	case "className":
		return trocaNaClasse(t, criar, errado)
	default: // origin, god, godPower — todos strings simples
		if _, tem := criar[campo]; !tem {
			return false
		}
		bruto, err := json.Marshal(errado)
		if err != nil {
			t.Fatalf("marshal de %q: %v", errado, err)
		}
		criar[campo] = bruto
		return true
	}
}

func trocaNaLista(t *testing.T, criar map[string]json.RawMessage, chave, errado string) bool {
	t.Helper()
	bruto, tem := criar[chave]
	if !tem {
		return false
	}
	var lista []string
	if err := json.Unmarshal(bruto, &lista); err != nil || len(lista) == 0 {
		return false
	}
	lista[0] = errado
	novo, err := json.Marshal(lista)
	if err != nil {
		t.Fatalf("marshal de %s: %v", chave, err)
	}
	criar[chave] = novo
	return true
}

func trocaNaClasse(t *testing.T, criar map[string]json.RawMessage, errado string) bool {
	t.Helper()
	bruto, tem := criar["classes"]
	if !tem {
		return false
	}
	var classes []map[string]json.RawMessage
	if err := json.Unmarshal(bruto, &classes); err != nil || len(classes) == 0 {
		return false
	}
	nome, err := json.Marshal(errado)
	if err != nil {
		t.Fatalf("marshal de %q: %v", errado, err)
	}
	classes[0]["className"] = nome
	novo, err := json.Marshal(classes)
	if err != nil {
		t.Fatalf("marshal das classes: %v", err)
	}
	criar["classes"] = novo
	return true
}

func trocaNoItem(t *testing.T, criar map[string]json.RawMessage, errado string) bool {
	t.Helper()
	bruto, tem := criar["items"]
	if !tem {
		return false
	}
	var itens []map[string]json.RawMessage
	if err := json.Unmarshal(bruto, &itens); err != nil || len(itens) == 0 {
		return false
	}
	id, err := json.Marshal(errado)
	if err != nil {
		t.Fatalf("marshal de %q: %v", errado, err)
	}
	itens[0]["catalogId"] = id
	novo, err := json.Marshal(itens)
	if err != nil {
		t.Fatalf("marshal dos itens: %v", err)
	}
	criar["items"] = novo
	return true
}
