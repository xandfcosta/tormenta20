package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"t20engine/db/sqlcgen"
	"t20engine/plataforma"
	"t20engine/sheet"
	"t20engine/web/sheetui"
	"testing"

	"github.com/go-chi/chi/v5"
)

func guerreiro(t *testing.T) (pilotoFixture, int64) {
	t.Helper()
	f := novoPiloto(t)
	id := seedCharacterAtLevel(t, f.s, f.jogador, "Guerreiro", 3, 20, 20, 10, 10)
	seedClasse(t, f.s, id, "Guerreiro", 3)
	return f, id
}

// saved lê o blob da coluna, que é a única fonte da verdade do painel.
func saved(t *testing.T, f pilotoFixture, id int64) map[string]bool {
	t.Helper()
	row, err := f.s.Queries().GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler o personagem: %v", err)
	}
	return sheet.ToStringSet(sheet.UnmarshalStrings(row.Proficiencies))
}

// saveHand põe um estado que só um ajuste manual produziria.
func saveHand(t *testing.T, f pilotoFixture, id int64, blob string) {
	t.Helper()
	err := f.s.Queries().SetProficiencies(context.Background(), sqlcgen.SetProficienciesParams{
		Proficiencies: blob, UpdatedAt: plataforma.NowISO(), ID: id,
	})
	if err != nil {
		t.Fatalf("gravar %q: %v", blob, err)
	}
}

// A TABELA DO LIVRO CHEGA NA TELA, e a etiqueta diz DE ONDE vem.
//
// O guerreiro do livro (p64) sabe usar armas marciais, armaduras pesadas e
// escudos — e não sabe usar armas exóticas, que nenhuma classe concede. A tela
// tem de dizer as duas coisas: o que a classe deu, com o nome da classe, e o que
// ela não deu.
func TestTheProficienciesPanelSaysWhatTheClassGrants(t *testing.T) {
	f, id := guerreiro(t)

	tela := f.pede(t, f.jogador, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=proficiencies", id), "").Body.String()

	if !strings.Contains(tela, "Padrão: Guerreiro") {
		t.Error("a etiqueta não diz de qual classe vem a proficiência")
	}
	// A ETIQUETA "classe" é o que separa um ajuste deliberado do que veio de
	// fábrica; sem ela, uma proficiência tirada na mão parece defeito da conta.
	if strings.Count(tela, ">classe</span>") == 0 {
		t.Error("nenhuma linha está marcada como padrão de classe")
	}
	for _, rotulo := range []string{"Armas simples", "Armas marciais", "Armas exóticas", "Escudos"} {
		if !strings.Contains(tela, rotulo) {
			t.Errorf("a categoria %q não está no painel", rotulo)
		}
	}
}

// ARMADURA PESADA CONCEDE A LEVE — e isso o LIVRO NÃO DIZ.
//
// Conferi a p148: ela define as duas categorias e a penalidade por não
// proficiência, e não há linha dizendo que uma implica a outra. É decisão de
// produto, herdada da SPA, e está presa aqui porque é a que NÃO machuca: sem
// ela, "restaurar o padrão de classe" tiraria a armadura leve de um guerreiro, e
// o motor passaria a aplicar a penalidade da p148 num personagem treinado em
// algo mais pesado.
//
// Este caso existe para a decisão ser REVISTA de propósito e não redescoberta
// como defeito: se alguém decidir seguir o livro à risca, é aqui que ele quebra.
func TestHeavyArmorGrantsTheLightOne(t *testing.T) {
	f, id := guerreiro(t)

	rec := f.pede(t, f.jogador, http.MethodPost,
		fmt.Sprintf("/personagens/%d/proficiencias/padrao", id), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("restaurar deu %d: %s", rec.Code, rec.Body.String())
	}

	tem := saved(t, f, id)
	if !tem["armaduras-leves"] {
		t.Error("o guerreiro ficou sem armadura leve: o motor vai penalizá-lo por vestir couro")
	}
	if !tem["armaduras-pesadas"] {
		t.Error("o guerreiro ficou sem armadura pesada, que a p64 concede")
	}
}

// RESTAURAR DESCARTA O AJUSTE MANUAL — e é o predicado que importa.
//
// O caso prende as DUAS direções, e é isso que o faz discriminar: uma
// proficiência acrescentada na mão SAI, e uma concedida pela classe que foi
// tirada na mão VOLTA. Um caso só numa direção passaria por igual se o restaurar
// fizesse união em vez de substituição.
func TestRestoringTheDefaultDiscardsTheManualAdjustment(t *testing.T) {
	f, id := guerreiro(t)
	// Exóticas nenhuma classe concede; marciais o guerreiro concede e aqui está
	// AUSENTE — os dois lados do erro, num blob só.
	saveHand(t, f, id, `["armas-exoticas"]`)

	rec := f.pede(t, f.jogador, http.MethodPost,
		fmt.Sprintf("/personagens/%d/proficiencias/padrao", id), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("restaurar deu %d: %s", rec.Code, rec.Body.String())
	}

	tem := saved(t, f, id)
	if tem["armas-exoticas"] {
		t.Error("o acréscimo manual sobreviveu ao restaurar: ele fez união, não substituição")
	}
	if !tem["armas-marciais"] {
		t.Error("a proficiência da classe não voltou: o restaurar não restaurou nada")
	}
}

// ALTERNAR INVERTE O QUE ESTÁ GUARDADO, nos dois sentidos.
//
// O comando manda a CATEGORIA e não o estado desejado, e é o servidor que decide
// o novo valor: mandar "ligada" perderia para o clique repetido e para a mesma
// ficha aberta em dois aparelhos.
func TestTogglingTurnsTheProficiencyOnAndOff(t *testing.T) {
	f, id := guerreiro(t)
	saveHand(t, f, id, `["armas-marciais"]`)
	rota := fmt.Sprintf("/personagens/%d/proficiencias/alterna/armas-marciais", id)

	if rec := f.pede(t, f.jogador, http.MethodPost, rota, ""); rec.Code != http.StatusOK {
		t.Fatalf("alternar deu %d: %s", rec.Code, rec.Body.String())
	}
	if saved(t, f, id)["armas-marciais"] {
		t.Fatal("o primeiro toque não DESLIGOU a proficiência")
	}

	if rec := f.pede(t, f.jogador, http.MethodPost, rota, ""); rec.Code != http.StatusOK {
		t.Fatalf("alternar de volta deu %d: %s", rec.Code, rec.Body.String())
	}
	if !saved(t, f, id)["armas-marciais"] {
		t.Error("o segundo toque não LIGOU de volta: o comando não alterna, ele fixa")
	}
}

// CATEGORIA INVENTADA É RECUSADA, e o banco não se mexe.
//
// A rota tem a categoria no CAMINHO, então qualquer coisa chega até o handler.
// Recusar sem gravar é o que impede um endereço digitado de pôr lixo na coluna —
// e o lixo seria silencioso, porque o leitor do blob ignora o que não conhece.
func TestAProficiencyOutsideTheCatalogIsNotSaved(t *testing.T) {
	f, id := guerreiro(t)
	saveHand(t, f, id, `["armas-marciais"]`)

	rec := f.pede(t, f.jogador, http.MethodPost,
		fmt.Sprintf("/personagens/%d/proficiencias/alterna/armas-de-laser", id), "")

	// A MENSAGEM CARREGA O VALOR OFENSOR E O FORMATO ESPERADO: "proficiência
	// inválida" não ajudaria ninguém a descobrir o que digitar.
	recusa := sceneRefusal(rec.Body.String())
	if recusa == "" {
		t.Fatal("uma categoria inventada foi aceita sem uma palavra na tela")
	}
	if !strings.Contains(recusa, "armas-de-laser") || !strings.Contains(recusa, "armas-simples") {
		t.Errorf("a recusa não diz o valor recusado e as opções: %q", recusa)
	}
	if !saved(t, f, id)["armas-marciais"] {
		t.Error("a recusa mexeu no que já estava guardado")
	}
}

// VARREDURA: nenhuma escrita da ficha aceita quem não é dono.
//
// A trava mora no `sheetCommand`, e a fatia 1 já a prende uma vez. O que ESTE
// guarda prende é outra coisa, e é a que a lista de rotas vai quebrar: uma rota
// nova registrada FORA do gateway. Ela funcionaria, passaria nos testes do painel
// dela, e deixaria a ficha de qualquer pessoa aberta para qualquer conta.
//
// Ele varre as rotas de verdade, do roteador de verdade, e **falha se encontrar
// um parâmetro que não sabe preencher** — é isso que força a varredura: a fatia
// que acrescentar `{item}` ou `{magia}` tem de vir aqui dizer com que valor se
// preenche, e nesse mesmo instante a rota nova passa a ser conferida.
func TestNoSheetWriteAcceptsAStranger(t *testing.T) {
	f, id := guerreiro(t)
	// Valores plausíveis por parâmetro. O 403 tem de vir ANTES de qualquer
	// validação de conteúdo, então o valor só precisa existir — mas um valor
	// impossível esconderia uma rota que valida primeiro e barra depois.
	valores := map[string]string{
		"id":        fmt.Sprintf("%d", id),
		"qual":      "pv",
		"passo":     "1",
		"classe":    "Guerreiro",
		"categoria": "armas-marciais",
		// A PERÍCIA vai ESCAPADA, como o comando a escreve: "Atuação" no caminho
		// é `Atua%C3%A7%C3%A3o`, e um valor cru aqui mediria uma rota que o
		// cliente nunca chama.
		"nome":     url.PathEscape("Atuação"),
		"atributo": "charisma",
		// Os quatro da aba Efeitos (fatia 5). Todos PLAUSÍVEIS: o 403 tem de vir
		// antes de qualquer validação de conteúdo, e um valor impossível
		// esconderia uma rota que valida primeiro e barra depois.
		"cond":   "caido",
		"magia":  "armadura-arcana",
		"efeito": "1",
		"flag":   "furia",
		// Os três da Mochila (fatia 7).
		"item":     "1",
		"slot":     "vested",
		"catalogo": "adaga",
		// Os dos Poderes (fatia 8) — a `flag` já entrou com os Efeitos.
		"poder":       "class.barbaro.brado-assustador",
		"beneficio":   "origin-batedor-pericia-Furtividade",
		"variante":    "suraggel-aggelus",
		"escolha":     "caminho",
		"valor":       "bruxo",
		"ascendencia": "aggelus",
	}

	var visitadas int
	andar := func(metodo string, rota string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if metodo != http.MethodPost || !strings.HasPrefix(rota, "/personagens/{id}/") {
			return nil
		}
		caminho := rota
		for _, pedaco := range strings.Split(rota, "/") {
			if !strings.HasPrefix(pedaco, "{") {
				continue
			}
			nome := strings.Trim(pedaco, "{}")
			valor, sabe := valores[nome]
			if !sabe {
				t.Errorf("a rota %s tem o parâmetro {%s} e este guarda não sabe preenchê-lo — "+
					"acrescente um valor plausível ao mapa `valores`", rota, nome)
				return nil
			}
			caminho = strings.Replace(caminho, pedaco, valor, 1)
		}
		visitadas++
		rec := f.pede(t, f.mestre, http.MethodPost, caminho, "")
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s aceitou quem não é dono: %d — a rota não passa pelo `comandoDaFicha`",
				rota, rec.Code)
		}
		return nil
	}
	roteador, ok := f.s.WebRouter().(chi.Routes)
	if !ok {
		t.Fatal("o roteador do piloto deixou de ser um chi.Mux: esta varredura não alcança mais as rotas")
	}
	if err := chi.Walk(roteador, andar); err != nil {
		t.Fatalf("varrer as rotas: %v", err)
	}
	// CONTROLE: sem ele, um filtro que não casa com rota nenhuma passaria verde
	// afirmando que zero rotas estão seguras. É a lição da ALE-238 — provar que o
	// canal existe antes de ler o silêncio como resultado.
	if visitadas < 4 {
		t.Fatalf("a varredura achou %d rotas de escrita da ficha, e existem pelo menos 4: "+
			"o filtro parou de casar com o roteador", visitadas)
	}
}

func TestEverySheetTabDrawsSomething(t *testing.T) {
	f, id := guerreiro(t)

	var visitadas int
	for _, aba := range sheetui.Tabs() {
		titulo, sabe := panelTitle[aba.Valor]
		if !sabe {
			t.Errorf("a aba %q não tem título esperado neste guarda — acrescente a "+
				"linha ao mapa `oTituloDoPainel`", aba.Valor)
			continue
		}
		visitadas++
		tela := f.pede(t, f.jogador, http.MethodGet,
			fmt.Sprintf("/personagens/%d?tab=%s", id, aba.Valor), "").Body.String()
		if !strings.Contains(tela, ">"+titulo+"</h2>") {
			t.Errorf("a aba %q não desenhou painel nenhum", aba.Valor)
		}
	}
	// CONTROLE: sem ele, um `Tabs` que virasse vazio faria o laço não
	// rodar nenhuma vez e o guarda passar afirmando nada.
	if visitadas != 7 {
		t.Fatalf("%d abas visitadas, e a ficha tem sete: o guarda mediu outra coisa", visitadas)
	}
}

// VARREDURA: nenhum comando da ficha perde a aba aberta.
//
// # O defeito que este guarda existe para não voltar
//
// Todo `@post` da ficha responde redesenhando a CENA INTEIRA, e o handler
// descobre em que seção redesenhar lendo `?tab=` da própria requisição. Um
// comando escrito sem o `?tab=` faz o `AskedTab` cair na primeira aba: o
// jogador mexe no PV com a Mochila aberta e a ficha pula para Perícias — parece
// que ela se fechou sozinha.
//
// Foi ENTREGUE na fatia 1 e ninguém viu, porque com todas as abas mostrando o
// mesmo aviso de "vive na ficha antiga" o salto não tinha aparência. O primeiro
// painel portado o denunciou no primeiro clique da bancada.
//
// Ele varre as SETE abas e falha nomeando o comando que saiu sem o `?tab=`. É
// mecanizável com o que já roda, então virou guarda em vez de parágrafo — e cada
// fatia nova ganha a cobertura de graça, porque o que ele lê é o HTML.
func TestNoSheetCommandLosesTheTab(t *testing.T) {
	f, id := guerreiro(t)
	// Multiclasse: é o que faz o diálogo do degrau existir, e os comandos DELE
	// são os mais fáceis de esquecer — moram noutro templ.
	seedClasse(t, f.s, id, "Arcanista", 2)

	// O APÓSTROFO SAI ESCAPADO: valor de atributo DINÂMICO passa pelo escape de
	// HTML do templ (`&#39;`), enquanto uma constante sai literal — a armadilha
	// está no `engine-go/CLAUDE.md`. A primeira versão desta expressão procurava
	// a aspa crua e achou ZERO comandos; foi o CONTROLE abaixo que disse isso, em
	// vez de o teste passar verde afirmando que nada estava errado.
	postados := regexp.MustCompile(`@post\((?:&#39;|')([^'&]+)(?:&#39;|')\)`)
	var vistos int
	for _, aba := range sheetui.Tabs() {
		tela := f.pede(t, f.jogador, http.MethodGet,
			fmt.Sprintf("/personagens/%d?tab=%s", id, aba.Valor), "").Body.String()
		for _, achado := range postados.FindAllStringSubmatch(tela, -1) {
			vistos++
			if !strings.HasSuffix(achado[1], "?tab="+aba.Valor) {
				t.Errorf("na aba %q o comando %q não carrega a aba: o clique joga o jogador para %q",
					aba.Valor, achado[1], sheetui.Tabs()[0].Valor)
			}
		}
	}
	// CONTROLE: sem ele, um `@post` que deixasse de ser escrito assim — ou uma
	// expressão regular que parasse de casar — daria verde afirmando que zero
	// comandos estão certos. São 12 por aba no mínimo (8 passos de vital, 2 do
	// degrau, 2 do diálogo do multiclasse).
	if vistos < 12*len(sheetui.Tabs()) {
		t.Fatalf("a varredura achou %d comandos nas sete abas, e são pelo menos %d: "+
			"a expressão parou de casar com o HTML", vistos, 12*len(sheetui.Tabs()))
	}
}
