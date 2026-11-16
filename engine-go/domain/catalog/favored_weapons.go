package catalog

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// A ARMA PREFERIDA DA DIVINDADE, como arma do catálogo (ALE-652). O verbete do deus a escreve pelo NOME, como
// o livro ("Espada longa", p96-106), e quem a usa é o motor — a Arma Sagrada troca o dado do Golpe Divino só
// nela (p82) —, que conhece a arma pelo id. A ligação é feita aqui, pelo nome do item, e não escrevendo o id
// no verbete: seriam duas fontes para o mesmo fato, e a do nome é a que se confere no livro.
//
// Nimb tem "Nenhuma e todas!" (p102) — qualquer arma, de acordo com o mestre —, e fica fora do mapa: quem
// decide é o mestre, na voz. Lena e Marah não têm arma preferida.

// everyWeaponAtTheMastersCall é o "todas" de Nimb (p102).
const everyWeaponAtTheMastersCall = "todas"

// favoredWeapons devolve, por id da divindade, o id da arma preferida dela no catálogo de itens. Um nome que
// não acha UMA arma é erro, com o deus e o nome: a Arma Sagrada calaria em silêncio para os devotos dele.
func favoredWeapons(gods, items json.RawMessage) (map[string]string, error) {
	var deities []struct {
		ID            string  `json:"id"`
		FavoredWeapon *string `json:"armaPreferida"`
	}
	if err := json.Unmarshal(gods, &deities); err != nil {
		return nil, fmt.Errorf("gods.json devia ser uma lista de deuses: %w", err)
	}
	var catalogItems []struct {
		ID     string          `json:"id"`
		Name   string          `json:"name"`
		Weapon json.RawMessage `json:"weapon"`
	}
	if err := json.Unmarshal(items, &catalogItems); err != nil {
		return nil, fmt.Errorf("items.json devia ser uma lista de itens: %w", err)
	}
	weaponsByName := map[string][]string{}
	for _, item := range catalogItems {
		if item.Weapon != nil {
			name := strings.ToLower(item.Name)
			weaponsByName[name] = append(weaponsByName[name], item.ID)
		}
	}
	out := map[string]string{}
	for _, god := range deities {
		if god.FavoredWeapon == nil || *god.FavoredWeapon == everyWeaponAtTheMastersCall {
			continue
		}
		found := weaponsByName[strings.ToLower(*god.FavoredWeapon)]
		if len(found) != 1 {
			sort.Strings(found)
			return nil, fmt.Errorf("a arma preferida de %q é %q, e o catálogo de itens tem %d armas com esse nome %v; "+
				"want exatamente uma", god.ID, *god.FavoredWeapon, len(found), found)
		}
		out[god.ID] = found[0]
	}
	return out, nil
}
