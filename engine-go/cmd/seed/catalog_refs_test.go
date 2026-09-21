package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// O SEED QUE MENTE É CARO DE UM JEITO ESPECÍFICO.
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
	cases := []struct {
		field string
		wrong string
		wait  string
	}{
		{"races", "Anãoo", "races"},
		{"origin", "Acóito", "origin"},
		{"className", "Arcanistaa", "classes"},
		{"god", "Alihanna", "god"},
		{"godPower", "Benção do Mana", "godPower"},
		{"spell", "bola-de-fog", "magia"},
		{"item", "machado-de-batalha", "item"},
	}
	for _, tc := range cases {
		t.Run(tc.field, func(t *testing.T) {
			sf := sujaOCampo(t, tc.field, tc.wrong)
			err := validateCatalogRefs(sf)
			if err == nil {
				t.Fatalf("%s = %q passou em silêncio: o personagem nasceria sem isso e a ficha abriria normal",
					tc.field, tc.wrong)
			}
			// A MENSAGEM carrega o valor ofensor, que é a regra da casa — e aqui
			// ela vale duplo, porque quem a lê está a meio caminho de escrever um
			// teste sobre esse personagem.
			if !strings.Contains(err.Error(), tc.wrong) {
				t.Errorf("a mensagem não diz qual valor está errado: %v", err)
			}
			if !strings.Contains(err.Error(), tc.wait) {
				t.Errorf("a mensagem não diz de que catálogo se trata (esperava %q): %v", tc.wait, err)
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
func sujaOCampo(t *testing.T, field, wrong string) seedFile {
	t.Helper()
	var sf seedFile
	if err := json.Unmarshal(seedData, &sf); err != nil {
		t.Fatalf("seed-data.json: %v", err)
	}
	for iu := range sf.Users {
		for ic := range sf.Users[iu].Characters {
			ch := &sf.Users[iu].Characters[ic]
			if field == "spell" {
				if len(ch.Spells) == 0 {
					continue
				}
				ch.Spells[0].ID = wrong
				return sf
			}
			var create map[string]json.RawMessage
			if err := json.Unmarshal(ch.Create, &create); err != nil {
				t.Fatalf("create do personagem %d: %v", ic, err)
			}
			if !sujaOCriar(t, create, field, wrong) {
				continue
			}
			raw, err := json.Marshal(create)
			if err != nil {
				t.Fatalf("remontar o create: %v", err)
			}
			ch.Create = raw
			return sf
		}
	}
	t.Fatalf("nenhum personagem do seed tem o campo %q: o caso mediria um seed que não existe", field)
	return sf
}

func sujaOCriar(t *testing.T, create map[string]json.RawMessage, field, wrong string) bool {
	t.Helper()
	switch field {
	case "races":
		return trocaNaLista(t, create, "races", wrong)
	case "item":
		return trocaNoItem(t, create, wrong)
	case "className":
		return trocaNaClasse(t, create, wrong)
	default: // origin, god, godPower — todos strings simples
		if _, found := create[field]; !found {
			return false
		}
		raw, err := json.Marshal(wrong)
		if err != nil {
			t.Fatalf("marshal de %q: %v", wrong, err)
		}
		create[field] = raw
		return true
	}
}

func trocaNaLista(t *testing.T, create map[string]json.RawMessage, key, wrong string) bool {
	t.Helper()
	raw, found := create[key]
	if !found {
		return false
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err != nil || len(list) == 0 {
		return false
	}
	list[0] = wrong
	novo, err := json.Marshal(list)
	if err != nil {
		t.Fatalf("marshal de %s: %v", key, err)
	}
	create[key] = novo
	return true
}

func trocaNaClasse(t *testing.T, create map[string]json.RawMessage, wrong string) bool {
	t.Helper()
	raw, found := create["classes"]
	if !found {
		return false
	}
	var classes []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &classes); err != nil || len(classes) == 0 {
		return false
	}
	name, err := json.Marshal(wrong)
	if err != nil {
		t.Fatalf("marshal de %q: %v", wrong, err)
	}
	classes[0]["className"] = name
	novo, err := json.Marshal(classes)
	if err != nil {
		t.Fatalf("marshal das classes: %v", err)
	}
	create["classes"] = novo
	return true
}

func trocaNoItem(t *testing.T, create map[string]json.RawMessage, wrong string) bool {
	t.Helper()
	raw, found := create["items"]
	if !found {
		return false
	}
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil || len(items) == 0 {
		return false
	}
	id, err := json.Marshal(wrong)
	if err != nil {
		t.Fatalf("marshal de %q: %v", wrong, err)
	}
	items[0]["catalogId"] = id
	novo, err := json.Marshal(items)
	if err != nil {
		t.Fatalf("marshal dos itens: %v", err)
	}
	create["items"] = novo
	return true
}
