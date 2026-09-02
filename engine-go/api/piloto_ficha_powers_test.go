package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"t20engine/db/sqlcgen"
	"t20engine/plataforma"
)

// Os guardas da aba PODERES (ALE-272, fatia 8).
//
// O que eles prendem é a REGRA — o que o personagem possui, o que ele pode
// ativar agora, e o que entrar numa postura custa — e a decisão de tela que
// separa o que se USA do que só está lá.

// oBarbaro é a ficha das posturas: a Fúria é a postura de escala do livro e a
// Alma de Bronze é o único poder do catálogo que CONCEDE algo ao entrar nela.
func oBarbaro(t *testing.T, nivel int64) (pilotoFixture, int64) {
	t.Helper()
	f := novoPiloto(t)
	id, err := f.s.queries.CreateCharacter(context.Background(), sqlcgen.CreateCharacterParams{
		OwnerId: f.jogador, Name: "Furioso", Origin: "Batedor", Level: nivel,
		HpMax: 60, HpCurrent: 60, MpMax: 20, MpCurrent: 20,
		Strength: 4, Dexterity: 2, Constitution: 3, Intelligence: 0, Wisdom: 1, Charisma: 0,
		Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: "[]", ClassPowers: "[]", ClassChoices: "{}", PowerChoices: "{}",
		CreatedAt: plataforma.NowISO(), UpdatedAt: plataforma.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear o bárbaro: %v", err)
	}
	seedClasse(t, f.s, id, "Bárbaro", nivel)
	return f, id
}

func aTelaDosPoderes(t *testing.T, f pilotoFixture, id int64) string {
	t.Helper()
	return f.pede(t, f.jogador, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=abilities", id), "").Body.String()
}

func oComandoDoPoder(t *testing.T, f pilotoFixture, id int64, caminho, corpo string) string {
	t.Helper()
	alvo := fmt.Sprintf("/personagens/%d/poderes/%s?tab=abilities", id, caminho)
	return aRecusaDaCena(f.pede(t, f.jogador, http.MethodPost, alvo, corpo).Body.String())
}

// O ACERVO junta as cinco procedências, e SÓ o que o personagem tem.
func TestTheCollectionJoinsTheFiveOrigins(t *testing.T) {
	f, id := oBarbaro(t, 5)
	seedRaca(t, f.s, id, "Anão")
	comEscolhas(t, f, id, `["class.barbaro.golpe-poderoso"]`, `["origin-batedor-pericia-Sobrevivência"]`)

	// O RECORTE É DO PAINEL, e não da tela: o diálogo de escolher desenha TODOS
	// os benefícios e TODOS os poderes eletivos como opções, então procurar na
	// tela inteira acharia justamente o que a lista não deve mostrar — e o
	// guarda afirmaria o contrário do que mede.
	tela := oPainelDosPoderes(aTelaDosPoderes(t, f, id))
	for _, esperado := range []string{
		"Fúria",          // automática da classe, nível 1
		"Golpe Poderoso", // escolhida
		"Sobrevivência",  // benefício de origem escolhido
		"Bárbaro",        // o crachá da fonte, encurtado
	} {
		if !strings.Contains(tela, esperado) {
			t.Errorf("a tela não tem %q", esperado)
		}
	}
	// O QUE NÃO FOI ESCOLHIDO não aparece: a origem oferece cinco benefícios e o
	// personagem leva três, então listar todos daria como possuído o que ninguém
	// escolheu.
	//
	// A asserção é sobre o NOME e não sobre o id, e isso não é detalhe: a tela
	// escreve nomes, então um id aqui seria uma string que nunca poderia
	// aparecer — asserção que não pode falhar. Medido: com o filtro de escolha
	// REMOVIDO, a versão por id continuava verde.
	if strings.Contains(tela, "À Prova de Tudo") {
		t.Error("um benefício de origem não escolhido apareceu como possuído")
	}
	// E O QUE O NÍVEL AINDA NÃO DEU também não: a Fúria Raivosa é do 6º.
	if strings.Contains(tela, "Fúria Raivosa") {
		t.Error("um poder acima do nível apareceu como possuído")
	}
}

func comEscolhas(t *testing.T, f pilotoFixture, id int64, poderes, origem string) {
	t.Helper()
	var set setBuilder
	set.Add("classPowers = ?", poderes)
	set.Add("originChoices = ?", origem)
	if err := set.exec(context.Background(), f.s.db, "UPDATE characters", id); err != nil {
		t.Fatalf("semear as escolhas: %v", err)
	}
}

// AS AÇÕES vêm em cima e ordenadas; as passivas ficam na outra seção.
func TestActionsComeSortedAndPassivesComeApart(t *testing.T) {
	f, id := oBarbaro(t, 5)
	tela := aTelaDosPoderes(t, f, id)

	if !strings.Contains(tela, "Ações") || !strings.Contains(tela, "Passivas · mostrar") {
		t.Fatal("a tela não desenhou as duas seções")
	}
	// A FÚRIA é postura e vai para as ações; o Instinto Selvagem é passivo.
	acoes := oRecorteDasAcoes(tela)
	if !strings.Contains(acoes, "Ativar Fúria") {
		t.Error("a postura não está entre as ações")
	}
	if strings.Contains(acoes, "Instinto Selvagem") {
		t.Error("uma passiva foi para a seção de ações")
	}
}

// oPainelDosPoderes corta a LISTA, deixando de fora os diálogos que vêm depois
// dela — o de escolher poderes mostra o catálogo inteiro de opções.
func oPainelDosPoderes(tela string) string {
	// O CORTE é no ABRIR do primeiro diálogo, e não no primeiro `</section>`: as
	// duas seções da lista são `<section>` ANINHADAS, e cortar no primeiro
	// fechamento deixaria de fora justamente as passivas. Os diálogos começam
	// depois do painel, e todos são sobreposições de tela cheia.
	fim := strings.Index(tela, `class="fixed inset-0`)
	if fim < 0 {
		return tela
	}
	return tela[:fim]
}

// oRecorteDasAcoes corta a seção de Ações — procurar na tela inteira acharia a
// passiva na seção de baixo e o guarda diria o contrário do que mede.
func oRecorteDasAcoes(tela string) string {
	inicio := strings.Index(tela, ">Ações</h3>")
	if inicio < 0 {
		return ""
	}
	fim := strings.Index(tela[inicio:], "Passivas ·")
	if fim < 0 {
		return tela[inicio:]
	}
	return tela[inicio : inicio+fim]
}

// A SEÇÃO DE AÇÕES FICA MESMO VAZIA, e a frase depende de quem lê.
//
// Um arcanista de nível 20 tem ZERO ações ativáveis — 26 habilidades, todas
// passivas (medido na ALE-217). Sumir com a seção faria a tela mudar de forma
// por classe; e mandar às Magias quem não conjura seria mandá-lo a uma aba
// vazia.
func TestWithoutActionsTheScreenExplainsInsteadOfShowingAVoid(t *testing.T) {
	f, id := oCombatente(t)
	if tela := aTelaDosPoderes(t, f, id); !strings.Contains(tela, "Suas habilidades são passivas") {
		t.Error("o guerreiro sem ação não recebeu a frase que não manda a lugar nenhum")
	}

	fArcano, idArcano := oArcanista(t)
	if tela := aTelaDosPoderes(t, fArcano, idArcano); !strings.Contains(tela, "aba Magias") {
		t.Error("quem conjura não foi mandado para as Magias")
	}
}

// OS DEGRAUS da postura saem do nível NA CLASSE (p40).
func TestTheStanceStepsComeFromTheLevelInTheClass(t *testing.T) {
	// A Fúria abre o primeiro degrau no 5º e ganha outro a cada 5 níveis.
	semDegrau, id4 := oBarbaro(t, 4)
	if tela := aTelaDosPoderes(t, semDegrau, id4); !strings.Contains(tela, "Ativar 2 PM") {
		t.Error("no 4º nível a Fúria devia entrar num toque só, por 2 PM")
	}

	comDegrau, id10 := oBarbaro(t, 10)
	tela := aTelaDosPoderes(t, comDegrau, id10)
	if !strings.Contains(tela, "POSTURA · 2+ PM") {
		t.Error("a postura que escala não avisa o '+' no custo")
	}
	// Dois degraus no 10º: o primeiro no 5º, o segundo no 10º.
	if !strings.Contains(tela, "Math.min(2,") {
		t.Error("o contador não conhece o teto de dois degraus do 10º nível")
	}
}

// ENTRAR NA POSTURA cobra o PM dos degraus e registra o que foi pago.
func TestEnteringTheStanceChargesTheStepsAndRecordsThePayment(t *testing.T) {
	f, id := oBarbaro(t, 10)

	if recusa := oComandoDoPoder(t, f, id, "postura/furia/entra", `{"poderdegraus":2}`); recusa != "" {
		t.Fatalf("entrar foi recusado: %q", recusa)
	}
	// Base 2 + dois degraus de 1 PM = 4 PM sobre os 20 semeados.
	if pm := oPmDe(t, f, id); pm != 16 {
		t.Errorf("o PM ficou %d, quer 16 (20 − 2 de base − 2 degraus de 1)", pm)
	}
	posturas, err := f.s.queries.ListCharacterStances(context.Background(), id)
	if err != nil {
		t.Fatalf("ler as posturas: %v", err)
	}
	if len(posturas) != 1 || posturas[0].Pmpaid != 4 || posturas[0].Steps != 2 {
		t.Errorf("o pagamento gravado foi %+v, quer 4 PM em 2 degraus", posturas)
	}
	// E A TELA passa a oferecer o encerrar.
	if !strings.Contains(aTelaDosPoderes(t, f, id), "Encerrar Fúria") {
		t.Error("a postura em curso não oferece encerrar")
	}
}

// MAIS DEGRAUS DO QUE O NÍVEL DÁ é recusado.
func TestAStanceAboveTheStepCeilingIsRefused(t *testing.T) {
	f, id := oBarbaro(t, 5)

	recusa := oComandoDoPoder(t, f, id, "postura/furia/entra", `{"poderdegraus":3}`)
	if !strings.Contains(recusa, "1 degraus") {
		t.Errorf("a recusa não diz o teto: %q", recusa)
	}
	if pm := oPmDe(t, f, id); pm != 20 {
		t.Errorf("a recusa cobrou assim mesmo: sobrou %d", pm)
	}
}

// ENCERRAR não devolve PM — é o que a tabela de posturas existe para lembrar.
func TestEndingTheStanceGivesNoMpBack(t *testing.T) {
	f, id := oBarbaro(t, 5)
	if recusa := oComandoDoPoder(t, f, id, "postura/furia/entra", `{"poderdegraus":0}`); recusa != "" {
		t.Fatalf("entrar foi recusado: %q", recusa)
	}
	antes := oPmDe(t, f, id)

	alvo := fmt.Sprintf("/personagens/%d/efeitos/postura/furia?tab=abilities", id)
	if recusa := aRecusaDaCena(f.pede(t, f.jogador, http.MethodPost, alvo, "").Body.String()); recusa != "" {
		t.Fatalf("encerrar foi recusado: %q", recusa)
	}
	if depois := oPmDe(t, f, id); depois != antes {
		t.Errorf("encerrar devolveu PM: %d → %d", antes, depois)
	}
	posturas, err := f.s.queries.ListCharacterStances(context.Background(), id)
	if err != nil {
		t.Fatalf("ler as posturas: %v", err)
	}
	if len(posturas) != 0 {
		t.Errorf("a postura sobreviveu ao encerrar: %+v", posturas)
	}
}

// USAR cobra o PM e soma o uso; o segundo uso do "1/cena" é recusado.
func TestUsingChargesTheMpAndCountsTheUse(t *testing.T) {
	f, id := oBarbaro(t, 5)
	comEscolhas(t, f, id, `["class.barbaro.brado-assustador"]`, `[]`)

	if recusa := oComandoDoPoder(t, f, id, "usa/class.barbaro.brado-assustador", ""); recusa != "" {
		t.Fatalf("usar foi recusado: %q", recusa)
	}
	if pm := oPmDe(t, f, id); pm != 19 {
		t.Errorf("o PM ficou %d, quer 19 (20 − 1)", pm)
	}
	usos, err := f.s.queries.ListCharacterPowerUses(context.Background(), id)
	if err != nil {
		t.Fatalf("ler os usos: %v", err)
	}
	if len(usos) != 1 || usos[0].Scope != "scene" || usos[0].Used != 1 {
		t.Errorf("o uso gravado foi %+v, quer 1 na cena", usos)
	}

	// O SEGUNDO uso é barrado, e o PM não sai de novo.
	recusa := oComandoDoPoder(t, f, id, "usa/class.barbaro.brado-assustador", "")
	if !strings.Contains(recusa, "limite por cena") {
		t.Errorf("o segundo uso não foi barrado pelo limite: %q", recusa)
	}
	if pm := oPmDe(t, f, id); pm != 19 {
		t.Errorf("a recusa cobrou de novo: %d", pm)
	}
}

// O CUSTO VARIÁVEL NÃO SE USA PELA FICHA — quem sabe o total é a mesa.
//
// Medido na bancada: 33 das 411 ativações escrevem "variavel" no `pmCost`, e o
// campo tipado como `int` deixava ZERO no lugar sem estourar — a Paródia
// aparecia como "0 PM" com o botão ativo, e usá-la não cobrava nada.
func TestAVariableCostCannotBeSpentFromTheSheet(t *testing.T) {
	f, id := oBarbaro(t, 10)
	comEscolhas(t, f, id, `["class.barbaro.vigor-primal"]`, `[]`)

	tela := aTelaDosPoderes(t, f, id)
	// A CAIXA ALTA da tela é do CSS: o HTML escreve "PM variável" e o
	// `uppercase` do crachá é que a mostra gritada. Afirmar o que o navegador
	// pinta seria afirmar a folha de estilo.
	if !strings.Contains(tela, "PM variável") {
		t.Error("o custo variável não é anunciado na tela")
	}
	if recusa := oComandoDoPoder(t, f, id, "usa/class.barbaro.vigor-primal", ""); !strings.Contains(recusa, "variável") {
		t.Errorf("a ficha aceitou usar um poder de custo variável: %q", recusa)
	}
	if pm := oPmDe(t, f, id); pm != 20 {
		t.Errorf("a recusa cobrou assim mesmo: %d", pm)
	}
}

// A CONCESSÃO DA POSTURA vira efeito ao entrar e sai ao encerrar.
//
// A Alma de Bronze (p41) dá PV temporários de nível + Força "enquanto a Fúria
// durar". Deixá-los para trás daria PV que uma postura encerrada continua
// pagando.
func TestTheStanceGrantComesAndGoesWithIt(t *testing.T) {
	f, id := oBarbaro(t, 5)

	if recusa := oComandoDoPoder(t, f, id, "postura/furia/entra", `{"poderdegraus":0}`); recusa != "" {
		t.Fatalf("entrar foi recusado: %q", recusa)
	}
	efeitos := osEfeitosDe(t, f, id)
	if !efeitos["class.barbaro.alma-de-bronze"] {
		t.Fatalf("a Alma de Bronze não virou efeito ao entrar na Fúria: %v", efeitos)
	}

	alvo := fmt.Sprintf("/personagens/%d/efeitos/postura/furia?tab=abilities", id)
	f.pede(t, f.jogador, http.MethodPost, alvo, "")
	if efeitos := osEfeitosDe(t, f, id); efeitos["class.barbaro.alma-de-bronze"] {
		t.Error("a reserva de PV temporários sobreviveu ao fim da postura")
	}
}

func osEfeitosDe(t *testing.T, f pilotoFixture, id int64) map[string]bool {
	t.Helper()
	linhas, err := f.s.queries.ListActiveEffectsByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler os efeitos: %v", err)
	}
	fora := map[string]bool{}
	for _, l := range linhas {
		fora[l.Catalogid] = true
	}
	return fora
}

// A BUSCA achata as duas seções e ignora acento.
func TestThePowerSearchFoldsAndIgnoresAccents(t *testing.T) {
	f, id := oBarbaro(t, 5)

	tela := f.pede(t, f.jogador, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=abilities&poderbusca=furia", id), "").Body.String()

	if !strings.Contains(tela, "Fúria") {
		t.Error("a busca sem acento não achou a Fúria")
	}
	// COM BUSCA as seções somem: o resultado é uma lista só, por nome.
	if strings.Contains(tela, "Passivas · mostrar") {
		t.Error("a busca deixou as seções em pé")
	}
	if strings.Contains(tela, "Instinto Selvagem") {
		t.Error("a busca trouxe quem não casa com o termo")
	}
}
