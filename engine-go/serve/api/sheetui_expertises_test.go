package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"t20engine/infra/db/sqlcgen"
	"testing"
)

func seedOficio(t *testing.T, s *Server, id int64, name, attribute string) {
	t.Helper()
	_, err := s.sceneCore().Queries().CreateExpertise(context.Background(), sqlcgen.CreateExpertiseParams{
		Characterid: id, Name: name, Attribute: attribute, Trained: 1, Custom: 1,
	})
	if err != nil {
		t.Fatalf("semear o ofício %q: %v", name, err)
	}
}
func expertiseFixture(t *testing.T) (sceneFixture, int64) {
	t.Helper()
	f, id := fighterFixture(t)
	seedPericia(t, f.s, id, "Acrobacia", "dexterity", false)
	seedPericia(t, f.s, id, "Atuação", "charisma", false)
	seedOficio(t, f.s, id, "Ferreiro", "intelligence")
	return f, id
}

func expertiseScreen(t *testing.T, f sceneFixture, id int64, search string) string {
	t.Helper()
	target := fmt.Sprintf("/personagens/%d?tab=expertises", id)
	if search != "" {
		target += "&busca=" + url.QueryEscape(search)
	}
	return f.requests(t, f.player, http.MethodGet, target, "").Body.String()
}

// expertiseAt manda um dos gestos e devolve a tela redesenhada.
func expertiseAt(t *testing.T, f sceneFixture, id int64, path string) string {
	t.Helper()
	target := fmt.Sprintf("/personagens/%d/pericias/%s?tab=expertises", id, path)
	rec := f.requests(t, f.player, http.MethodPost, target, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("o comando %q respondeu %d: %s", path, rec.Code, rec.Body.String())
	}
	return expertiseScreen(t, f, id, "")
}

// training lê o que o BANCO guarda, que é a única fonte da verdade do gesto.
func training(t *testing.T, f sceneFixture, id int64, name string) (trained bool, attribute string) {
	t.Helper()
	all, err := f.s.sceneCore().Queries().ListExpertisesByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler as perícias: %v", err)
	}
	for _, e := range all {
		if e.Name == name {
			return e.Trained != 0, e.Attribute
		}
	}
	t.Fatalf("a perícia %q não está na ficha", name)
	return false, ""
}

// O PAINEL CHEGA NA TELA com os números do motor.
//
// Os esperados são escritos À MÃO, do combatente de nível 3: ½ nível 1, Luta
// treinada com Força 4 e treino 2, e as demais sem treino.
//
//	Luta      = 1 + 4 + 2 = +7
//	Fortitude = 1 + 3     = +4
//	Acrobacia = 1 + 2     = +3
func TestTheExpertisesPanelSaysTheEngineNumbers(t *testing.T) {
	f, id := expertiseFixture(t)
	screen := expertiseScreen(t, f, id, "")

	for _, want := range []string{"Detalhar Luta +7", "Detalhar Fortitude +4", "Detalhar Acrobacia +3"} {
		if !strings.Contains(screen, `aria-label="`+want+`"`) {
			t.Errorf("a tela não tem %q", want)
		}
	}
	// O CABEÇALHO diz as duas parcelas que valem para todas as linhas.
	if !strings.Contains(screen, "treino +2 • ½ nível 1") {
		t.Error("o cabeçalho não diz o treino e o ½ nível do personagem")
	}
}

// AS RESISTÊNCIAS VÊM PRIMEIRO, e os ofícios por último.
//
// "Teste de Reflexos CD 20" é a consulta mais quente da mesa. Prender a ordem é
// prender uma decisão de produto que uma reordenação alfabética inocente
// desfaria sem ninguém notar.
func TestTheOrderPutsSavesFirstAndCraftsLast(t *testing.T) {
	f, id := expertiseFixture(t)
	screen := expertiseScreen(t, f, id, "")

	position := func(name string) int { return strings.Index(screen, `aria-label="`+name+` treinada"`) }
	fort, refl, acro, craft := position("Fortitude"), position("Reflexos"), position("Acrobacia"), position("Ferreiro")
	if fort < 0 || refl < 0 || acro < 0 || craft < 0 {
		t.Fatalf("alguma linha não saiu: fort=%d refl=%d acro=%d oficio=%d", fort, refl, acro, craft)
	}
	if !(fort < refl && refl < acro) {
		t.Error("as resistências não vieram antes do resto da lista")
	}
	if craft < acro {
		t.Error("o ofício do jogador veio antes das perícias do livro")
	}
}

// A BUSCA IGNORA ACENTO, porque ninguém digita "Atuação" com o til.
func TestTheSearchFindsWithoutAccentAndWithoutCase(t *testing.T) {
	f, id := expertiseFixture(t)
	for _, term := range []string{"atuacao", "ATUACAO", "Atuação", "tuaç"} {
		screen := expertiseScreen(t, f, id, term)
		if !strings.Contains(screen, `aria-label="Atuação treinada"`) {
			t.Errorf("buscar %q não achou Atuação", term)
		}
		if strings.Contains(screen, `aria-label="Fortitude treinada"`) {
			t.Errorf("buscar %q trouxe Fortitude junto: o filtro não filtra", term)
		}
	}
	// SEM ACHADO a lista diz isso, em vez de ficar vazia parecendo defeito.
	if empty := expertiseScreen(t, f, id, "zzz"); !strings.Contains(empty, "Nenhuma perícia para") {
		t.Error("uma busca sem achado deixou a lista muda")
	}
}

// O `@get` DA BUSCA CARREGA A ABA.
//
// A varredura `TestNoSheetCommandLosesTheTab` olha só os `@post`, então este
// caminho precisa de guarda próprio: sem o `?tab=`, digitar na busca devolveria
// a ficha desenhada na PRIMEIRA aba — e a primeira aba É Perícias, o que faria o
// defeito parecer funcionar até alguém buscar de outra seção.
func TestTheSearchGetCarriesTheTab(t *testing.T) {
	f, id := expertiseFixture(t)
	screen := expertiseScreen(t, f, id, "")
	if !strings.Contains(screen, "?tab=expertises&#39;)") {
		t.Error("o `@get` da busca não carrega o `?tab=`: a resposta viria noutra aba")
	}
}

// O TREINO ALTERNA, e o comando manda a perícia e não o estado.
func TestTrainingTogglesBothWays(t *testing.T) {
	f, id := expertiseFixture(t)
	if trained, _ := training(t, f, id, "Acrobacia"); trained {
		t.Fatal("a Acrobacia começou treinada: o caso não mede a ida")
	}

	expertiseAt(t, f, id, "treino/"+url.PathEscape("Acrobacia"))
	if trained, _ := training(t, f, id, "Acrobacia"); !trained {
		t.Error("o primeiro toque não treinou a Acrobacia")
	}
	expertiseAt(t, f, id, "treino/"+url.PathEscape("Acrobacia"))
	if trained, _ := training(t, f, id, "Acrobacia"); trained {
		t.Error("o segundo toque não destreinou a Acrobacia: o comando manda o ESTADO em vez da perícia")
	}
}

// O ATRIBUTO TROCA, e um atributo inventado é recusado.
func TestTheAttributeSwitchesAndOnlyAcceptsTheSix(t *testing.T) {
	f, id := expertiseFixture(t)
	expertiseAt(t, f, id, "atributo/"+url.PathEscape("Acrobacia")+"/strength")
	if _, attribute := training(t, f, id, "Acrobacia"); attribute != "strength" {
		t.Errorf("a Acrobacia ficou em %q, quer strength", attribute)
	}

	target := fmt.Sprintf("/personagens/%d/pericias/atributo/Acrobacia/sorte?tab=expertises", id)
	if refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, target, "").Body.String()); refusal == "" {
		t.Error("um atributo inventado foi aceito sem uma palavra na tela")
	}
	if _, attribute := training(t, f, id, "Acrobacia"); attribute != "strength" {
		t.Error("a recusa mexeu no banco assim mesmo")
	}
}

// O OFÍCIO ACEITA TREINO E ATRIBUTO.
//
// Exigir que o nome seja uma das 29 do livro recusa todo ofício com 400, e a
// tela desenha o botão de treino e o seletor em TODA linha — promessa de tela
// que o servidor não cumpre, invisível para quem não mexe num ofício depois de
// criá-lo.
func TestACraftAcceptsTrainingAndAnAttribute(t *testing.T) {
	f, id := expertiseFixture(t)

	expertiseAt(t, f, id, "treino/Ferreiro")
	if trained, _ := training(t, f, id, "Ferreiro"); trained {
		t.Error("o ofício não destreinou: o servidor recusa editar o que não é do livro")
	}
	expertiseAt(t, f, id, "atributo/Ferreiro/dexterity")
	if _, attribute := training(t, f, id, "Ferreiro"); attribute != "dexterity" {
		t.Errorf("o ofício ficou em %q, quer dexterity: o servidor recusou a troca", attribute)
	}
}

// O OFÍCIO NASCE TREINADO E SE REMOVE; a perícia do livro NÃO se remove.
//
// A recusa é do SERVIDOR e não da tela: a ficha não desenha lixeira numa perícia
// do livro, mas travar só na UI deixaria a regra sem fronteira — quem montar o
// `@post` à mão apagaria a Fortitude.
func TestACraftIsBornTrainedAndOnlyItCanBeRemoved(t *testing.T) {
	f, id := expertiseFixture(t)
	if trained, _ := training(t, f, id, "Ferreiro"); !trained {
		t.Error("o ofício não nasceu treinado")
	}

	expertiseAt(t, f, id, "remover/Ferreiro")
	all, _ := f.s.sceneCore().Queries().ListExpertisesByCharacter(context.Background(), id)
	for _, e := range all {
		if e.Name == "Ferreiro" {
			t.Fatal("o ofício sobreviveu ao remover")
		}
	}

	target := fmt.Sprintf("/personagens/%d/pericias/remover/Fortitude?tab=expertises", id)
	if refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, target, "").Body.String()); refusal == "" {
		t.Error("uma perícia do LIVRO foi removida da ficha")
	}
	if _, _ = training(t, f, id, "Fortitude"); false {
		t.Error("inalcançável")
	}
}

// A REGRA DO NOME É UMA SÓ.
//
// Um ofício não pode ROUBAR o nome de uma das 29: a ficha passaria a ter duas
// linhas com o mesmo nome, e a decomposição de uma cairia sobre a outra.
//
// As três recusas saem do MESMO gesto que grava (`AddCraft`), e é isso que a
// ALE-350 mudou: antes a regra devolvia `nil` e quem inseria era a cena, uma
// camada acima. O "já tem" hoje vem da `UNIQUE (characterId, name)` traduzida,
// e não de uma leitura anterior à escrita.
func TestACraftDoesNotStealTheNameOfABookExpertise(t *testing.T) {
	f, id := expertiseFixture(t)
	crafts := f.s.characterPlays()
	howMany := func() int {
		return countRows(t, f.s, fmt.Sprintf(
			"SELECT COUNT(*) FROM character_expertises WHERE characterId = %d", id))
	}
	before := howMany()
	cases := []struct {
		name    string
		failure string
	}{
		{"", "dê um nome"},
		{"Fortitude", "é uma perícia do livro"},
		{"Ferreiro", "já tem"},
	}
	for _, tc := range cases {
		err := crafts.AddCraft(context.Background(), id, tc.name, "intelligence")
		if err == nil {
			t.Errorf("o nome %q foi aceito", tc.name)
			continue
		}
		if !strings.Contains(err.Error(), tc.failure) {
			t.Errorf("o nome %q deu %q, e a mensagem devia falar de %q", tc.name, err, tc.failure)
		}
	}
	if err := crafts.AddCraft(context.Background(), id, "Marinheiro", "intelligence"); err != nil {
		t.Errorf("um nome legítimo foi recusado: %v", err)
	}
	// E o legítimo GRAVOU: sem esta metade, um `AddCraft` que recusasse tudo e
	// devolvesse `nil` no fim passaria verde em todos os casos acima.
	if got := countRows(t, f.s, fmt.Sprintf(
		"SELECT COUNT(*) FROM character_expertises WHERE characterId = %d AND name = 'Marinheiro'", id)); got != 1 {
		t.Errorf("o ofício aceito não foi gravado: %d linhas de Marinheiro", got)
	}
	// E as recusadas NÃO gravaram: a ficha cresceu de UMA, que é o Marinheiro.
	// Medido contra o que a bancada semeou, e não contra um número escrito à mão
	// — a `expertiseFixture` monta um guerreiro inteiro, e chutar o total dele
	// foi o primeiro erro deste caso.
	if after := howMany(); after != before+1 {
		t.Errorf("a ficha foi de %d para %d perícias, esperado exatamente uma a mais", before, after)
	}
}

// A RECUSA DO NOME REPETIDO CHEGA AO JOGADOR, e não o erro do driver.
//
// Este caso NASCEU VERDE, e a honestidade sobre isso é parte dele: a ALE-350
// suspeitou de uma janela entre a conferência e o `INSERT` — que moravam em
// camadas diferentes — e foi medir. Quatro pedidos na mesma largada, três
// corridas: o banco ficou com uma linha em todas, e as doze respostas trouxeram
// a recusa DESENHADA. O DSN abre com `_txlock=immediate` e `busy_timeout`, então
// a escrita serializa e o perdedor já enxerga a linha do vencedor.
//
// O que ele prende, então, não é a regra — essa é do
// `TestACraftDoesNotStealTheNameOfABookExpertise`, uma camada abaixo. É a
// LIGAÇÃO: a rota chega na regra, e o texto do driver não alcança a tela.
func TestTheSecondCraftWithTheSameNameSaysWhyInsteadOfLeakingTheDriver(t *testing.T) {
	f, id := expertiseFixture(t)
	const body = `{"new_expertise":"Marinheiro","new_attribute":"intelligence"}`
	path := fmt.Sprintf("/personagens/%d/pericias/nova?tab=expertises", id)

	first := f.posts(t, f.player, path, body)
	second := f.posts(t, f.player, path, body)

	// O CONTROLE: sem ele, "a segunda recusou" não diz se a primeira gravou.
	if strings.Contains(first, "UNIQUE") || strings.Contains(first, "constraint") {
		t.Fatalf("a PRIMEIRA já falhou — o caso não chegou a medir a segunda:\n%s", first)
	}
	howMany := countRows(t, f.s, fmt.Sprintf(
		"SELECT COUNT(*) FROM character_expertises WHERE characterId = %d AND name = 'Marinheiro'", id))
	if howMany != 1 {
		t.Fatalf("o banco tem %d linhas de Marinheiro, esperado exatamente 1", howMany)
	}
	if !strings.Contains(second, "já tem") {
		t.Errorf("a segunda tentativa não explicou por quê:\n%s", second)
	}
	// E o texto do DRIVER não chega à tela: ele nomeia tabela e coluna, que é
	// dizer a estranho como o banco é feito.
	for _, leak := range []string{"UNIQUE", "constraint", "character_expertises"} {
		if strings.Contains(second, leak) {
			t.Errorf("o erro do driver vazou para a tela (%q):\n%s", leak, second)
		}
	}
}

// A ISENÇÃO DA PENALIDADE CHEGA À DECOMPOSIÇÃO DA PERÍCIA (p339, ALE-437).
//
// "Você recebe +5 em Furtividade e ignora a penalidade de armadura do item para
// testes dessa perícia" — o encanto Sombrio. Na tela isso tem de aparecer como
// as duas coisas que ele faz: o bônus e a penalidade que deixou de pesar.
//
// O caso usa ARMADURA e ESCUDO de propósito: a isenção é do item que carrega o
// encanto, então a do escudo continua valendo. Com uma peça só, uma isenção que
// apagasse a penalidade inteira passaria verde.
func TestTheEnchantExemptsOneExpertiseFromTheArmorPenaltyOnScreen(t *testing.T) {
	f, id := expertiseFixture(t)
	seedPericia(t, f.s, id, "Furtividade", "dexterity", false)
	// PROFICIENTE nas duas peças de propósito: sem proficiência o livro cobra
	// uma penalidade SEPARADA, em TODA perícia (p142), e ela não é a penalidade
	// de armadura que o encanto isenta. Com ela no meio, os dois números do
	// caso deixariam de dizer o que o Sombrio faz.
	if _, err := f.s.db.ExecContext(context.Background(),
		`UPDATE characters SET proficiencies = ? WHERE id = ?`,
		`["armaduras-pesadas","escudos"]`, id,
	); err != nil {
		t.Fatalf("semear as proficiências: %v", err)
	}
	armadura := itemSemeia(t, f, id, "armadura-completa", "Armadura completa", "vested")
	itemSemeia(t, f, id, "escudo-pesado", "Escudo pesado", "wielded")
	if err := f.s.characterPlays().EnchantItem(
		context.Background(), armadura, []string{"encanto-sombrio"},
	); err != nil {
		t.Fatalf("encantar a armadura: %v", err)
	}

	screen := expertiseScreen(t, f, id, "")
	// O CONTROLE é a Acrobacia: mesma perícia penalizada, sem encanto nenhum, e
	// ela continua pagando as duas peças.
	if !strings.Contains(screen, `aria-label="Detalhar Acrobacia -4"`) {
		t.Fatalf("a Acrobacia não sofreu as duas penalidades: ½ nível 1 + DES 2 − 5 da " +
			"armadura − 2 do escudo = −4")
	}
	// E A FURTIVIDADE: 1 + 2 + 5 do encanto − 2 do escudo = +6. A armadura saiu
	// da conta, e só ela.
	if !strings.Contains(screen, `aria-label="Detalhar Furtividade +6"`) {
		t.Errorf("a Furtividade não ficou em +6 (½ nível 1 + DES 2 + 5 do Sombrio − 2 do escudo)")
	}
	// E A LINHA DIZ POR QUÊ: um número que muda sem motivo escrito é a ficha
	// escondendo o que o item faz (ALE-430).
	if !strings.Contains(screen, "a penalidade deste item não se aplica") {
		t.Error("a decomposição não diz que a penalidade da armadura foi isentada")
	}
}

// O BENEFÍCIO DE PERÍCIA DA ORIGEM TREINA, E A ABA DIZ QUEM TREINOU (p85, ALE-490).
//
// O gesto é o da aba Poderes, pela rota de verdade, e o esperado é escrito à
// mão: o combatente soldado de nível 3 tem Fortitude ½ nível 1 + Constituição 3
// = +4, e escolhendo o benefício passa a +6 com o treino.
//
// A coluna `trained` NÃO muda — quem treina é a origem —, e por isso o
// interruptor fica marcado e travado: alternar a coluna não mudaria nada, e um
// interruptor que não muda nada mente.
func TestTheOriginSkillBenefitTrainsAndTheTabSaysWhoTrainedIt(t *testing.T) {
	f, id := expertiseFixture(t)
	if !strings.Contains(expertiseScreen(t, f, id, ""), `aria-label="Detalhar Fortitude +4"`) {
		t.Fatal("o controle já estava errado: sem o benefício, a Fortitude não é +4")
	}

	target := fmt.Sprintf("/personagens/%d/poderes/origem/origin-soldado-pericia-fortitude?tab=powers", id)
	if rec := f.requests(t, f.player, http.MethodPost, target, ""); rec.Code != http.StatusOK {
		t.Fatalf("escolher o benefício respondeu %d: %s", rec.Code, rec.Body.String())
	}
	screen := expertiseScreen(t, f, id, "")

	if !strings.Contains(screen, `aria-label="Detalhar Fortitude +6"`) {
		t.Error("o soldado que escolhe a perícia Fortitude não ficou com +6 (1 + 3 + treino 2)")
	}
	// A TAG do interruptor inteira, e não um trecho dela: a ordem dos atributos
	// é do `templ`, e ninguém a prometeu.
	button := regexp.MustCompile(`<button[^>]*aria-label="Fortitude treinada"[^>]*>`).FindString(screen)
	if button == "" {
		t.Fatal("a aba não desenhou o interruptor de treino da Fortitude")
	}
	for _, want := range []string{`aria-checked="true"`, `aria-disabled="true"`} {
		if !strings.Contains(button, want) {
			t.Errorf("o interruptor da Fortitude não tem %s: a tela diz que ela não é "+
				"treinada, ou deixa alternar, enquanto o total soma o treino da origem\n    %s", want, button)
		}
	}
	if strings.Contains(button, "data-on:click") {
		t.Errorf("o interruptor travado ainda manda o gesto, que alternaria a coluna sem efeito\n    %s", button)
	}
	if !strings.Contains(screen, "Treinada pela Origem: Soldado") {
		t.Error("a aba não diz de onde veio o treino da Fortitude")
	}
	if trained, _ := training(t, f, id, "Fortitude"); trained {
		t.Error("escolher o benefício escreveu a coluna `trained`: quem treina é a origem, " +
			"e a coluna é das perícias da classe e das escolhas livres")
	}
}
