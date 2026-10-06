package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// A SUPERFÍCIE AÇÕES, DA FICHA ATÉ A MESA (ALE-423, p233).
//
//	"No seu turno, você pode fazer uma ação padrão e uma ação de movimento, em
//	 qualquer ordem… Você também pode abrir mão das duas para fazer uma ação
//	 completa." (p233)
//
// O QUE ELA CONSERTA está medido na issue: na vez dele, o jogador via `SUA VEZ`
// e nenhuma oferta. Para atacar ele ia a Ficha → Combate pelo número, voltava a
// Ficha → Poderes para lembrar da Fúria, e conferia o PM no rodapé — três
// trocas de aba para uma ação.
//
// INTEGRAÇÃO porque o que pode quebrar é a COMPOSIÇÃO: os números já existem
// nos painéis de Combate, Poderes e Magias, e o que esta fatia faz é REAGRUPAR
// os três pelo que a vez custa. Um teste de unidade do agrupador provaria a
// tabela e nada sobre as linhas chegarem à tela que o jogador abre.
//
// O QUE ESTE ARQUIVO NÃO MEDE são os NÚMEROS. Eles são da aba Combate e das
// perícias, e lá eles já têm caso: uma regra é presa UMA vez, onde ela mora.
// Aqui a pergunta é o AGRUPAMENTO, que é a decisão nova.

// barbarianAtTheTable é a bancada: um bárbaro de machado na mão, e ele é o
// ÚNICO personagem do jogador na campanha.
//
// ÚNICO não é detalhe de arrumação: a mesa resolve "quem sou eu" como o PRIMEIRO
// membro que pertence a quem olha (`tableRoster`), então um segundo personagem
// do mesmo dono faria a cena desenhar a ficha do outro — e o caso mediria a
// superfície de um arcanista sem Fúria nenhuma, em verde.
//
// O BÁRBARO e não o arcanista da bancada comum porque ele tem as três chaves de
// ação numa classe só: a Fúria é `livre`, o Brado Assustador é `movimento`, e a
// Esquiva Sobrenatural é `passivo`. Um caso por chave sairia caro; esta ficha os
// traz juntos, e é por isso que ela carrega o nome dela.
func barbarianAtTheTable(t *testing.T, level int64) (sceneFixture, int64) {
	t.Helper()
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t.com")
	player := seedUser(t, s, "jogador@t.com")
	campaignID := seedCampaign(t, s, gm)
	sessionID := seedSession(t, s, campaignID)

	id, err := s.queries.CreateCharacter(context.Background(), sqlcgen.CreateCharacterParams{
		OwnerId: player, Name: "Furioso", Origin: "batedor", Level: level,
		Strength: 4, Dexterity: 2, Constitution: 3, Intelligence: 0, Wisdom: 1, Charisma: 0,
		Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: "[]", ClassPowers: "[]", ClassChoices: "{}", PowerChoices: "{}",
		CreatedAt: dbvalue.NowISO(), UpdatedAt: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear o bárbaro: %v", err)
	}
	seedClasse(t, s, id, "Bárbaro", level)
	seedMember(t, s, campaignID, id)

	f := sceneFixture{s: s, gm: gm, player: player,
		campaignID: campaignID, sessionID: sessionID, charID: id}
	// A ARMA NA MÃO é o que faz a linha de Atacar existir: sem ela a aba Combate
	// não desenha cartão nenhum, e a superfície mediria uma ficha de mãos vazias.
	if _, err := s.queries.CreateItem(context.Background(), sqlcgen.CreateItemParams{
		Characterid: id, Catalogid: sql.NullString{String: "machado-batalha", Valid: true},
		Name: "Machado de batalha", Quantity: 1, Slots: 1,
		Equipped:     sql.NullString{String: "wielded", Valid: true},
		Improvements: "[]", Createdat: dbvalue.NowISO(),
	}); err != nil {
		t.Fatalf("empunhar o machado: %v", err)
	}
	// O BRADO É ESCOLHIDO, e pelo gesto da ficha e não por um `UPDATE`: ele é um
	// poder de bárbaro de vaga, e gravar a coluna à mão montaria uma ficha que o
	// app não sabe produzir.
	if refused := powerCommand(t, f, id, "escolhe/class.barbaro.brado-assustador", ""); refused != "" {
		t.Fatalf("escolher o Brado Assustador: %s", refused)
	}
	return f, id
}

// acoesDaMesa é a cena da mesa como o JOGADOR a recebe.
//
// Pela mesa e não pela rota da ficha: a superfície é da sessão, porque é lá que
// o jogador está na vez dele — e é a composição mesa+ficha que esta fatia monta.
func acoesDaMesa(t *testing.T, f sceneFixture) string {
	t.Helper()
	rec := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("abrir a mesa deu %d", rec.Code)
	}
	return rec.Body.String()
}

// entre devolve o que a cena escreveu ENTRE dois rótulos de grupo.
//
// É o instrumento que torna "sob a ação padrão" uma pergunta respondível por um
// teste de HTML: sem recortar, `strings.Contains(cena, "Mover")` é verdadeiro
// mesmo com o Mover no grupo errado, e o caso passaria verde medindo apenas que
// a palavra existe em algum lugar da página.
//
// Ele FALHA quando não acha o começo, em vez de devolver vazio: um recorte vazio
// e um grupo sem a linha procurada se parecem na asserção seguinte.
func entre(t *testing.T, cena, comeco, fim string) string {
	t.Helper()
	i := strings.Index(cena, comeco)
	if i < 0 {
		t.Fatalf("a cena não tem o grupo %q — os grupos escritos são %v",
			comeco, gruposEscritos(cena))
	}
	resto := cena[i+len(comeco):]
	if j := strings.Index(resto, fim); j >= 0 {
		return resto[:j]
	}
	return resto
}

// gruposEscritos existe para a MENSAGEM DE FALHA: "a cena não tem Ação padrão"
// manda procurar, e "os grupos escritos são [Ação de movimento]" diz o que
// aconteceu.
func gruposEscritos(cena string) []string {
	var achados []string
	for _, titulo := range []string{
		"Ação padrão", "Ação de movimento", "Ação completa",
		"Livre e reação", "Custo que a mesa decide",
	} {
		if strings.Contains(cena, titulo) {
			achados = append(achados, titulo)
		}
	}
	return achados
}

// superficieDeAcoes recorta o PAINEL, pelo id dele.
//
// Pelo id e não pelo rótulo do botão: a cena inteira vem servida com a Ficha
// dentro, e `Contains(cena, "Passivas")` casa o rótulo da ABA PODERES — foi
// exatamente o que fez a primeira versão destes casos medir a ficha em vez do
// painel.
func superficieDeAcoes(t *testing.T, cena string) string {
	t.Helper()
	return entre(t, cena, `id="actions-scene"`, `$surface === &#34;mesa&#34;`)
}

// A SUPERFÍCIE ESTÁ NO SELETOR, que é o que a torna alcançável.
//
// Pelo SELETOR e não por um `Contains` na cena inteira: o painel é desenhado
// escondido (`data-show`), então o HTML dele existe antes de qualquer clique —
// procurar "Ação padrão" na página provaria que o servidor montou o painel e
// nada sobre haver botão que o abra.
//
// AQUI NÃO SE AFIRMA NADA SOBRE O MESTRE, e a razão foi medida: a cena do
// mestre não tem seletor de superfície NENHUM — ele não recebe "Mesa",
// "Tabuleiro" nem "O que ver na sessão". Um `!Contains(mestre, "Ações")` seria
// verdadeiro para sempre e por acidente, que é verde sem medição.
func TestTheActionsSurfaceIsReachableFromTheSelector(t *testing.T) {
	f := newSceneFixture(t)
	cena := acoesDaMesa(t, f)

	seletor := entre(t, cena, `aria-label="O que ver na sessão"`, "</div>")
	if !strings.Contains(seletor, ">Ações<") {
		t.Errorf("o seletor de superfícies não oferece Ações:\n%s", recorte(seletor))
	}
	// O CONTROLE: sem ele, um recorte que errasse o fim daria uma string vazia e
	// a asserção de cima reprovaria por outro motivo que não o medido.
	if !strings.Contains(seletor, ">Mesa<") {
		t.Fatalf("o recorte do seletor não pegou nem a Mesa, que sempre está lá:\n%s",
			recorte(seletor))
	}
}

// CADA LINHA CAI NO GRUPO QUE O LIVRO LHE DÁ, e este é o caso que a fatia
// inteira existe para prender.
//
// O agrupamento lê a CHAVE CRUA do catálogo (`padrao`, `movimento`, …), nunca o
// custo já escrito ("MOVIMENTO · 1 PM"): um analisador do texto de tela mediria
// o que foi ESCRITO e não o que foi declarado, e a primeira mudança de rótulo o
// mandaria para o grupo errado em silêncio.
//
// O BRADO ASSUSTADOR é o caso que separa "funciona" de "funciona por acaso": ele
// é a ÚNICA habilidade de bárbaro que custa uma ação de MOVIMENTO, e um
// agrupador que jogasse todo poder na ação padrão passaria em tudo menos nele.
func TestEveryRowLandsInTheGroupTheBookGivesIt(t *testing.T) {
	f, barbaro := barbarianAtTheTable(t, 6)

	cena := acoesDaMesa(t, f)
	padrao := entre(t, cena, "Ação padrão", "Ação de movimento")
	movimento := entre(t, cena, "Ação de movimento", "Ação completa")

	// MOVER é a ação de movimento do livro ("percorrer uma distância igual a seu
	// deslocamento", p233), e ela é a linha que toda ficha tem.
	if !strings.Contains(movimento, "Mover") {
		t.Errorf("o Mover não está na ação de movimento:\n%s", recorte(movimento))
	}
	if strings.Contains(padrao, "Mover") {
		t.Errorf("o Mover caiu na ação padrão, e a p233 o põe no movimento")
	}

	// O BRADO (`action: movimento` no catálogo) segue a chave dele.
	if !strings.Contains(movimento, "Brado Assustador") {
		t.Errorf("o Brado Assustador custa uma ação de MOVIMENTO no catálogo e não está "+
			"nesse grupo:\n%s", recorte(movimento))
	}
	if strings.Contains(padrao, "Brado Assustador") {
		t.Errorf("o Brado Assustador caiu na ação padrão")
	}

	// A FÚRIA é postura e `action: livre` — entrar nela não gasta a vez (p41).
	livres := entre(t, cena, "Livre e reação", "</section>")
	if !strings.Contains(livres, "Fúria") {
		t.Errorf("a Fúria custa uma ação LIVRE e não está entre as que não gastam a vez:\n%s",
			recorte(livres))
	}
	_ = barbaro
}

// A ARMA EMPUNHADA VIRA A LINHA DE ATACAR, e ela é a primeira da ação padrão.
//
// Primeira porque é a ação mais comum do turno e porque o livro a nomeia assim
// ("fazer um ataque ou lançar uma magia são as ações padrão mais comuns", p233)
// — e porque a 390px a primeira linha é a única que se lê sem rolar.
func TestTheWieldedWeaponIsTheFirstStandardAction(t *testing.T) {
	f, _ := barbarianAtTheTable(t, 6)

	padrao := entre(t, acoesDaMesa(t, f), "Ação padrão", "Ação de movimento")
	if !strings.Contains(padrao, "Atacar") {
		t.Fatalf("a ação padrão não oferece Atacar:\n%s", recorte(padrao))
	}
	if i, j := strings.Index(padrao, "Atacar"), strings.Index(padrao, "Manobra"); j >= 0 && i > j {
		t.Errorf("a Manobra vem antes do Atacar, e o ataque é a ação mais comum do turno")
	}
}

// AS PASSIVAS SÃO UMA CONTAGEM e nunca linhas.
//
// Elas já entram nos números da ficha — a Redução de Dano que o bárbaro tem já
// está dentro da RD que a aba Combate mostra —, então uma linha por passiva
// encheria a tela de coisas sobre as quais não há o que fazer. São 238 das 411
// ativações do catálogo, e é esse o tamanho do ruído evitado.
//
// O CONTROLE VEM PRIMEIRO, e ele não é zelo: a primeira versão deste caso
// afirmava que a "Esquiva Sobrenatural" não estava no painel — e ela não estava
// porque este bárbaro NÃO A TEM. A ausência era verdadeira e não media nada; o
// caso passou verde com a sabotagem que põe toda passiva na tela. Por isso a
// ordem: prove que o poder existe na ficha, depois leia a ausência dele aqui.
func TestThePassivesAreCountedAndNeverListed(t *testing.T) {
	f, id := barbarianAtTheTable(t, 6)

	// O CONTROLE LÊ O BLOCO DE PASSIVAS da aba Poderes, e não a aba inteira: o
	// DIÁLOGO DE ESCOLHER poder lista o acervo da classe, então um `Contains` na
	// página casa poder que o personagem apenas PODERIA ter. Medido — com a aba
	// inteira, "Esquiva Sobrenatural" passava no controle e o bárbaro não a tem.
	ficha := f.requests(t, f.player, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=abilities", id), "").Body.String()
	tidas := entre(t, ficha, "Passivas ·", "</section>")
	if !strings.Contains(tidas, passivaDoBarbaro) {
		t.Fatalf("o controle falhou: este bárbaro não TEM %q, e a ausência dela no "+
			"painel não provaria nada:\n%s", passivaDoBarbaro, recorte(tidas))
	}

	painel := superficieDeAcoes(t, acoesDaMesa(t, f))
	if strings.Contains(painel, passivaDoBarbaro) {
		t.Errorf("a passiva %q virou linha da superfície de Ações — ela já está dentro "+
			"dos números da ficha, e não há o que acionar", passivaDoBarbaro)
	}
	if !strings.Contains(painel, "Passivas ·") {
		t.Errorf("o painel não diz quantas passivas existem, e aí \"não tenho\" e "+
			"\"não mostrei\" viram a mesma coisa:\n%s", recorte(painel))
	}
}

// passivaDoBarbaro é uma passiva que o bárbaro de nível 6 TEM — as dele são a
// Redução de Dano e o Instinto Selvagem. Escrita aqui e não no corpo do caso
// porque ela é o que o controle e a asserção comparam, e duas grafias fariam o
// controle provar a existência de uma coisa e a asserção procurar outra.
const passivaDoBarbaro = "Redução de Dano"

// recorte encurta um pedaço de HTML para caber numa mensagem de falha sem
// despejar a cena inteira no terminal.
func recorte(s string) string {
	if len(s) > 600 {
		return s[:600] + "…"
	}
	return s
}

// O RAMO DE QUEM CONJURA, e ele é um CASO À PARTE porque a tela ramifica pelo
// DADO: metade das classes tem magia, e percorrer a superfície de um guerreiro
// não diz nada sobre a de um arcanista.
//
// A MAGIA CAI NO GRUPO QUE A EXECUÇÃO DELA CUSTA, lida da mesma chave crua que
// o `spells_commands.go` já cobra do turno. A Bola de Fogo é `padrao` (p177), e
// uma superfície que empilhasse magia num bloco próprio diria a um conjurador
// que ele tem duas economias de ação, quando ele tem uma.
func TestTheSpellLandsInTheGroupItsExecutionCosts(t *testing.T) {
	f := newSceneFixture(t)
	if rec := f.requests(t, f.player, http.MethodPost,
		fmt.Sprintf("/personagens/%d/magias/aprende/bola-de-fogo?tab=spells", f.charID),
		""); rec.Code != http.StatusOK {
		t.Fatalf("aprender a Bola de Fogo deu %d", rec.Code)
	}

	cena := acoesDaMesa(t, f)
	padrao := entre(t, cena, "Ação padrão", "Ação de movimento")
	if !strings.Contains(padrao, "Bola de Fogo") {
		t.Errorf("a Bola de Fogo é uma ação PADRÃO e não está nesse grupo:\n%s", recorte(padrao))
	}
	// O VERBO vem antes do nome, como em toda linha desta superfície: uma lista
	// em que umas linhas começam por verbo e outras por substantivo obriga a ler
	// cada uma para saber o que ela é.
	if !strings.Contains(padrao, "Conjurar") {
		t.Errorf("a magia entrou sem o verbo que as outras linhas têm")
	}
}

// ── OS CHIPS DE MODIFICADOR (fatia 2) ────────────────────────────────────────
//
// O chip é o interruptor que já existe — a postura dos Poderes, o situacional
// dos Efeitos — desenhado SOBRE O NÚMERO QUE ELE MUDA, em vez de em outra aba.
//
// É a metade da dor que a fatia 1 não cura: ver o `+8` não lembra ninguém de que
// a Fúria existe. Ligá-la pela aba Poderes exige saber que ela existe, que é o
// que a pessoa esqueceu.

// A FÚRIA APARECE NO CARTÃO DE ATACAR, porque é o ataque que ela muda.
//
// O roteamento lê o ALVO do modificador (`{k: attack}`, `{k: damage}`), nunca o
// nome do poder: uma lista de "quais poderes vão no ataque" seria uma lista de
// PROIBIDOS às avessas, que subconta em silêncio a cada poder novo do catálogo.
func TestTheStanceChipRidesTheNumberItChanges(t *testing.T) {
	f, _ := barbarianAtTheTable(t, 6)

	painel := superficieDeAcoes(t, acoesDaMesa(t, f))
	padrao := entre(t, painel, "Ação padrão", "Ação de movimento")
	if !strings.Contains(padrao, "Fúria") {
		t.Errorf("a ação padrão não oferece a Fúria, que soma +2 no ataque e no "+
			"dano (p41):\n%s", recorte(padrao))
	}
	// O PREÇO VAI JUNTO: um interruptor que cobra PM sem dizer quanto faz a
	// pessoa descobrir o custo depois de pagar.
	if !strings.Contains(padrao, "PM") {
		t.Errorf("o chip da Fúria não diz o que custa:\n%s", recorte(padrao))
	}
	// E ELA NÃO ENTRA NO MOVIMENTO: o deslocamento não é ataque nem dano, e um
	// interruptor oferecido sobre um número que ele não muda é pior que nenhum.
	movimento := entre(t, painel, "Ação de movimento", "Ação completa")
	if strings.Contains(movimento, "Fúria") {
		t.Errorf("a Fúria apareceu na ação de movimento, e ela não muda "+
			"deslocamento:\n%s", recorte(movimento))
	}
	// UMA VEZ POR GRUPO, e não uma por linha: com ela ligada, por linha seriam
	// três botões "Encerrar Fúria" idênticos empilhados.
	if n := strings.Count(padrao, "Fúria"); n != 1 {
		t.Errorf("a Fúria aparece %d vezes na ação padrão, e o grupo a oferece UMA", n)
	}
}

// LIGAR O CHIP MOVE O NÚMERO, e é isto que separa esta fatia de um crachá
// bonito: o `+8` vira `+10` na mesma tela, e o PM sai do poço.
//
// INTEGRAÇÃO pelo gesto de verdade, porque o que pode quebrar é a composição: o
// chip chama o comando que a aba Poderes já tinha, o motor recomputa com a flag
// ligada, e a superfície tem de ser REDESENHADA — ela é um nó irmão do
// `#sheet-scene`, e um comando que remendasse só a ficha deixaria o número velho
// na tela.
func TestTurningTheChipOnMovesTheNumberOnTheSameScreen(t *testing.T) {
	f, id := barbarianAtTheTable(t, 6)

	antes := ataqueDoCartao(t, superficieDeAcoes(t, acoesDaMesa(t, f)))

	rec := f.requests(t, f.player, http.MethodPost,
		fmt.Sprintf("/personagens/%d/poderes/postura/furia/entra?embutida=1", id),
		`{"stance_degrees":0}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("entrar em Fúria deu %d", rec.Code)
	}
	// O REMENDO VOLTA COM A SUPERFÍCIE, e não só com a ficha.
	if !strings.Contains(rec.Body.String(), `id="actions-scene"`) {
		t.Errorf("o comando redesenhou a ficha e não a superfície de Ações — o número " +
			"ficaria velho na tela até alguém recarregar")
	}

	depois := ataqueDoCartao(t, superficieDeAcoes(t, acoesDaMesa(t, f)))
	if depois <= antes {
		t.Errorf("a Fúria foi ligada e o ataque ficou em %+d, vindo de %+d — a p41 dá "+
			"+2, e o número tem de se mover", depois, antes)
	}
}

// ataqueDoCartao lê o número que o cartão de Atacar mostra.
//
// Pelo DELTA e não por um número escrito à mão: o bônus de ataque do bárbaro sai
// de Luta mais atributo mais o que a arma dá, e cravá-lo aqui seria reimplementar
// a conta da aba Combate — o erro que o `CLAUDE.md` chama de derivar o esperado
// do código sob teste.
func ataqueDoCartao(t *testing.T, painel string) int {
	t.Helper()
	cartao := entre(t, painel, "Atacar · Machado de batalha", "Manobra")
	achados := regexp.MustCompile(`>([+-]\d+)<`).FindStringSubmatch(cartao)
	if achados == nil {
		t.Fatalf("o cartão de Atacar não mostra número nenhum:\n%s", recorte(cartao))
	}
	n, err := strconv.Atoi(achados[1])
	if err != nil {
		t.Fatalf("o cartão mostra %q, que não é número", achados[1])
	}
	return n
}
