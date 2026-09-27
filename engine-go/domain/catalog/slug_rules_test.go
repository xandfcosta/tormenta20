package catalog_test

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"t20engine/domain/catalog"
)

// O `id` É O ENDEREÇO, E ENDEREÇO NÃO CARREGA RÓTULO (ALE-404).
//
// Com o `uid` fazendo o papel de identidade (ALE-402), sobra para o `id` o
// papel de SLUG: o que a URL mostra, no padrão `/<recurso>/<id>` que o
// `/mestre/encontros/adicionar/ogro` já usa.
//
// Antes desta fatia, dois arquivos descreviam as mesmas raças e dois as mesmas
// origens, e NENHUM id coincidia byte a byte — `races.json` dizia `anao` e
// `race-defs.json` dizia `Anão`. Os cinquenta e dois coincidiam depois de
// slugificar, o que é outro jeito de dizer que a grafia era o único desacordo.
//
// O ponto duro é que em `race-defs.json` o id ERA o rótulo, e o `raceId` das
// habilidades apontava para ele. Renomear a raça na tela renomeava a chave, e
// o `getRace` falha devolvendo `nil` — a raça simplesmente deixa de conceder,
// sem log e sem tela vermelha.
//
// A forma é PERMITIDOS e não proibidos: uma lista de grafias feias subcontaria
// em silêncio a próxima que ninguém previu.
var formaDeSlug = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*(\.[a-z0-9]+(-[a-z0-9]+)*)*$`)

// TestEveryCatalogIdIsASlug varre TODO objeto com `id`, em qualquer
// profundidade de qualquer catálogo, e não tem linha de base.
//
// O ponto na forma é namespacing deliberado, não display: `class.barbaro.impeto`
// diz que o Ímpeto do bárbaro é concessão daquela classe, e cada pedaço entre
// pontos é um slug por si. O que a forma recusa é caixa alta, acento e espaço —
// que é o que o rótulo tem e o endereço não pode ter.
func TestEveryCatalogIdIsASlug(t *testing.T) {
	ids := lerIds(t)
	if len(ids) < 1900 {
		t.Fatalf("só %d ids no catálogo — eram 2163 quando isto foi escrito, e o "+
			"caso não estaria medindo nada", len(ids))
	}

	reprovados := map[string][]string{}
	for _, i := range ids {
		if !formaDeSlug.MatchString(i.id) {
			reprovados[i.arquivo] = append(reprovados[i.arquivo], i.id)
		}
	}
	if len(reprovados) == 0 {
		return
	}

	arquivos := make([]string, 0, len(reprovados))
	for a := range reprovados {
		arquivos = append(arquivos, a)
	}
	sort.Strings(arquivos)
	for _, a := range arquivos {
		fora := reprovados[a]
		sort.Strings(fora)
		t.Errorf("%s: %d ids não são slug — o id é o ENDEREÇO, e ele não pode ter "+
			"caixa alta, acento nem espaço. Slugifique e acerte quem aponta para "+
			"ele (o `raceId` das habilidades, o `origin` da ficha). Os primeiros: %v",
			a, len(fora), fora[:min(6, len(fora))])
	}
}

// TestNoSlugAddressesTwoTopLevelEntries — o slug é o que a URL carrega, e duas
// linhas do mesmo recurso com o mesmo slug fazem `/<recurso>/<id>` resolver
// para uma das duas, em silêncio.
//
// Ele olha só o TOPO do arquivo, e a razão é que `/<recurso>/<id>` endereça uma
// linha de topo — o benefício de origem e a habilidade de raça moram dentro de
// um verbete e não têm URL própria.
//
// O primeiro rascunho varria a árvore inteira e acusou 36 poderes de classe
// repetidos. Não eram: `{"id": "class.inventor.engenhoqueiro", "kind": "power"}`
// é o PRÉ-REQUISITO apontando para o poder, e uma referência não é uma
// definição. Contar as duas responde sobre quantas vezes o nome aparece, que
// não é a pergunta.
//
// A pergunta é POR ARQUIVO de propósito: `acrobacia` é uma perícia e um poder
// geral, e as duas URLs são diferentes porque o recurso é diferente.
func TestNoSlugAddressesTwoTopLevelEntries(t *testing.T) {
	arquivos, err := filepath.Glob("data/*.json")
	if err != nil || len(arquivos) == 0 {
		t.Fatalf("não achei os catálogos em data/*.json: %v", err)
	}

	medidos := 0
	for _, caminho := range arquivos {
		linhas := linhasDeTopo(t, caminho)
		if linhas == nil {
			continue
		}
		vistos := map[string]string{}
		for _, linha := range linhas {
			id, tem := linha["id"].(string)
			if !tem {
				continue
			}
			medidos++
			rotulo, _ := linha["name"].(string)
			if anterior, repetido := vistos[id]; repetido && anterior != rotulo {
				t.Errorf("%s: o slug %q endereça duas linhas de topo diferentes "+
					"(%q e %q) — a URL `/<recurso>/%s` resolveria para uma delas e "+
					"ninguém saberia qual", filepath.Base(caminho), id, anterior,
					rotulo, id)
			}
			vistos[id] = rotulo
		}
	}
	if medidos < 1400 {
		t.Fatalf("só %d linhas de topo medidas — eram 1560 quando isto foi escrito, "+
			"e o caso não estaria olhando o catálogo inteiro", medidos)
	}
}

type entradaComId struct {
	arquivo string
	id      string
}

func lerIds(t *testing.T) (todos []entradaComId) {
	t.Helper()
	arquivos, err := filepath.Glob("data/*.json")
	if err != nil || len(arquivos) == 0 {
		t.Fatalf("não achei os catálogos em data/*.json: %v", err)
	}
	for _, caminho := range arquivos {
		nome := strings.TrimSuffix(filepath.Base(caminho), ".json")
		raw, ok := catalog.Resource(nome)
		if !ok {
			continue
		}
		var arvore any
		if json.Unmarshal(raw, &arvore) != nil {
			continue
		}
		for _, no := range comId(arvore) {
			todos = append(todos, entradaComId{
				arquivo: filepath.Base(caminho),
				id:      no["id"].(string),
			})
		}
	}
	return todos
}

// linhasDeTopo devolve as entradas de primeiro nível, seja o arquivo uma lista
// ou um mapa — `origins-source.json` é mapa e `origins.json` é lista.
func linhasDeTopo(t *testing.T, caminho string) []map[string]any {
	t.Helper()
	raw, ok := catalog.Resource(strings.TrimSuffix(filepath.Base(caminho), ".json"))
	if !ok {
		return nil
	}
	var arvore any
	if json.Unmarshal(raw, &arvore) != nil {
		return nil
	}
	fora := []map[string]any{}
	switch v := arvore.(type) {
	case []any:
		for _, item := range v {
			if m, é := item.(map[string]any); é {
				fora = append(fora, m)
			}
		}
	case map[string]any:
		for _, item := range v {
			if m, é := item.(map[string]any); é {
				fora = append(fora, m)
			}
		}
	}
	return fora
}

// comId: todo objeto que TEM id, onde quer que esteja na árvore — o benefício
// de origem mora dentro da origem, e a habilidade de raça dentro da raça.
func comId(no any) []map[string]any {
	fora := []map[string]any{}
	switch v := no.(type) {
	case map[string]any:
		if _, tem := v["id"].(string); tem {
			fora = append(fora, v)
		}
		chaves := make([]string, 0, len(v))
		for k := range v {
			chaves = append(chaves, k)
		}
		sort.Strings(chaves)
		for _, k := range chaves {
			fora = append(fora, comId(v[k])...)
		}
	case []any:
		for _, item := range v {
			fora = append(fora, comId(item)...)
		}
	}
	return fora
}
