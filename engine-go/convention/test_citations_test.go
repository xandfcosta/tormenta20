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
	"TestAAbaAindaNaoPortadaLevaParaAFichaAntiga":  true,
	"TestABaixaLimpaOOuvinte":                      true,
	"TestABaixaTiraOOuvinteDaFicha":                true,
	"TestAsFontesEmbutidasSaoAsMesmasDaSPA":        true,
	"TestCharacterWritesRejectNonOwner":            true,
	"TestGeneratedTypesAreCurrent":                 true,
	"TestReabrirTrocaACenaEGuardaAQueEstavaNaMesa": true,
	"TestReopeningKeepsVersionMovingForward":       true,
	"TestResolveConditionalDisplayEmpty":           true,
	"TestSignVerifyRoundtrip":                      true,
	"TestSocketOriginFollowsTheHttpPolicy":         true,
	"TestStatForAbsentTarget":                      true,
	"TestTodaAbaDaFichaEstaPortada":                true,

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
	// que a ALE-205 removeu e que o GLOSSARIO desmente com todas as letras. Um
	// teste que dirige uma porta morta não fica obsoleto junto com ela; ele passa
	// a afirmar o oposto do produto, e continua passando.
	//
	// O segundo prendia uma regra VIVA (a posse do lugar) na porta errada, e por
	// isso mudou de casa em vez de morrer.
	"TestSwitchingScenesArchivesTheOneOnTheTable":      true,
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
	declarados := map[string]bool{}
	arquivos := arquivosParaCitacao(t)
	for _, caminho := range arquivos {
		conteudo, err := os.ReadFile(caminho)
		if err != nil {
			t.Fatalf("ler %s: %v", caminho, err)
		}
		for _, achado := range declaracaoDeTeste.FindAllStringSubmatch(string(conteudo), -1) {
			declarados[achado[1]] = true
		}
	}

	medidas := 0
	for _, caminho := range arquivos {
		conteudo, err := os.ReadFile(caminho)
		if err != nil {
			t.Fatalf("ler %s: %v", caminho, err)
		}
		for numero, linha := range strings.Split(string(conteudo), "\n") {
			if strings.HasPrefix(linha, "func Test") {
				continue
			}
			for _, citado := range citacaoDeTeste.FindAllString(linha, -1) {
				medidas++
				if declarados[citado] || tombstones[citado] || familias[citado] {
					continue
				}
				t.Errorf("%s:%d cita %s, que não existe.\n"+
					"Se o teste foi RENOMEADO, a citação acompanha; se ele foi APAGADO de\n"+
					"propósito, declare o nome em `tombstones` — a lápide é boa, e o que ela\n"+
					"precisa é de ser um ato explícito.",
					caminho, numero+1, citado)
			}
		}
	}

	// O denominador. Sem ele, "nenhuma citação pendurada" e "o regex não casou
	// com nada" são a mesma linha verde. Eram 485 em setembro de 2026, e o piso
	// está longe dele de propósito: o que ele pega é a varredura QUEBRAR, não a
	// prosa encolher.
	if medidas < 300 {
		t.Fatalf("só %d citações lidas — o guarda ficou cego", medidas)
	}
}

func arquivosParaCitacao(t *testing.T) []string {
	t.Helper()
	var achados []string
	// O `e2e/` entra porque cada spec de Playwright se JUSTIFICA citando o teste
	// de Go que já cobre a parte barata — é a regra "e2e é o menor conjunto
	// possível" escrita caso a caso. Uma citação podre ali faz o próximo autor
	// procurar uma garantia que não existe e escrever um e2e a mais.
	for _, raiz := range []string{"..", "../..", "../../e2e"} {
		err := filepath.WalkDir(raiz, func(caminho string, entrada os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entrada.IsDir() {
				if entrada.Name() == "node_modules" || entrada.Name() == ".git" {
					return filepath.SkipDir
				}
				// A raiz do repositório entra só pelos `.md` dela: o `engine-go`
				// já foi varrido inteiro pela primeira raiz.
				if raiz == "../.." && caminho != raiz {
					return filepath.SkipDir
				}
				return nil
			}
			nome := entrada.Name()
			if strings.HasSuffix(nome, ".go") || strings.HasSuffix(nome, ".md") ||
				strings.HasSuffix(nome, ".ts") {
				achados = append(achados, caminho)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("varrer %s: %v", raiz, err)
		}
	}
	return achados
}
