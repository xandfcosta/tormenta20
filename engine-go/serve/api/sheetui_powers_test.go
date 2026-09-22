package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
	"testing"
)

func barbaro(t *testing.T, level int64) (sceneFixture, int64) {
	t.Helper()
	return barbarianWithMpSpent(t, level, 0)
}

// barbarianWithMpSpent semeia o mesmo bárbaro com um tanto de PM JÁ GASTO, para
// os casos em que o bolso vazio É o assunto. (O `barbaro` ao lado é dívida de
// idioma baselinada; nome NOVO sai em inglês.)
//
// Ele diz quanto foi GASTO e não quanto sobrou porque o poço é do catálogo: o
// máximo de um bárbaro de nível 5 é o que o livro dá, e uma bancada que o
// escolhesse estaria medindo um personagem que não pode existir (ALE-355).
func barbarianWithMpSpent(t *testing.T, level, pmSpent int64) (sceneFixture, int64) {
	t.Helper()
	f := newSceneFixture(t)
	id, err := f.s.sceneCore().Queries().CreateCharacter(context.Background(), sqlcgen.CreateCharacterParams{
		OwnerId: f.player, Name: "Furioso", Origin: "Batedor", Level: level,
		Strength: 4, Dexterity: 2, Constitution: 3, Intelligence: 0, Wisdom: 1, Charisma: 0,
		Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: "[]", ClassPowers: "[]", ClassChoices: "{}", PowerChoices: "{}",
		CreatedAt: dbvalue.NowISO(), UpdatedAt: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear o bárbaro: %v", err)
	}
	seedClasse(t, f.s, id, "Bárbaro", level)
	arrangePools(t, f.s, id, func(pools sheet.Pools) (sheet.Pools, error) {
		pools.HpCurrent, pools.MpCurrent = pools.HpMax, pools.MpMax-pmSpent
		return pools, nil
	})
	return f, id
}

func powerScreen(t *testing.T, f sceneFixture, id int64) string {
	t.Helper()
	return f.requests(t, f.player, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=abilities", id), "").Body.String()
}

func powerCommand(t *testing.T, f sceneFixture, id int64, path, body string) string {
	t.Helper()
	target := fmt.Sprintf("/personagens/%d/poderes/%s?tab=abilities", id, path)
	return sceneRefusal(f.requests(t, f.player, http.MethodPost, target, body).Body.String())
}

// O ACERVO junta as cinco procedências, e SÓ o que o personagem tem.
func TestTheCollectionJoinsTheFiveOrigins(t *testing.T) {
	f, id := barbaro(t, 5)
	seedRaca(t, f.s, id, "Anão")
	choiceCom(t, f, id, `["class.barbaro.golpe-poderoso"]`, `["origin-batedor-pericia-Sobrevivência"]`)

	// O RECORTE É DO PAINEL, e não da tela: o diálogo de escolher desenha TODOS
	// os benefícios e TODOS os poderes eletivos como opções, então procurar na
	// tela inteira acharia justamente o que a lista não deve mostrar — e o
	// guarda afirmaria o contrário do que mede.
	screen := powerPanel(powerScreen(t, f, id))
	for _, want := range []string{
		"Fúria",          // automática da classe, nível 1
		"Golpe Poderoso", // escolhida
		"Sobrevivência",  // benefício de origem escolhido
		"Bárbaro",        // o crachá da fonte, encurtado
	} {
		if !strings.Contains(screen, want) {
			t.Errorf("a tela não tem %q", want)
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
	if strings.Contains(screen, "À Prova de Tudo") {
		t.Error("um benefício de origem não escolhido apareceu como possuído")
	}
	// E O QUE O NÍVEL AINDA NÃO DEU também não: a Fúria Raivosa é do 6º.
	if strings.Contains(screen, "Fúria Raivosa") {
		t.Error("um poder acima do nível apareceu como possuído")
	}
}

func choiceCom(t *testing.T, f sceneFixture, id int64, powers, origin string) {
	t.Helper()
	if _, err := f.s.db.ExecContext(context.Background(),
		"UPDATE characters SET classPowers = ?, originChoices = ? WHERE id = ?",
		powers, origin, id,
	); err != nil {
		t.Fatalf("semear as escolhas: %v", err)
	}
}

// AS AÇÕES vêm em cima e ordenadas; as passivas ficam na outra seção.
func TestActionsComeSortedAndPassivesComeApart(t *testing.T) {
	f, id := barbaro(t, 5)
	screen := powerScreen(t, f, id)

	if !strings.Contains(screen, "Ações") || !strings.Contains(screen, "Passivas · mostrar") {
		t.Fatal("a tela não desenhou as duas seções")
	}
	// A FÚRIA é postura e vai para as ações; o Instinto Selvagem é passivo.
	actions := actionsSlice(screen)
	if !strings.Contains(actions, "Ativar Fúria") {
		t.Error("a postura não está entre as ações")
	}
	if strings.Contains(actions, "Instinto Selvagem") {
		t.Error("uma passiva foi para a seção de ações")
	}
}

// A SEÇÃO DE AÇÕES FICA MESMO VAZIA, e a frase depende de quem lê.
//
// Um arcanista de nível 20 tem ZERO ações ativáveis — 26 habilidades, todas
// passivas. Sumir com a seção faria a tela mudar de forma
// por classe; e mandar às Magias quem não conjura seria mandá-lo a uma aba
// vazia.
func TestWithoutActionsTheScreenExplainsInsteadOfShowingAVoid(t *testing.T) {
	f, id := fighterFixture(t)
	if screen := powerScreen(t, f, id); !strings.Contains(screen, "Suas habilidades são passivas") {
		t.Error("o guerreiro sem ação não recebeu a frase que não manda a lugar nenhum")
	}

	arcanistF, arcanistID := arcanista(t)
	if screen := powerScreen(t, arcanistF, arcanistID); !strings.Contains(screen, "aba Magias") {
		t.Error("quem conjura não foi mandado para as Magias")
	}
}

// OS DEGRAUS da postura saem do nível NA CLASSE (p40).
func TestTheStanceStepsComeFromTheLevelInTheClass(t *testing.T) {
	// A Fúria abre o primeiro degrau no 5º e ganha outro a cada 5 níveis.
	noStep, id4 := barbaro(t, 4)
	if screen := powerScreen(t, noStep, id4); !strings.Contains(screen, "Ativar 2 PM") {
		t.Error("no 4º nível a Fúria devia entrar num toque só, por 2 PM")
	}

	withStep, id10 := barbaro(t, 10)
	page := powerScreen(t, withStep, id10)
	if !strings.Contains(page, "POSTURA · 2+ PM") {
		t.Error("a postura que escala não avisa o '+' no custo")
	}
	// Dois degraus no 10º: o primeiro no 5º, o segundo no 10º.
	if !strings.Contains(page, "Math.min(2,") {
		t.Error("o contador não conhece o teto de dois degraus do 10º nível")
	}
}

// ENTRAR NA POSTURA cobra o PM dos degraus e registra o que foi pago.
func TestEnteringTheStanceChargesTheStepsAndRecordsThePayment(t *testing.T) {
	f, id := barbaro(t, 10)

	before := pm(t, f, id)
	if refusal := powerCommand(t, f, id, "postura/furia/entra", `{"stance_degrees":2}`); refusal != "" {
		t.Fatalf("entrar foi recusado: %q", refusal)
	}
	// O que se prende é o CUSTO e não o saldo: base 2 + dois degraus de 1 PM.
	// Em delta, porque o saldo de partida é o poço que o livro dá ao bárbaro de
	// nível 10, e não um número que a bancada escolha.
	if after := pm(t, f, id); before-after != 4 {
		t.Errorf("a entrada cobrou %d PM (%d → %d), quer 4: 2 de base + 2 degraus de 1",
			before-after, before, after)
	}
	stances, err := f.s.sceneCore().Queries().ListCharacterStances(context.Background(), id)
	if err != nil {
		t.Fatalf("ler as posturas: %v", err)
	}
	if len(stances) != 1 || stances[0].Pmpaid != 4 || stances[0].Steps != 2 {
		t.Errorf("o pagamento gravado foi %+v, quer 4 PM em 2 degraus", stances)
	}
	// E A TELA passa a oferecer o encerrar.
	if !strings.Contains(powerScreen(t, f, id), "Encerrar Fúria") {
		t.Error("a postura em curso não oferece encerrar")
	}
}

// MAIS DEGRAUS DO QUE O NÍVEL DÁ é recusado.
func TestAStanceAboveTheStepCeilingIsRefused(t *testing.T) {
	f, id := barbaro(t, 5)

	before := pm(t, f, id)
	refusal := powerCommand(t, f, id, "postura/furia/entra", `{"stance_degrees":3}`)
	if !strings.Contains(refusal, "1 degraus") {
		t.Errorf("a recusa não diz o teto: %q", refusal)
	}
	if after := pm(t, f, id); after != before {
		t.Errorf("a recusa cobrou assim mesmo: %d → %d", before, after)
	}
}

// SEM PM NO BOLSO, nem o poder sai nem a postura abre.
//
// # Este caso nasceu de uma SABOTAGEM, e vale dizer como
//
// A ALE-351 moveu as regras de ativação da cena para o `domain/book` e, para
// provar que os guardas existentes mediam o código que mudou de casa, sabotou
// cada ramo. O do teto de degraus reprovou; **o do PM não reprovou nada** —
// trocar `ActivationPm(spec) > contexto.PmAtual` por `false` deixava a suíte
// inteira verde.
//
// O buraco era de COBERTURA e não de comportamento: o servidor já recusava, e
// ninguém afirmava isso. Sem o caso, o dia em que a comparação inverter passa
// batido — e o `chargeMp` tem PISO EM ZERO, então o sintoma não seria um erro:
// seria o jogador usando o que não pode pagar e o PM indo a zero calado.
//
// As duas metades do mesmo buraco estão aqui, porque são dois caminhos
// diferentes até a mesma comparação: o `UseDecision` e o `StanceDecision`.
func TestWithoutMpNeitherThePowerNorTheStanceGoesThrough(t *testing.T) {
	f, id := barbarianWithMpSpent(t, 5, 15)
	choiceCom(t, f, id, `["class.barbaro.brado-assustador"]`, `[]`)

	use := powerCommand(t, f, id, "usa/class.barbaro.brado-assustador", "")
	if !strings.Contains(use, "PM insuficiente") {
		t.Errorf("o poder saiu com o bolso vazio: %q", use)
	}
	enter := powerCommand(t, f, id, "postura/furia/entra", `{"stance_degrees":0}`)
	if !strings.Contains(enter, "PM insuficiente") {
		t.Errorf("a postura abriu com o bolso vazio: %q", enter)
	}
	// NADA foi gravado por nenhum dos dois: sem esta metade, "a recusa apareceu"
	// não diz se ela apareceu ANTES ou DEPOIS da escrita.
	if pm := pm(t, f, id); pm != 0 {
		t.Errorf("o PM saiu do zero: %d", pm)
	}
	uses, err := f.s.sceneCore().Queries().ListCharacterPowerUses(context.Background(), id)
	if err != nil {
		t.Fatalf("ler os usos: %v", err)
	}
	if len(uses) != 0 {
		t.Errorf("a recusa somou um uso assim mesmo: %+v", uses)
	}
	stances, err := f.s.sceneCore().Queries().ListCharacterStances(context.Background(), id)
	if err != nil {
		t.Fatalf("ler as posturas: %v", err)
	}
	if len(stances) != 0 {
		t.Errorf("a recusa abriu a postura assim mesmo: %+v", stances)
	}
}

// ENCERRAR não devolve PM — é o que a tabela de posturas existe para lembrar.
func TestEndingTheStanceGivesNoMpBack(t *testing.T) {
	f, id := barbaro(t, 5)
	if refusal := powerCommand(t, f, id, "postura/furia/entra", `{"stance_degrees":0}`); refusal != "" {
		t.Fatalf("entrar foi recusado: %q", refusal)
	}
	before := pm(t, f, id)

	target := fmt.Sprintf("/personagens/%d/efeitos/postura/furia?tab=abilities", id)
	if refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, target, "").Body.String()); refusal != "" {
		t.Fatalf("encerrar foi recusado: %q", refusal)
	}
	if after := pm(t, f, id); after != before {
		t.Errorf("encerrar devolveu PM: %d → %d", before, after)
	}
	stances, err := f.s.sceneCore().Queries().ListCharacterStances(context.Background(), id)
	if err != nil {
		t.Fatalf("ler as posturas: %v", err)
	}
	if len(stances) != 0 {
		t.Errorf("a postura sobreviveu ao encerrar: %+v", stances)
	}
}

// USAR cobra o PM e soma o uso; o segundo uso do "1/cena" é recusado.
func TestUsingChargesTheMpAndCountsTheUse(t *testing.T) {
	f, id := barbaro(t, 5)
	choiceCom(t, f, id, `["class.barbaro.brado-assustador"]`, `[]`)

	before := pm(t, f, id)
	if refusal := powerCommand(t, f, id, "usa/class.barbaro.brado-assustador", ""); refusal != "" {
		t.Fatalf("usar foi recusado: %q", refusal)
	}
	after := pm(t, f, id)
	if before-after != 1 {
		t.Errorf("o uso cobrou %d PM (%d → %d), quer 1", before-after, before, after)
	}
	uses, err := f.s.sceneCore().Queries().ListCharacterPowerUses(context.Background(), id)
	if err != nil {
		t.Fatalf("ler os usos: %v", err)
	}
	if len(uses) != 1 || uses[0].Scope != "scene" || uses[0].Used != 1 {
		t.Errorf("o uso gravado foi %+v, quer 1 na cena", uses)
	}

	// O SEGUNDO uso é barrado, e o PM não sai de novo.
	refused := powerCommand(t, f, id, "usa/class.barbaro.brado-assustador", "")
	if !strings.Contains(refused, "limite por cena") {
		t.Errorf("o segundo uso não foi barrado pelo limite: %q", refused)
	}
	if other := pm(t, f, id); other != after {
		t.Errorf("a recusa cobrou de novo: %d → %d", after, other)
	}
}

// O CUSTO VARIÁVEL NÃO SE USA PELA FICHA — quem sabe o total é a mesa.
//
// Medido na bancada: 33 das 411 ativações escrevem "variavel" no `pmCost`, e o
// campo tipado como `int` deixava ZERO no lugar sem estourar — a Paródia
// aparecia como "0 PM" com o botão ativo, e usá-la não cobrava nada.
func TestAVariableCostCannotBeSpentFromTheSheet(t *testing.T) {
	f, id := barbaro(t, 10)
	choiceCom(t, f, id, `["class.barbaro.vigor-primal"]`, `[]`)

	screen := powerScreen(t, f, id)
	// A CAIXA ALTA da tela é do CSS: o HTML escreve "PM variável" e o
	// `uppercase` do crachá é que a mostra gritada. Afirmar o que o navegador
	// pinta seria afirmar a folha de estilo.
	if !strings.Contains(screen, "PM variável") {
		t.Error("o custo variável não é anunciado na tela")
	}
	before := pm(t, f, id)
	if refusal := powerCommand(t, f, id, "usa/class.barbaro.vigor-primal", ""); !strings.Contains(refusal, "variável") {
		t.Errorf("a ficha aceitou usar um poder de custo variável: %q", refusal)
	}
	if after := pm(t, f, id); after != before {
		t.Errorf("a recusa cobrou assim mesmo: %d → %d", before, after)
	}
}

// A CONCESSÃO DA POSTURA vira efeito ao entrar e sai ao encerrar.
//
// A Alma de Bronze (p41) dá PV temporários de nível + Força "enquanto a Fúria
// durar". Deixá-los para trás daria PV que uma postura encerrada continua
// pagando.
func TestTheStanceGrantComesAndGoesWithIt(t *testing.T) {
	f, id := barbaro(t, 5)

	if refusal := powerCommand(t, f, id, "postura/furia/entra", `{"stance_degrees":0}`); refusal != "" {
		t.Fatalf("entrar foi recusado: %q", refusal)
	}
	effectList := effects(t, f, id)
	if !effectList["class.barbaro.alma-de-bronze"] {
		t.Fatalf("a Alma de Bronze não virou efeito ao entrar na Fúria: %v", effectList)
	}

	target := fmt.Sprintf("/personagens/%d/efeitos/postura/furia?tab=abilities", id)
	f.requests(t, f.player, http.MethodPost, target, "")
	if gotEffects := effects(t, f, id); gotEffects["class.barbaro.alma-de-bronze"] {
		t.Error("a reserva de PV temporários sobreviveu ao fim da postura")
	}
}

func effects(t *testing.T, f sceneFixture, id int64) map[string]bool {
	t.Helper()
	rows, err := f.s.sceneCore().Queries().ListActiveEffectsByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler os efeitos: %v", err)
	}
	outside := map[string]bool{}
	for _, l := range rows {
		outside[l.Catalogid] = true
	}
	return outside
}

// A BUSCA achata as duas seções e ignora acento.
func TestThePowerSearchFoldsAndIgnoresAccents(t *testing.T) {
	f, id := barbaro(t, 5)

	screen := f.requests(t, f.player, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=abilities&poderbusca=furia", id), "").Body.String()

	if !strings.Contains(screen, "Fúria") {
		t.Error("a busca sem acento não achou a Fúria")
	}
	// COM BUSCA as seções somem: o resultado é uma lista só, por nome.
	if strings.Contains(screen, "Passivas · mostrar") {
		t.Error("a busca deixou as seções em pé")
	}
	if strings.Contains(screen, "Instinto Selvagem") {
		t.Error("a busca trouxe quem não casa com o termo")
	}
}
