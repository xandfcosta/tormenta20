package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"t20engine/domain/book"
	"testing"
)

// A FORJA CURTA.
//
// Os testes batem no roteador de verdade com formulário de verdade, porque é
// isso que o navegador manda: a folha é um `<form method="post">` e o redesenho
// do Datastar manda o MESMO formulário (`contentType: 'form'`).

// postaAForja manda um formulário autenticado pelo roteador do app.
func postaAForja(t *testing.T, f sceneFixture, userID int64, path string, fields url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(fields.Encode()))
	req.Header.Set("Authorization", "Bearer "+f.token(t, userID))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	f.s.WebRouter().ServeHTTP(rec, req)
	return rec
}

// aFolhaPreenchida é uma resposta completa e válida — um guerreiro de couro.
func aFolhaPreenchida() url.Values {
	return url.Values{
		"name": {"Thessa de Valkaria"}, "race": {"Elfo"}, "class": {"Guerreiro"},
		"origin": {"Acólito"}, "weaponSimple": {"adaga"}, "weaponMartial": {"espada-longa"},
		"armor": {"couro-batido"}, "shield": {"1"},
	}
}

// Guarda de varredura: a folha não pode ter lista escrita à mão.
//
// O regime é AMOSTRAGEM e não enumeração — a folha desenha o que o catálogo
// tem, então uma raça nova aparece sozinha. O que este guarda pega é a
// regressão contrária: alguém trocar o catálogo por uma lista curta "só das
// principais", que é como as cartas costumam começar.
func TestEveryBookRaceAndClassHasACardInTheForge(t *testing.T) {
	f := newSceneFixture(t)
	body := f.pede(t, f.player, http.MethodGet, "/personagens/nova", "").Body.String()

	races, classes, _ := book.CharacterCatalogs()
	if len(races) < 17 || len(classes) != 14 {
		t.Fatalf("o catálogo chegou com %d raças e %d classes — o livro tem 17 e 14",
			len(races), len(classes))
	}
	for _, race := range races {
		if !strings.Contains(body, `value="`+race.Name+`"`) {
			t.Errorf("a raça %q não tem carta na folha", race.Name)
		}
	}
	for _, class := range classes {
		if !strings.Contains(body, `value="`+class.Name+`"`) {
			t.Errorf("a classe %q não tem carta na folha", class.Name)
		}
	}
}

// O kit de p140 se conhece pela classe, e antes dela a seção não existe.
func TestTheFormOnlyOffersEquipmentAfterTheClass(t *testing.T) {
	f := newSceneFixture(t)
	empty := f.pede(t, f.player, http.MethodGet, "/personagens/nova", "").Body.String()
	if strings.Contains(empty, "Equipamento inicial") {
		t.Error("a folha vazia já oferece equipamento, sem saber a classe")
	}

	withClass := postaAForja(t, f, f.player, "/personagens/nova/esboco",
		url.Values{"class": {"Guerreiro"}}).Body.String()
	if !strings.Contains(withClass, "Equipamento inicial") {
		t.Fatal("o esboço com classe não trouxe o equipamento")
	}
}

// p140, o mesmo par de casos do teste de regra do motor, agora atravessando a
// cena.
func TestTheOfferedEquipmentFollowsTheClass(t *testing.T) {
	f := newSceneFixture(t)
	cases := []struct {
		class   string
		present []string
		absent  []string
	}{
		{
			"Guerreiro",
			[]string{"Arma marcial", "Brunea", "Escudo leve", "Gibão de peles"},
			nil,
		},
		{
			// O arcanista é a exceção escrita do livro, e ele não é proficiente
			// em nada: sem marcial, sem armadura, sem escudo.
			"Arcanista",
			[]string{"Arcanistas começam sem armadura"},
			[]string{"Arma marcial", "Brunea", "Escudo leve", "Gibão de peles"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.class, func(t *testing.T) {
			body := postaAForja(t, f, f.player, "/personagens/nova/esboco",
				url.Values{"class": {tc.class}}).Body.String()
			for _, text := range tc.present {
				if !strings.Contains(body, text) {
					t.Errorf("faltou %q na folha de %s", text, tc.class)
				}
			}
			for _, text := range tc.absent {
				if strings.Contains(body, text) {
					t.Errorf("%q apareceu na folha de %s", text, tc.class)
				}
			}
		})
	}
}

// A tela esconde, o servidor RECUSA: a fronteira de segurança é o handler.
func TestTheForgeRefusesWhatTheKitDoesNotOffer(t *testing.T) {
	f := newSceneFixture(t)
	cases := []struct {
		name    string
		changes func(url.Values)
		refusal string
	}{
		{"raça que não existe", func(v url.Values) { v.Set("race", "Hobbit") }, "não é uma raça do livro"},
		{"origem que não existe", func(v url.Values) { v.Set("origin", "Pirata Espacial") }, "não é uma origem do livro"},
		{"nome vazio", func(v url.Values) { v.Set("name", "  ") }, "O nome é obrigatório"},
		{
			"brunea numa classe que não usa pesadas",
			func(v url.Values) { v.Set("class", "Ladino"); v.Set("weaponMartial", ""); v.Set("armor", "brunea") },
			"Escolha uma das armaduras que o kit oferece",
		},
		{
			"armadura no arcanista",
			func(v url.Values) {
				v.Set("class", "Arcanista")
				v.Set("weaponMartial", "")
				v.Set("shield", "")
				v.Set("armor", "couro-batido")
			},
			"Arcanistas começam sem armadura",
		},
		{
			"arma marcial numa classe sem marciais",
			func(v url.Values) { v.Set("class", "Ladino"); v.Set("armor", "couro-batido") },
			"Esta classe não começa com arma marcial",
		},
		{
			"armadura no lugar da arma simples",
			func(v url.Values) { v.Set("weaponSimple", "brunea") },
			"não é uma arma da categoria que o kit oferece",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fields := aFolhaPreenchida()
			tc.changes(fields)
			before := quantosHerois(t, f)

			rec := postaAForja(t, f, f.player, "/personagens/nova", fields)
			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status %d, esperado 422", rec.Code)
			}
			if !strings.Contains(rec.Body.String(), tc.refusal) {
				t.Errorf("a folha recusada não diz %q", tc.refusal)
			}
			if after := quantosHerois(t, f); after != before {
				t.Errorf("a recusa criou herói mesmo assim: %d → %d", before, after)
			}
		})
	}
}

// A folha volta PREENCHIDA: redigitar o que estava certo é castigo.
func TestTheRefusalGivesBackWhatWasAnswered(t *testing.T) {
	f := newSceneFixture(t)
	fields := aFolhaPreenchida()
	fields.Set("origin", "Pirata Espacial")

	body := postaAForja(t, f, f.player, "/personagens/nova", fields).Body.String()
	if !strings.Contains(body, `value="Thessa de Valkaria"`) {
		t.Error("o nome respondido não voltou no campo")
	}
	if !strings.Contains(body, `value="Elfo" checked`) {
		t.Error("a raça respondida não voltou marcada")
	}
	if !strings.Contains(body, `value="couro-batido" selected`) {
		t.Error("a armadura respondida não voltou escolhida")
	}
}

// O NASCIMENTO inteiro (p140).
func TestTheHeroIsBornDressedAndWithAPurse(t *testing.T) {
	f := newSceneFixture(t)
	rec := postaAForja(t, f, f.player, "/personagens/nova", aFolhaPreenchida())
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d, esperado 303: %s", rec.Code, rec.Body.String())
	}
	destination := rec.Header().Get("Location")
	if !strings.HasSuffix(destination, "/atributos") {
		t.Fatalf("o nascimento levou para %q, e não para os atributos", destination)
	}

	id := oIDDoDestino(t, destination)
	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("o herói não foi gravado: %v", err)
	}
	if row.Name != "Thessa de Valkaria" || row.Origin != "Acólito" || row.Level != 1 {
		t.Errorf("herói gravado: %q, origem %q, nível %d", row.Name, row.Origin, row.Level)
	}
	// O tamanho e o deslocamento saem da RAÇA e não de uma pergunta da folha.
	// O elfo é Médio e anda 12m — a Graça de Glórienn diz "seu deslocamento é
	// 12m (em vez de 9m)" (p22).
	if row.Size != "Médio" || row.Displacement != 12 {
		t.Errorf("tamanho %q e deslocamento %d, esperado Médio e 12m", row.Size, row.Displacement)
	}
	// T$ 4d6 vai de 4 a 24; o Acólito não concede dinheiro.
	if row.Tibar < 4 || row.Tibar > 24 {
		t.Errorf("bolsa de T$ %v fora de 4d6", row.Tibar)
	}
	// Os atributos nascem em zero: distribuí-los é a segunda cena.
	if row.Strength != 0 || row.Charisma != 0 {
		t.Errorf("o herói nasceu com atributo distribuído: For %d, Car %d", row.Strength, row.Charisma)
	}
	// PV cheio. O guerreiro de 1º nível tem 20 (p34) e o elfo leva Constituição
	// −1 (p22), então o poço é 19 — o modificador da RAÇA já entra no
	// nascimento, mesmo com os atributos base ainda em zero.
	if pool := poolsOf(t, f.s, id); pool.HpMax != 19 || pool.HpCurrent != pool.HpMax {
		t.Errorf("PV %d/%d, esperado 19/19", pool.HpCurrent, pool.HpMax)
	}

	items, err := f.s.queries.ListItemsByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("itens: %v", err)
	}
	inBag := map[string]bool{}
	for _, item := range items {
		inBag[item.Name] = true
		// NADA nasce equipado: empunhar tem regra de mãos (p141) e quem empunha
		// é o jogador, na Mochila.
		if item.Equipped.Valid && item.Equipped.String != "" {
			t.Errorf("%q nasceu equipado como %q", item.Name, item.Equipped.String)
		}
	}
	for _, want := range []string{
		"Mochila", "Saco de Dormir", "Traje de viajante", // o kit de todo mundo
		"Adaga", "Espada longa", "Couro batido", "Escudo leve", // as escolhas
		"Símbolo sagrado", "Traje de sacerdote", // os itens do Acólito (p85)
	} {
		if !inBag[want] {
			t.Errorf("o herói nasceu sem %q", want)
		}
	}
}

// As perícias FIXAS da classe e as proficiências dela. O que se ESCOLHE não
// nasce escolhido — vira pendência.
func TestTheHeroIsBornWithWhatTheClassTrainsAndUses(t *testing.T) {
	f := newSceneFixture(t)
	rec := postaAForja(t, f, f.player, "/personagens/nova", aFolhaPreenchida())
	id := oIDDoDestino(t, rec.Header().Get("Location"))

	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("herói: %v", err)
	}
	for _, category := range []string{"armas-simples", "armas-marciais", "armaduras-pesadas", "escudos"} {
		if !strings.Contains(row.Proficiencies, category) {
			t.Errorf("o guerreiro nasceu sem %q: %s", category, row.Proficiencies)
		}
	}

	expertises, err := f.s.queries.ListExpertisesByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("perícias: %v", err)
	}
	trained := map[string]bool{}
	for _, p := range expertises {
		if p.Trained != 0 {
			trained[p.Name] = true
		}
	}
	if !trained["Fortitude"] {
		t.Error("Fortitude é a perícia FIXA do guerreiro e não nasceu treinada")
	}
	// "Luta ou Pontaria" é escolha do bloco da classe: ela é pendência da ficha.
	if trained["Luta"] || trained["Pontaria"] {
		t.Error("a escolha entre Luta e Pontaria foi feita pela forja")
	}
}

// p17, Tabela 1-1.
func TestTheForgePointBuyRefusesWhatTheBookForbids(t *testing.T) {
	f := newSceneFixture(t)
	rec := postaAForja(t, f, f.player, "/personagens/nova", aFolhaPreenchida())
	id := oIDDoDestino(t, rec.Header().Get("Location"))
	attributes := "/personagens/" + strconv.FormatInt(id, 10) + "/atributos"

	// Quatro em Força custa 7 e cabe.
	for i := 0; i < 4; i++ {
		if code := postaAForja(t, f, f.player, attributes+"/strength/1", nil).Code; code != http.StatusOK {
			t.Fatalf("subir Força %d: status %d", i+1, code)
		}
	}
	// O quinto passa do máximo de +4, e a recusa é CONTEÚDO em 200.
	refused := postaAForja(t, f, f.player, attributes+"/strength/1", nil)
	if refused.Code != http.StatusOK {
		t.Fatalf("a recusa veio em %d — o Datastar descarta remendo que não é 2xx", refused.Code)
	}
	if sentence := sceneRefusal(refused.Body.String()); !strings.Contains(sentence, "compra de pontos") {
		t.Errorf("a cena não disse por que recusou: %q", sentence)
	}

	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("herói: %v", err)
	}
	if row.Strength != 4 {
		t.Errorf("Força ficou em %d — a recusa gravou mesmo assim", row.Strength)
	}
	// E a Constituição mexe no PV: subir um ponto sobe o poço, e ele fica cheio.
	if code := postaAForja(t, f, f.player, attributes+"/constitution/1", nil).Code; code != http.StatusOK {
		t.Fatalf("subir Constituição: status %d", code)
	}
	// Con base +1 com a Constituição −1 do elfo dá Con 0: o poço volta a 20.
	if pool := poolsOf(t, f.s, id); pool.HpMax != 20 || pool.HpCurrent != 20 {
		t.Errorf("PV %d/%d, esperado 20/20 com Con base +1 num elfo", pool.HpCurrent, pool.HpMax)
	}
}

// A posse é conferida como em toda rota de personagem.
func TestTheForgeAttributesBelongToTheOwner(t *testing.T) {
	f := newSceneFixture(t)
	rec := postaAForja(t, f, f.player, "/personagens/nova", aFolhaPreenchida())
	id := oIDDoDestino(t, rec.Header().Get("Location"))
	path := "/personagens/" + strconv.FormatInt(id, 10) + "/atributos"

	if code := postaAForja(t, f, f.gm, path+"/strength/1", nil).Code; code != http.StatusForbidden {
		t.Errorf("o mestre distribuiu atributo de herói alheio: status %d", code)
	}
	if code := f.pede(t, f.gm, http.MethodGet, path, "").Code; code != http.StatusForbidden {
		t.Errorf("o mestre abriu os atributos de herói alheio: status %d", code)
	}
}

// oIDDoDestino tira o id de "/personagens/7/atributos".
//
// Ele confere o SEGMENTO em vez de contar posição: contar quebra quando a rota
// muda de profundidade, e o modo de falhar é ruim — `ParseInt("atributos")` não
// diz que o endereço mudou, diz que um número está mal escrito.
func oIDDoDestino(t *testing.T, destination string) int64 {
	t.Helper()
	parts := strings.Split(strings.Trim(destination, "/"), "/")
	if len(parts) < 2 || parts[0] != "personagens" {
		t.Fatalf("destino inesperado: %q — esperava /personagens/{id}/…", destination)
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		t.Fatalf("id no destino %q: %v", destination, err)
	}
	return id
}

func quantosHerois(t *testing.T, f sceneFixture) int {
	t.Helper()
	list, err := f.s.queries.ListCharactersByOwner(context.Background(), f.player)
	if err != nil {
		t.Fatalf("listar heróis: %v", err)
	}
	return len(list)
}

// "Não escolheu" e "escolheu o que não existe" chegam no mesmo campo e não são a
// mesma coisa.
func TestTheBlankFormAsksForTheChoiceInsteadOfBlamingAnEmptyValue(t *testing.T) {
	f := newSceneFixture(t)
	fields := url.Values{"name": {"Sem escolhas"}}

	body := postaAForja(t, f, f.player, "/personagens/nova", fields).Body.String()
	for _, sentence := range []string{
		"Escolha a linhagem do herói.", "Escolha o ofício do herói.", "Escolha a origem do herói.",
	} {
		if !strings.Contains(body, sentence) {
			t.Errorf("a folha vazia não pede %q", sentence)
		}
	}
	if strings.Contains(body, `"" não é`) {
		t.Error(`a folha acusou o vazio como valor desconhecido`)
	}
}
