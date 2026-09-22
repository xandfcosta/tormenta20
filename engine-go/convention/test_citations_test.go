package convention

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// tombstones são os nomes de teste que a prosa cita DE PROPÓSITO sem eles
// existirem: `// Aqui morava o TestX, que prendia…`.
//
// A lápide é uma prática deste repositório e ela é boa — ela diz por que uma
// garantia SAIU, que é a informação que o `git log` esconde de quem lê o arquivo.
// Ela só precisa ser declarada, e é essa a fricção que esta lista existe para
// criar: apagar um teste é um ato, e o ato aparece aqui.
var tombstones = map[string]bool{
	// Os três mediam a marca `Dirty` do tabuleiro e a tarja "a mesa não está
	// sendo salva" que ela acendia. A ALE-375 pôs a gravação dentro da mutação:
	// a marca, a tarja e a pergunta que elas respondiam deixaram de existir, e
	// o que protegiam — a mesa não roda de memória em silêncio — passou a ser
	// prendido pela RECUSA, no `TestABoardWriteRefusedLeavesTheMapUntouched` e
	// no `TestACommandTheDiskRefusesComesBackRefused`.
	"TestBoardPersistFailureIsReported":            true,
	"TestTheGmIsWarnedWhenSavingFails":             true,
	"TestThePlayerIsNotWarnedAboutSaving":          true,
	"TestAAbaAindaNaoPortadaLevaParaAFichaAntiga":  true,
	"TestABaixaLimpaOOuvinte":                      true,
	"TestABaixaTiraOOuvinteDaFicha":                true,
	"TestAsFontesEmbutidasSaoAsMesmasDaSPA":        true,
	"TestCharacterWritesRejectNonOwner":            true,
	"TestGeneratedTypesAreCurrent":                 true,
	"TestHpToneAtTheThresholds":                    true,
	"TestReabrirTrocaACenaEGuardaAQueEstavaNaMesa": true,
	"TestReopeningKeepsVersionMovingForward":       true,
	"TestResolveConditionalDisplayEmpty":           true,
	"TestSignVerifyRoundtrip":                      true,
	"TestSocketOriginFollowsTheHttpPolicy":         true,
	"TestStatForAbsentTarget":                      true,
	"TestTodaAbaDaFichaEstaPortada":                true,

	// O TETO DO RETÂNGULO SAIU (ALE-315), por decisão do dono: o app roda LOCAL
	// e o limite de mil casas mordia gesto de verdade — no zoom mínimo o
	// tabuleiro visível tem 68×29 = 1.972 casas, e "pinte tudo o que estou
	// vendo" era recusado. A garantia não sumiu, ela INVERTEU: o
	// `TestTheWholeViewportFitsInOneRectangle` prende que o gesto grande chega
	// inteiro. O irmão TRAÇO mantém o teto dele, e por outro motivo — cem casas
	// num quadro de 16ms continuam impossíveis para um dedo.
	"TestAForgedRectangleIsRefused":           true,
	"TestAForgedRectangleIsRefusedByTheRoute": true,

	// AS ROTAS JSON SEM CONSUMIDOR (ALE-277). Dezessete casos que dirigiam rotas
	// que a ALE-277 apagou. Nenhum deles some sem substituto: a REGRA que cada
	// um prendia foi repontada para a função que a possui, ou já estava presa na
	// cena que hoje faz o gesto — e a lápide, no arquivo de origem, nomeia qual
	// das duas coisas aconteceu com cada garantia.
	"TestANormalBodyStillPasses":                          true,
	"TestEveryCharacterRouteIsCoveredByTheOwnershipTable": true,
	"TestEveryCharacterRouteRejectsAnIntruder":            true,
	"TestAResetRefusesAWeakPassword":                      true,
	"TestAdminScreenRoutesRejectEveryoneElse":             true,
	"TestAdminStatusReportsTheRunningServer":              true,
	"TestAdminUserListCountsWhatEachAccountOwns":          true,
	"TestAnOversizedBodyIsRefusedBySize":                  true,
	"TestCampaignDescriptionBlankIsTheSameEitherWay":      true,
	"TestCampaignWritesRejectNonOwner":                    true,
	"TestConsumeRejectsAStranger":                         true,
	"TestGetCampaignAuthorization":                        true,
	"TestMeCarriesTheAdminFlag":                           true,
	"TestOnlyAnAdminIssuesInvites":                        true,
	"TestSessionRoutesRejectCrossCampaignAndNonOwner":     true,
	"TestTheAdminCannotDeleteThemselves":                  true,
	"TestTheCampaignDetailLoadsTheRules":                  true,
	"TestTheJsonApiRefusesAChoiceOutsideTheRule":          true,
	"TestUpdateMemberRole":                                true,

	// A PORTA QUE A ALE-205 APOSENTOU (ALE-289). Os dois dirigiam o `ShowPlace`,
	// que nenhuma rota chamava havia três fatias.
	//
	// O primeiro é a razão de a varredura ter acontecido: ele afirmava, em
	// verde, que trocar de cena ARQUIVA a que estava na mesa — o comportamento
	// que a ALE-205 removeu e que o GLOSSARY desmente com todas as letras. Um
	// teste que dirige uma porta morta não fica obsoleto junto com ela; ele passa
	// a afirmar o oposto do produto, e continua passando.
	//
	// O segundo prendia uma regra VIVA (a posse do lugar) na porta errada, e por
	// isso mudou de casa em vez de morrer.
	"TestSwitchingScenesArchivesTheOneOnTheTable": true,

	// A REGRA QUE SAIU COM A CAUSA (ALE-291). Ele prendia o `nextFreeSpot`, que
	// foi apagado junto: três
	// peças avulsas criadas seguidas não podiam nascer na mesma casa (ALE-166).
	// A regra existia porque o "+ Peça" não tinha ONDE pôr a peça; o gesto que
	// chegou posiciona, e o guarda que o substitui afirma coisa mais forte — não
	// que duas não se empilhem, mas que cada uma nasce exatamente onde o mestre
	// clicou.
	"TestLoosePiecesDoNotStack":                        true,
	"TestASceneFromAnotherCampaignCannotReachTheTable": true,
}

// familias são os PREFIXOS que a prosa usa para falar de um conjunto de guardas
// (`TestEvery…`, `TestNo…`), e o `TestMain`, que existe em mais de um pacote.
var familias = map[string]bool{
	"TestEvery": true, "TestNo": true, "TestMain": true, "TestServer": true,
}

var citacaoDeTeste = regexp.MustCompile(`\bTest[A-Z]\w+`)

// NENHUMA CITAÇÃO NOMEIA UM TESTE QUE NÃO EXISTE (ALE-282).
//
// Um `.md` ou um comentário fica errado sem ninguém mexer nele: renomear um teste
// deixa a explicação dele falando de um nome que não existe mais. Não aparece no
// diff do código, não quebra compilação, e nenhum outro teste pega.
//
// # O número que justifica o guarda
//
// A varredura mediu 136 citações penduradas ANTES dela. 97 eram a mesma forma: o
// comentário grita uma palavra do próprio nome (`…ContadorTemQUATROEstados`),
// e por isso o renomeador casa sem olhar a caixa. Outras 23 eram nome velho ou
// partido no meio, e foram conferidas uma a uma contra a função que cada uma
// encabeça. As 13 que sobraram são lápides, e estão declaradas acima.
//
// Ou seja: 120 defeitos de documentação viviam neste repositório sem que nada os
// acusasse. Este guarda é o que impede o 121º.
func TestNoCitationNamesAMissingTest(t *testing.T) {
	declared := map[string]bool{}
	files := arquivosParaCitacao(t)
	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ler %s: %v", path, err)
		}
		for _, found := range declaracaoDeTeste.FindAllStringSubmatch(string(content), -1) {
			declared[found[1]] = true
		}
	}

	measured := 0
	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ler %s: %v", path, err)
		}
		for number, row := range strings.Split(string(content), "\n") {
			if strings.HasPrefix(row, "func Test") {
				continue
			}
			for _, cited := range citacaoDeTeste.FindAllString(row, -1) {
				measured++
				if declared[cited] || tombstones[cited] || familias[cited] {
					continue
				}
				t.Errorf("%s:%d cita %s, que não existe.\n"+
					"Se o teste foi RENOMEADO, a citação acompanha; se ele foi APAGADO de\n"+
					"propósito, declare o nome em `tombstones` — a lápide é boa, e o que ela\n"+
					"precisa é de ser um ato explícito.",
					path, number+1, cited)
			}
		}
	}

	// O denominador. Sem ele, "nenhuma citação pendurada" e "o regex não casou
	// com nada" são a mesma linha verde.
	//
	// O PISO DESCEU DE 300 PARA 150, e o motivo é o que ele mesmo dizia existir
	// para tolerar: a prosa encolheu. A poda de comentários levou as citações de
	// 485 para 264, e um piso de 300 passou a reprovar exatamente o caso que ele
	// NÃO quer pegar. Medido dos dois lados — 264 hoje, ZERO se o regex parar de
	// casar —, e 150 fica longe das duas pontas.
	if measured < 150 {
		t.Fatalf("só %d citações lidas — o guarda ficou cego", measured)
	}
}

func arquivosParaCitacao(t *testing.T) []string {
	t.Helper()
	var findings []string
	// O `e2e/` entra porque cada spec de Playwright se JUSTIFICA citando o teste
	// de Go que já cobre a parte barata — é a regra "e2e é o menor conjunto
	// possível" escrita caso a caso. Uma citação podre ali faz o próximo autor
	// procurar uma garantia que não existe e escrever um e2e a mais.
	for _, root := range []string{"..", "../..", "../../e2e"} {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if entry.Name() == "node_modules" || entry.Name() == ".git" {
					return filepath.SkipDir
				}
				// A raiz do repositório entra só pelos `.md` dela: o `engine-go`
				// já foi varrido inteiro pela primeira raiz.
				if root == "../.." && path != root {
					return filepath.SkipDir
				}
				return nil
			}
			name := entry.Name()
			if strings.HasSuffix(name, ".go") || strings.HasSuffix(name, ".md") ||
				strings.HasSuffix(name, ".ts") {
				findings = append(findings, path)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("varrer %s: %v", root, err)
		}
	}
	return findings
}
