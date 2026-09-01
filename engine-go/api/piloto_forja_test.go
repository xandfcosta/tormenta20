package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"t20engine/book"
	"testing"
)

// A FORJA CURTA (ALE-272, fatia 9).
//
// Os testes batem no roteador de verdade com formulário de verdade, porque é
// isso que o navegador manda: a folha é um `<form method="post">` e o redesenho
// do Datastar manda o MESMO formulário (`contentType: 'form'`).

// postaAForja manda um formulário autenticado pelo roteador do piloto.
func postaAForja(t *testing.T, f pilotoFixture, userID int64, caminho string, campos url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, caminho, strings.NewReader(campos.Encode()))
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

// TestTodaRacaEClasseDoLivroTemCartaNaForja é guarda de varredura: a folha não
// pode ter lista escrita à mão.
//
// O regime é AMOSTRAGEM e não enumeração — a folha desenha o que o catálogo
// tem, então uma raça nova aparece sozinha. O que este guarda pega é a
// regressão contrária: alguém trocar o catálogo por uma lista curta "só das
// principais", que é como as cartas costumam começar.
func TestTodaRacaEClasseDoLivroTemCartaNaForja(t *testing.T) {
	f := novoPiloto(t)
	corpo := f.pede(t, f.jogador, http.MethodGet, "/personagens/nova", "").Body.String()

	racas, classes, _ := book.CharacterCatalogs()
	if len(racas) < 17 || len(classes) != 14 {
		t.Fatalf("o catálogo chegou com %d raças e %d classes — o livro tem 17 e 14",
			len(racas), len(classes))
	}
	for _, raca := range racas {
		if !strings.Contains(corpo, `value="`+raca.Name+`"`) {
			t.Errorf("a raça %q não tem carta na folha", raca.Name)
		}
	}
	for _, classe := range classes {
		if !strings.Contains(corpo, `value="`+classe.Name+`"`) {
			t.Errorf("a classe %q não tem carta na folha", classe.Name)
		}
	}
}

// TestAFolhaSoOfereceEquipamentoDepoisDaClasse: o kit de p140 se conhece pela
// classe, e antes dela a seção não existe.
func TestAFolhaSoOfereceEquipamentoDepoisDaClasse(t *testing.T) {
	f := novoPiloto(t)
	vazia := f.pede(t, f.jogador, http.MethodGet, "/personagens/nova", "").Body.String()
	if strings.Contains(vazia, "Equipamento inicial") {
		t.Error("a folha vazia já oferece equipamento, sem saber a classe")
	}

	comClasse := postaAForja(t, f, f.jogador, "/personagens/nova/esboco",
		url.Values{"class": {"Guerreiro"}}).Body.String()
	if !strings.Contains(comClasse, "Equipamento inicial") {
		t.Fatal("o esboço com classe não trouxe o equipamento")
	}
}

// TestOEquipamentoOferecidoSegueAClasse — p140, e é o mesmo par de casos do
// teste de regra do motor, agora atravessando a cena.
func TestOEquipamentoOferecidoSegueAClasse(t *testing.T) {
	f := novoPiloto(t)
	casos := []struct {
		classe   string
		presente []string
		ausente  []string
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
	for _, caso := range casos {
		t.Run(caso.classe, func(t *testing.T) {
			corpo := postaAForja(t, f, f.jogador, "/personagens/nova/esboco",
				url.Values{"class": {caso.classe}}).Body.String()
			for _, texto := range caso.presente {
				if !strings.Contains(corpo, texto) {
					t.Errorf("faltou %q na folha de %s", texto, caso.classe)
				}
			}
			for _, texto := range caso.ausente {
				if strings.Contains(corpo, texto) {
					t.Errorf("%q apareceu na folha de %s", texto, caso.classe)
				}
			}
		})
	}
}

// TestAForjaRecusaOQueOKitNaoOferece: a tela esconde, o servidor RECUSA. É a
// fronteira que a rota JSON de criar personagem deixou aberta por escrito.
func TestAForjaRecusaOQueOKitNaoOferece(t *testing.T) {
	f := novoPiloto(t)
	casos := []struct {
		nome   string
		muda   func(url.Values)
		recusa string
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
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			campos := aFolhaPreenchida()
			caso.muda(campos)
			antes := quantosHerois(t, f)

			rec := postaAForja(t, f, f.jogador, "/personagens/nova", campos)
			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status %d, esperado 422", rec.Code)
			}
			if !strings.Contains(rec.Body.String(), caso.recusa) {
				t.Errorf("a folha recusada não diz %q", caso.recusa)
			}
			if depois := quantosHerois(t, f); depois != antes {
				t.Errorf("a recusa criou herói mesmo assim: %d → %d", antes, depois)
			}
		})
	}
}

// TestARecusaDevolveOQueFoiRespondido: a folha volta preenchida. Redigitar o
// que estava certo é o castigo que a campanha nova já evitava (ALE-246).
func TestARecusaDevolveOQueFoiRespondido(t *testing.T) {
	f := novoPiloto(t)
	campos := aFolhaPreenchida()
	campos.Set("origin", "Pirata Espacial")

	corpo := postaAForja(t, f, f.jogador, "/personagens/nova", campos).Body.String()
	if !strings.Contains(corpo, `value="Thessa de Valkaria"`) {
		t.Error("o nome respondido não voltou no campo")
	}
	if !strings.Contains(corpo, `value="Elfo" checked`) {
		t.Error("a raça respondida não voltou marcada")
	}
	if !strings.Contains(corpo, `value="couro-batido" selected`) {
		t.Error("a armadura respondida não voltou escolhida")
	}
}

// TestOHeroiNasceVestidoEComBolsa é o teste do NASCIMENTO inteiro (p140).
func TestOHeroiNasceVestidoEComBolsa(t *testing.T) {
	f := novoPiloto(t)
	rec := postaAForja(t, f, f.jogador, "/personagens/nova", aFolhaPreenchida())
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d, esperado 303: %s", rec.Code, rec.Body.String())
	}
	destino := rec.Header().Get("Location")
	if !strings.HasSuffix(destino, "/atributos") {
		t.Fatalf("o nascimento levou para %q, e não para os atributos", destino)
	}

	id := oIDDoDestino(t, destino)
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
	if row.Hpmax != 19 || row.Hpcurrent != row.Hpmax {
		t.Errorf("PV %d/%d, esperado 19/19", row.Hpcurrent, row.Hpmax)
	}

	itens, err := f.s.queries.ListItemsByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("itens: %v", err)
	}
	naMochila := map[string]bool{}
	for _, item := range itens {
		naMochila[item.Name] = true
		// NADA nasce equipado: empunhar tem regra de mãos (p141) e quem empunha
		// é o jogador, na Mochila.
		if item.Equipped.Valid && item.Equipped.String != "" {
			t.Errorf("%q nasceu equipado como %q", item.Name, item.Equipped.String)
		}
	}
	for _, esperado := range []string{
		"Mochila", "Saco de Dormir", "Traje de viajante", // o kit de todo mundo
		"Adaga", "Espada longa", "Couro batido", "Escudo leve", // as escolhas
		"Símbolo sagrado", "Traje de sacerdote", // os itens do Acólito (p85)
	} {
		if !naMochila[esperado] {
			t.Errorf("o herói nasceu sem %q", esperado)
		}
	}
}

// TestOHeroiNasceComOQueAClasseTreinaEUsa: as perícias FIXAS da classe e as
// proficiências dela. O que se ESCOLHE não nasce escolhido — vira pendência.
func TestOHeroiNasceComOQueAClasseTreinaEUsa(t *testing.T) {
	f := novoPiloto(t)
	rec := postaAForja(t, f, f.jogador, "/personagens/nova", aFolhaPreenchida())
	id := oIDDoDestino(t, rec.Header().Get("Location"))

	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("herói: %v", err)
	}
	for _, categoria := range []string{"armas-simples", "armas-marciais", "armaduras-pesadas", "escudos"} {
		if !strings.Contains(row.Proficiencies, categoria) {
			t.Errorf("o guerreiro nasceu sem %q: %s", categoria, row.Proficiencies)
		}
	}

	pericias, err := f.s.queries.ListExpertisesByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("perícias: %v", err)
	}
	treinadas := map[string]bool{}
	for _, p := range pericias {
		if p.Trained != 0 {
			treinadas[p.Name] = true
		}
	}
	if !treinadas["Fortitude"] {
		t.Error("Fortitude é a perícia FIXA do guerreiro e não nasceu treinada")
	}
	// "Luta ou Pontaria" é escolha do bloco da classe: ela é pendência da ficha.
	if treinadas["Luta"] || treinadas["Pontaria"] {
		t.Error("a escolha entre Luta e Pontaria foi feita pela forja")
	}
}

// TestACompraDePontosDaForjaRecusaOQueOLivroProibe — p17, Tabela 1-1.
func TestACompraDePontosDaForjaRecusaOQueOLivroProibe(t *testing.T) {
	f := novoPiloto(t)
	rec := postaAForja(t, f, f.jogador, "/personagens/nova", aFolhaPreenchida())
	id := oIDDoDestino(t, rec.Header().Get("Location"))
	atributos := "/personagens/" + strconv.FormatInt(id, 10) + "/atributos"

	// Quatro em Força custa 7 e cabe.
	for i := 0; i < 4; i++ {
		if code := postaAForja(t, f, f.jogador, atributos+"/strength/1", nil).Code; code != http.StatusOK {
			t.Fatalf("subir Força %d: status %d", i+1, code)
		}
	}
	// O quinto passa do máximo de +4, e a recusa é CONTEÚDO em 200.
	recusado := postaAForja(t, f, f.jogador, atributos+"/strength/1", nil)
	if recusado.Code != http.StatusOK {
		t.Fatalf("a recusa veio em %d — o Datastar descarta remendo que não é 2xx", recusado.Code)
	}
	if frase := aRecusaDaCena(recusado.Body.String()); !strings.Contains(frase, "compra de pontos") {
		t.Errorf("a cena não disse por que recusou: %q", frase)
	}

	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("herói: %v", err)
	}
	if row.Strength != 4 {
		t.Errorf("Força ficou em %d — a recusa gravou mesmo assim", row.Strength)
	}
	// E a Constituição mexe no PV: subir um ponto sobe o poço, e ele fica cheio.
	if code := postaAForja(t, f, f.jogador, atributos+"/constitution/1", nil).Code; code != http.StatusOK {
		t.Fatalf("subir Constituição: status %d", code)
	}
	row, _ = f.s.queries.GetCharacter(context.Background(), id)
	// Con base +1 com a Constituição −1 do elfo dá Con 0: o poço volta a 20.
	if row.Hpmax != 20 || row.Hpcurrent != 20 {
		t.Errorf("PV %d/%d, esperado 20/20 com Con base +1 num elfo", row.Hpcurrent, row.Hpmax)
	}
}

// TestOsAtributosDaForjaSaoDoDono: a posse é conferida como em toda rota de
// personagem.
func TestOsAtributosDaForjaSaoDoDono(t *testing.T) {
	f := novoPiloto(t)
	rec := postaAForja(t, f, f.jogador, "/personagens/nova", aFolhaPreenchida())
	id := oIDDoDestino(t, rec.Header().Get("Location"))
	caminho := "/personagens/" + strconv.FormatInt(id, 10) + "/atributos"

	if code := postaAForja(t, f, f.mestre, caminho+"/strength/1", nil).Code; code != http.StatusForbidden {
		t.Errorf("o mestre distribuiu atributo de herói alheio: status %d", code)
	}
	if code := f.pede(t, f.mestre, http.MethodGet, caminho, "").Code; code != http.StatusForbidden {
		t.Errorf("o mestre abriu os atributos de herói alheio: status %d", code)
	}
}

// oIDDoDestino tira o id de "/personagens/7/atributos".
//
// Ele lia `partes[2]` porque o endereço começava em `/piloto`, e o prefixo saiu
// na ALE-280. Contar segmento por POSIÇÃO é o que quebra quando a rota muda de
// profundidade, e o modo de falhar é ruim: `ParseInt("atributos")` não diz que o
// endereço mudou, diz que um número está mal escrito.
func oIDDoDestino(t *testing.T, destino string) int64 {
	t.Helper()
	partes := strings.Split(strings.Trim(destino, "/"), "/")
	if len(partes) < 2 || partes[0] != "personagens" {
		t.Fatalf("destino inesperado: %q — esperava /personagens/{id}/…", destino)
	}
	id, err := strconv.ParseInt(partes[1], 10, 64)
	if err != nil {
		t.Fatalf("id no destino %q: %v", destino, err)
	}
	return id
}

func quantosHerois(t *testing.T, f pilotoFixture) int {
	t.Helper()
	lista, err := f.s.queries.ListCharactersByOwner(context.Background(), f.jogador)
	if err != nil {
		t.Fatalf("listar heróis: %v", err)
	}
	return len(lista)
}

// TestAFolhaEmBrancoPedeAEscolhaEmVezDeAcusarValorVazio: "não escolheu" e
// "escolheu o que não existe" chegam no mesmo campo e não são a mesma coisa.
func TestAFolhaEmBrancoPedeAEscolhaEmVezDeAcusarValorVazio(t *testing.T) {
	f := novoPiloto(t)
	campos := url.Values{"name": {"Sem escolhas"}}

	corpo := postaAForja(t, f, f.jogador, "/personagens/nova", campos).Body.String()
	for _, frase := range []string{
		"Escolha a linhagem do herói.", "Escolha o ofício do herói.", "Escolha a origem do herói.",
	} {
		if !strings.Contains(corpo, frase) {
			t.Errorf("a folha vazia não pede %q", frase)
		}
	}
	if strings.Contains(corpo, `"" não é`) {
		t.Error(`a folha acusou o vazio como valor desconhecido`)
	}
}
