package catalog_test

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"testing"

	"t20engine/domain/catalog"
)

// AS ORIGENS MORAM EM DOIS ARQUIVOS, E OS DOIS TÊM DE DIZER A MESMA COISA.
//
// O `origins-source.json` é a transcrição do livro (perícias, poderes, itens,
// página) e o `origins.json` é a mesma origem achatada em benefícios, com o
// benefício de poder APONTANDO para `general-powers` (ALE-401). São as mesmas 35
// origens e elas compartilham o `uid`.
//
// Ninguém conferia se as duas cópias concordavam, e as duas discordavam: o poder
// ÚNICO do Batedor era "À Prova de Tudo" na fonte e "Estilo de Disparo" no
// achatado, e o do Trabalhador era "Atlético" contra "Esforçado" (ALE-405). Nos
// dois casos o achatado tinha um poder GERAL no lugar reservado ao poder
// exclusivo da origem, com uma descrição inventada — e a ficha oferecia o poder
// errado sem erro em lugar nenhum.
//
// Este guarda não precisa do PDF do livro, que é gitignorado: ele compara as duas
// cópias UMA COM A OUTRA e roda na CI. O que precisa do livro é o
// `scripts/audit-origins.py`, e é lá que a comparação contra a página mora.
func TestEveryOriginSaysTheSameThingInBothFiles(t *testing.T) {
	fonte, achatado := lerAsDuasCopiasDeOrigem(t)
	for slug, deLa := range fonte {
		aqui, existe := achatado[slug]
		if !existe {
			t.Errorf("%s está em origins-source.json e não em origins.json", slug)
			continue
		}
		if deLa.Name != aqui.Name {
			t.Errorf("%s: o nome é %q na fonte e %q no achatado", slug, deLa.Name, aqui.Name)
		}
		if deLa.Uid != aqui.Uid {
			t.Errorf("%s: o uid é %q na fonte e %q no achatado — o banco guarda um "+
				"deles, e a ficha deixa de achar a origem pelo outro",
				slug, deLa.Uid, aqui.Uid)
		}
		if deLa.UniquePower != aqui.UniquePower.Name {
			t.Errorf("%s: o poder único é %q na fonte e %q no achatado. Confira qual é "+
				"o poder EXCLUSIVO da origem na página %d do livro — o outro é um "+
				"poder geral, e ele pertence à lista de benefícios",
				slug, deLa.UniquePower, aqui.UniquePower.Name, deLa.BookPage)
		}
		confereOsBeneficiosDerivados(t, slug, deLa, aqui)
	}
	for slug := range achatado {
		if _, existe := fonte[slug]; !existe {
			t.Errorf("%s está em origins.json e não em origins-source.json", slug)
		}
	}
	// O CONTROLE: os dois mapas tinham o que comparar. Um `json` que não casou
	// devolve mapa vazio, e zero reprovados sobre zero origens tem no terminal a
	// mesma cara de trinta e cinco origens que concordam.
	if len(fonte) != 35 || len(achatado) != 35 {
		t.Fatalf("a fonte rendeu %d origens e o achatado %d — o livro tem 35, e o "+
			"caso não estaria medindo nada", len(fonte), len(achatado))
	}
}

// A ESPECIALIZAÇÃO entre parênteses. O livro escreve `Ofício (alquimista)` na
// linha de benefícios, e é essa frase que a fonte guarda; o achatado guarda só
// `Ofício`, porque é a perícia que o motor TREINA — não existe uma perícia
// chamada "Ofício (alquimista)". Comparar a frase inteira reprovaria as quatro
// origens com especialização: Assistente de Laboratório, Fazendeiro, Minerador e
// Taverneiro.
var especializacao = regexp.MustCompile(`\s*\([^)]*\)\s*$`)

// confereOsBeneficiosDerivados: a lista do achatado é `perícias + poderes` da
// fonte MENOS o poder único, que tem campo próprio e não se repete na lista.
//
// O poder "a sua escolha" não tem nome em nenhum dos dois: a fonte o guarda como
// `poderChoiceCategory` ("combate", "tormenta") e o achatado monta um benefício
// com `powerPick`. As cinco origens que o têm falhariam se o guarda comparasse
// nome com frase.
func confereOsBeneficiosDerivados(t *testing.T, slug string, deLa origemDaFonte,
	aqui origemAchatada) {
	t.Helper()
	esperados := make([]string, 0, len(deLa.Expertises)+len(deLa.Powers))
	for _, pericia := range deLa.Expertises {
		esperados = append(esperados, especializacao.ReplaceAllString(pericia, ""))
	}
	for _, poder := range deLa.Powers {
		if poder != deLa.UniquePower {
			esperados = append(esperados, poder)
		}
	}
	achados, escolhas := []string{}, []string{}
	for _, beneficio := range aqui.Benefits {
		if beneficio.PowerPick != "" {
			escolhas = append(escolhas, beneficio.PowerPick)
			continue
		}
		achados = append(achados, beneficio.Name)
	}
	sort.Strings(esperados)
	sort.Strings(achados)
	if strings.Join(esperados, " | ") != strings.Join(achados, " | ") {
		t.Errorf("%s: os benefícios do achatado são [%s] e a fonte deriva [%s]",
			slug, strings.Join(achados, ", "), strings.Join(esperados, ", "))
	}
	daFonte := []string{}
	if deLa.PowerChoice != "" {
		daFonte = append(daFonte, deLa.PowerChoice)
	}
	sort.Strings(escolhas)
	if strings.Join(escolhas, " | ") != strings.Join(daFonte, " | ") {
		t.Errorf("%s: a escolha de poder é [%s] no achatado e [%s] na fonte",
			slug, strings.Join(escolhas, ", "), strings.Join(daFonte, ", "))
	}
}

// A ORIGEM SEM PERÍCIA A OFERECER é o Amnésico, e ele é UM.
//
// O livro não dá lista de benefícios a ele: "em vez de dois benefícios de uma
// lista, você recebe uma perícia e um poder escolhidos pelo mestre" (p86). É a
// única origem em que `pericias` é vazia de propósito, e é por isso que ela é
// nomeada aqui.
//
// A lista é de PERMITIDOS e não de proibidos: o que reprova é uma origem
// DESCONHECIDA aparecer sem perícia, que é a cara de uma transcrição que perdeu a
// linha de benefícios — e, sem este caso, a perda seria silenciosa, porque uma
// lista vazia é um JSON perfeitamente válido. Uma lista de proibidos subcontaria
// sem nem ter denominador para denunciar a subcontagem.
func TestOnlyTheAmnesicOriginHasNoExpertiseToOffer(t *testing.T) {
	const semPericia = "amnesico"
	fonte, _ := lerAsDuasCopiasDeOrigem(t)
	vazias := 0
	for slug, origem := range fonte {
		if len(origem.Expertises) > 0 {
			continue
		}
		vazias++
		if slug != semPericia {
			t.Errorf("%s não oferece perícia nenhuma, e a única origem do livro assim é "+
				"%q. Confira a linha `Benefícios.` na página %d: ou a transcrição "+
				"perdeu as perícias, ou esta origem é uma segunda exceção e o nome "+
				"dela entra neste caso", slug, semPericia, origem.BookPage)
		}
	}
	if vazias != 1 {
		t.Errorf("%d origens sem perícia, e o esperado é exatamente 1 (%q). Se o "+
			"Amnésico ganhou perícias, tire-o deste caso", vazias, semPericia)
	}
	if !fonte[semPericia].GmDriven {
		t.Errorf("%q deixou de ser `gmDriven`, e é esse campo que diz à tela que os "+
			"benefícios dele são escolhidos pelo mestre", semPericia)
	}
}

type origemDaFonte struct {
	Name        string   `json:"name"`
	Expertises  []string `json:"pericias"`
	Powers      []string `json:"poderes"`
	UniquePower string   `json:"poderUnico"`
	PowerChoice string   `json:"poderChoiceCategory"`
	GmDriven    bool     `json:"gmDriven"`
	BookPage    int      `json:"bookPage"`
	Uid         string   `json:"uid"`
}

type origemAchatada struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Uid         string `json:"uid"`
	UniquePower struct {
		Name string `json:"name"`
	} `json:"poderUnico"`
	Benefits []struct {
		Name      string `json:"name"`
		PowerPick string `json:"powerPick"`
	} `json:"benefits"`
}

func lerAsDuasCopiasDeOrigem(t *testing.T) (map[string]origemDaFonte, map[string]origemAchatada) {
	t.Helper()
	fonte := map[string]origemDaFonte{}
	bruto, ok := catalog.Resource("origins-source")
	if !ok {
		t.Fatal("catálogo origins-source ausente")
	}
	if err := json.Unmarshal(bruto, &fonte); err != nil {
		t.Fatalf("origins-source: %v", err)
	}
	bruto, ok = catalog.Resource("origins")
	if !ok {
		t.Fatal("catálogo origins ausente")
	}
	var lista []origemAchatada
	if err := json.Unmarshal(bruto, &lista); err != nil {
		t.Fatalf("origins: %v", err)
	}
	achatado := make(map[string]origemAchatada, len(lista))
	for _, origem := range lista {
		achatado[origem.ID] = origem
	}
	return fonte, achatado
}

// O PODER ÚNICO DE CADA ORIGEM ACHA A ATIVAÇÃO DELE, e o elo é o NOME.
//
// O `activations.json` é o QUARTO arquivo a falar das origens, depois dos dois
// catálogos e do despejo de paridade, e é dele que saem a ação, o custo em PM e
// o limite de usos que o cartão da ficha mostra. O `book.ActivationOf` procura
// primeiro pelo id e depois pelo NOME — e o id do poder único (`origin-<slug>-
// unique`) nunca está lá, então quem resolve é sempre o nome.
//
// Isso faz de um renome de poder único um elo que se rompe em SILÊNCIO: a
// entrada continua no arquivo, a busca devolve `nil`, e o cartão perde a ação e
// o custo sem erro nenhum. A ALE-405 achou a entrada `origin.trabalhador.
// atletico` chamada "Atlético" — o nome do poder GERAL que a origem concede, e
// não o do poder exclusivo dela, que é "Esforçado" (p95).
func TestEveryOriginUniquePowerFindsItsActivation(t *testing.T) {
	_, achatado := lerAsDuasCopiasDeOrigem(t)
	bruto, ok := catalog.Resource("activations")
	if !ok {
		t.Fatal("catálogo de ativações ausente")
	}
	var ativacoes []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(bruto, &ativacoes); err != nil {
		t.Fatalf("activations: %v", err)
	}
	porNome := make(map[string]string, len(ativacoes))
	for _, ativacao := range ativacoes {
		porNome[ativacao.Name] = ativacao.ID
	}
	resolvidos := 0
	for slug, origem := range achatado {
		nome := origem.UniquePower.Name
		id, achou := porNome[nome]
		if !achou {
			t.Errorf("%s: o poder único %q não acha ativação nenhuma pelo nome, e é "+
				"pelo nome que o `book.ActivationOf` o resolve. O cartão da ficha "+
				"perde a ação e o custo em PM, calado — procure em activations.json a "+
				"entrada `origin.%s.*` e confira o `name` dela", slug, nome, slug)
			continue
		}
		resolvidos++
		// A entrada pode ter QUALQUER nome e ainda assim ser achada; o que a
		// prende a esta origem é o prefixo do id. Sem esta linha, o poder único
		// de uma origem poderia resolver para a ativação de outra.
		if !strings.HasPrefix(id, "origin."+slug+".") {
			t.Errorf("%s: o poder único %q resolve para a ativação %q, que é de outro "+
				"dono — a ficha mostraria a ação e o custo errados", slug, nome, id)
		}
	}
	if resolvidos != len(achatado) {
		t.Errorf("%d dos %d poderes únicos resolveram", resolvidos, len(achatado))
	}
}
