package convention

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// OUVINTE DE JANELA NÃO SE PENDURA DENTRO DE UM LAÇO (ALE-298).
//
// # O defeito, medido
//
// A aba de Perícias servia **trinta** ouvintes de `keydown__window` com Escape.
// Vinte e nove escreviam a MESMA coisa — `$detail = ”` —, porque a moldura de
// diálogo é desenhada por item da lista e cada instância pendurava o próprio.
//
// Um ouvinte de janela não pertence ao nó que o pendura: ele não sabe qual
// diálogo está por cima e roda de qualquer jeito. Vinte e nove cópias fazem o
// mesmo trabalho vinte e nove vezes, e o navegador guarda vinte e nove
// registros para uma tecla que uma linha resolve.
//
// # Ele cobra TECLA, e o ponteiro fica de fora com a razão medida
//
// A varredura pega `keydown__window` e irmãos, não `pointermove__window`. Não é
// recorte de conveniência: **uma tecla não tem alvo**. Ela chega à janela, e um
// ouvinte de tecla pertence à CENA por natureza — foi por isso que mover o da
// ficha para cima resolveu sem mudar comportamento nenhum.
//
// O arrasto é o contrário. A peça do tabuleiro pendura `pointermove__window` e
// `pointerup__window` por peça, e a expressão de cada uma CARREGA a peça: o
// gesto é sobre um elemento específico e o ouvinte é de janela só porque o
// ponteiro sai de cima dele. Juntá-los num só exige descobrir qual peça a partir
// de um sinal, e isso é refatoração com risco de comportamento — está medido na
// ALE-299 em vez de virar exceção declarada aqui.
//
// **A ALE-299 rodou, e a resposta foi que o ponteiro CONTINUA fora — por outro
// motivo.** O que ela achou não foi custo, foi CORREÇÃO: o sinal que separava um
// arrasto do outro guardava um literal compartilhado, e no rascunho pegar uma
// peça movia a primeira do DOM. Consertada a identidade, o custo foi medido com
// nove peças — nove ouvintes de `pointermove`, 10,5ms de handler em 60 quadros,
// 0,175ms por quadro contra um orçamento de 16,7ms. Juntar tudo num ouvinte
// devolveria 0,156ms por quadro no gesto mais medido do app. Não paga.
//
// Quem cobra o gesto por peça agora é o `TestNoTokenGestureAnswersForAnotherToken`,
// no `web/table`: ele não conta ouvintes, ele exige que cada um reconheça a
// PRÓPRIA peça.
//
// # O que este guarda cobra, e o que ele NÃO cobra
//
// Ele lê a FONTE e recusa um COMPONENTE que pendure ouvinte de tecla na janela
// sendo chamado de dentro de um bloco `for`.
// É a forma que produz o defeito, e ela é a única mecanizável: contar ouvintes
// na página servida daria um número diferente por dado semeado, e um piso
// escrito à mão sobre isso envelheceria na primeira criatura nova da seed.
//
// Ele NÃO diz que uma cena só pode ter um ouvinte de janela. Ter um por diálogo
// é o desenho certo — o que não pode é ter um por LINHA de uma lista.
//
// # Por que a fonte e não o HTML
//
// Porque a causa é estrutural e a fonte a mostra. No HTML servido as trinta
// cópias são indistinguíveis de trinta diálogos legítimos, e separar as duas
// coisas exigiria adivinhar quantos itens a lista tinha.
func TestNoWindowListenerIsHungInsideALoop(t *testing.T) {
	window := regexp.MustCompile(`data-on:key[a-z]+__window`)
	declares := regexp.MustCompile(`^templ\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	called := regexp.MustCompile(`@([A-Za-z_][A-Za-z0-9_]*)\(`)

	// PRIMEIRA PASSADA: quais componentes penduram ouvinte de janela.
	withListener := map[string]string{}
	var filesRead, listeners int
	read := func(pass func(name string, rows []string)) error {
		return filepath.WalkDir("..", func(name string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if entry.Name() == "node_modules" || entry.Name() == ".git" {
					return fs.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(name, ".templ") {
				return nil
			}
			raw, err := os.ReadFile(name)
			if err != nil {
				return err
			}
			pass(name, semComentario(strings.Split(string(raw), "\n")))
			return nil
		})
	}

	err := read(func(name string, rows []string) {
		filesRead++
		current := ""
		for _, row := range rows {
			if m := declares.FindStringSubmatch(row); m != nil {
				current = m[1]
			}
			if window.MatchString(row) {
				listeners++
				if current != "" {
					withListener[current] = name
				}
			}
		}
	})
	if err != nil {
		t.Fatalf("caminhar a árvore: %v", err)
	}

	// FECHO TRANSITIVO: quem CHAMA quem pendura, também pendura.
	//
	// Sem isto o guarda passa verde sobre o defeito que o originou, e eu descobri
	// isso do jeito certo — rodando-o contra a árvore de ONTEM. A cadeia real
	// era `for` → `@expertiseDetail` → `@overlay`, e só o último tinha o
	// atributo. Um guarda que olha UM nível mede a folha e ignora o galho.
	caller := map[string][]string{}
	err = read(func(name string, rows []string) {
		current := ""
		for _, row := range rows {
			if m := declares.FindStringSubmatch(row); m != nil {
				current = m[1]
				continue
			}
			if current == "" {
				continue
			}
			for _, c := range called.FindAllStringSubmatch(row, -1) {
				caller[current] = append(caller[current], c[1])
			}
		}
	})
	if err != nil {
		t.Fatalf("caminhar a árvore: %v", err)
	}
	for changed := true; changed; {
		changed = false
		for parent, children := range caller {
			if _, alreadyHas := withListener[parent]; alreadyHas {
				continue
			}
			for _, child := range children {
				if where, hasListener := withListener[child]; hasListener {
					withListener[parent] = where + " (via @" + child + ")"
					changed = true
					break
				}
			}
		}
	}

	// SEGUNDA PASSADA: algum deles é CHAMADO de dentro de um laço?
	//
	// É aqui que o defeito mora, e é por isso que o guarda precisa das duas: a
	// moldura do diálogo e o `for` que a repete estão em ARQUIVOS DIFERENTES.
	// Uma varredura de arquivo único passa verde sobre trinta ouvintes — foi o
	// que a primeira versão deste guarda fez, e ela só foi desmentida porque a
	// árvore de ontem passou nela.
	var flagged int
	err = read(func(name string, rows []string) {
		depth, inside := 0, 0
		for i, row := range rows {
			cut := strings.TrimSpace(row)
			if strings.HasPrefix(cut, "for ") && strings.HasSuffix(cut, "{") {
				inside++
				depth++
				continue
			}
			if depth > 0 {
				depth += strings.Count(row, "{") - strings.Count(row, "}")
				if depth <= 0 {
					depth, inside = 0, 0
				}
			}
			if inside == 0 {
				continue
			}
			for component, where := range withListener {
				if !strings.Contains(cut, "@"+component+"(") {
					continue
				}
				flagged++
				t.Errorf("%s:%d — `@%s` é chamado de dentro de um `for`, e ele pendura um ouvinte de JANELA (%s).\n"+
					"    Ele será registrado uma vez POR ITEM, e um ouvinte de janela roda de qualquer jeito —\n"+
					"    a aba de Perícias chegou a servir trinta deles para uma tecla. Pendure-o na CENA.",
					name, i+1, component, where)
			}
		}
	})
	if err != nil {
		t.Fatalf("caminhar a árvore: %v", err)
	}
	_ = flagged

	// O DENOMINADOR, em três metades.
	// O contador é da PRIMEIRA passada — a segunda relê os mesmos arquivos sem
	// somar, e um piso de 80 aqui era a minha própria conta errada de quantas
	// vezes ele incrementa.
	if filesRead < 40 {
		t.Fatalf("o guarda leu só %d arquivos `.templ`: a caminhada parou de achar as cenas", filesRead)
	}
	// Se o padrão do ATRIBUTO parar de casar, a primeira passada volta vazia e a
	// segunda não tem o que procurar — verde idêntico ao de "está tudo certo".
	if listeners < 5 {
		t.Fatalf("o guarda achou só %d ouvintes de janela na árvore: o padrão do atributo parou de casar", listeners)
	}
	// E se a DECLARAÇÃO parar de casar, a lista de componentes fica vazia pelo
	// mesmo efeito. Os que existem são a prova de que ela casa.
	if len(withListener) < 3 {
		t.Fatalf("só %d componentes penduram ouvinte de janela: o padrão de `templ Nome(` parou de casar", len(withListener))
	}
}
