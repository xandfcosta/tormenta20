package catalog_test

import (
	"encoding/json"
	"strings"
	"testing"

	"t20engine/domain/catalog"
)

// OS PODERES CONCEDIDOS MORAM EM DOIS ARQUIVOS, E ONDE OS DOIS FALAM A REGRA É A
// MESMA.
//
// A divisão é deliberada e o `catalog.go` a explica (ALE-397): o
// `divine-powers.json` é a lista COMPLETA, uma linha por par (deus, poder), e o
// `granted-powers.json` é o subconjunto que o motor alcança, com `deuses[]` e
// `modifiers`. O primeiro tem 80 pares, o segundo 48, e os 48 são um recorte dos
// 80 — isso não é defeito.
//
// Defeito é o TEXTO divergir. Os dois descrevem a mesma regra da mesma página, e
// ninguém os comparava: a ALE-408 achou DOZE pares com texto diferente, e em
// todos os doze quem tinha desviado do livro era o `granted`. Dois mudavam a
// regra — a "Fé Guerreira" perdera o "em combate" (p133) e o "Habitante do
// Deserto" dizia "água boa" onde a p134 diz "água pura".
//
// É a mesma costura que o `TestEveryOriginSaysTheSameThingInBothFiles` faz para
// as origens, e ela dispensa o PDF: comparar duas cópias uma com a outra roda na
// CI, onde o livro não existe por ser gitignorado.
func TestEveryGrantedPowerSaysTheSameThingInBothFiles(t *testing.T) {
	daRegra := lerOsConcedidos(t)
	daLista := lerOsDivinos(t)

	comuns := 0
	for par, regra := range daRegra {
		descricao, existe := daLista[par]
		if !existe {
			t.Errorf("%s: o par está em granted-powers.json e não em "+
				"divine-powers.json, que é a lista COMPLETA — ou o deus está errado "+
				"num dos dois, ou o poder entrou só no subconjunto", par)
			continue
		}
		comuns++
		if strings.TrimSpace(regra) != strings.TrimSpace(descricao) {
			t.Errorf("%s: a regra difere entre os dois arquivos.\n  granted: %q\n"+
				"  divine : %q\nOs dois descrevem a mesma página. Confira QUAL está "+
				"certo contra o livro antes de alinhar — na ALE-408 foi o `divine` nas "+
				"doze vezes, mas isso é medição, não regra", par, regra, descricao)
		}
	}
	// O CONTROLE: os dois mapas tinham o que comparar, e o recorte é o esperado.
	// Zero divergentes sobre zero pares lidos tem no terminal a mesma cara de
	// dois catálogos que concordam.
	if comuns != 48 || len(daLista) != 80 {
		t.Errorf("comparei %d pares e li %d no divine; eram 48 de 80 quando isto foi "+
			"escrito. Se o recorte mudou de propósito, mude estes números junto",
			comuns, len(daLista))
	}
}

func lerOsConcedidos(t *testing.T) map[string]string {
	t.Helper()
	raw, ok := catalog.Resource("granted-powers")
	if !ok {
		t.Fatal("catálogo granted-powers ausente")
	}
	var lista []struct {
		Name   string   `json:"name"`
		Deuses []string `json:"deuses"`
		Effect string   `json:"effect"`
	}
	if err := json.Unmarshal(raw, &lista); err != nil {
		t.Fatalf("granted-powers: %v", err)
	}
	fora := map[string]string{}
	for _, poder := range lista {
		for _, deus := range poder.Deuses {
			fora[parDeConcedido(deus, poder.Name)] = poder.Effect
		}
	}
	return fora
}

func lerOsDivinos(t *testing.T) map[string]string {
	t.Helper()
	raw, ok := catalog.Resource("divine-powers")
	if !ok {
		t.Fatal("catálogo divine-powers ausente")
	}
	var lista []struct {
		DeusID      string `json:"deusId"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(raw, &lista); err != nil {
		t.Fatalf("divine-powers: %v", err)
	}
	fora := map[string]string{}
	for _, poder := range lista {
		fora[parDeConcedido(poder.DeusID, poder.Name)] = poder.Description
	}
	return fora
}

// parDeConcedido: os dois arquivos escrevem o deus de formas diferentes — o
// `granted` guarda o NOME ("Lin-Wu") e o `divine` guarda o id ("linwu"). Sem
// normalizar, nenhum par casa e o caso passa verde sobre zero comparações; o
// `chaveDoConceito` (uid_rules_test.go) já é essa normalização.
func parDeConcedido(deus, poder string) string {
	return chaveDoConceito(deus) + "/" + chaveDoConceito(poder)
}
