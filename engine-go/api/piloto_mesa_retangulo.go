package api

import "fmt"

// O RETÂNGULO no tabuleiro (ALE-203, item 10 do dono).
//
// "Não temos ferramenta de seleção em área." A escolha do dono foi que o
// retângulo faz DUAS coisas, e quem decide é a ferramenta na mão: com um pincel
// ou a borracha ele ENCHE de terreno; com "mover" ele MARCA peças.
//
// # Por que `Shift` no pincel e arrasto puro no mover
//
// O arrasto já está ocupado, e de formas diferentes:
//
//   - com o PINCEL, arrastar pinta à mão livre (é o traço da fatia 3);
//   - com a MÃO, arrastar move a janela;
//   - com MOVER, arrastar em cima de uma peça move a peça — mas arrastar no
//     VAZIO não faz nada.
//
// Então marcar peças cabe no arrasto vazio de graça, e o retângulo de terreno
// precisa de um gesto a mais. `Shift` é o modificador de "retângulo" em todo
// editor de desenho, e ele não colide com nada aqui.
//
// # O DESENHO do laço mora em sinais
//
// Como a régua e o gabarito: um retângulo remendado dentro da região do mapa
// seria apagado pelo quadro seguinte do stream, no meio do gesto. Ele é um nó só,
// posicionado por uma expressão — e por isso não custa nó por casa.

// Os sinais do laço. `retangulando` é o modo em curso e o valor É o modo, como
// o `$ferramenta` e o `$pincelando`: vazio (parado), `terreno` ou `pecas`.
const (
	sinalDoRetangulo   = "retangulando"
	sinalDoRetanguloDe = "retangulode" // "x/y" do canto onde o dedo desceu
)

const (
	retanguloDeTerreno = "terreno"
	retanguloDePecas   = "pecas"
)

// osSinaisDoRetangulo entram na semente da Mesa.
//
// O canto de ORIGEM guarda-se como `"x/y"` pelo mesmo motivo do `$ultimacasa`: é
// o formato do CAMINHO, e converter na hora de montar a rota foi exatamente onde
// a vírgula produziu um 404 mudo.
var osSinaisDoRetangulo = fmt.Sprintf(
	"%s: '', %s: '', retangulodex: 0, retangulodey: 0, retanguloatex: 0, retanguloatey: 0",
	sinalDoRetangulo, sinalDoRetanguloDe)

// oRetanguloPega abre o laço no canto em que o dedo desceu.
//
// Guarda o canto DUAS vezes — como texto de caminho e como par de números — e
// isso não é redundância: o texto vai para a rota no fim, e os números desenham o
// laço a cada quadro. Derivar um do outro na expressão custaria um `split` por
// movimento do ponteiro.
func oRetanguloPega(modo string) string {
	return fmt.Sprintf(
		"evt.preventDefault(); const cx = %s, cy = %s; "+
			"$%s = %q; $%s = cx + '/' + cy; "+
			"$retangulodex = cx; $retangulodey = cy; $retanguloatex = cx; $retanguloatey = cy; "+
			"evt.currentTarget.setPointerCapture(evt.pointerId)",
		clicouEmX, clicouEmY, sinalDoRetangulo, modo, sinalDoRetanguloDe,
	)
}

// oRetanguloSegue leva o canto oposto atrás do dedo.
//
// NÃO fala com o servidor: o laço é geometria, e o resultado só é pedido quando o
// dedo solta. É a diferença para o pincel — lá cada casa cruzada é uma ação, aqui
// o gesto inteiro é UMA.
func oRetanguloSegue(modo string) string {
	return fmt.Sprintf(
		"$%s === %q && ($retanguloatex = %s, $retanguloatey = %s)",
		sinalDoRetangulo, modo, clicouEmX, clicouEmY,
	)
}

// oRetanguloDeTerrenoSolta fecha o laço e manda encher.
//
// O `$ferramenta` escolhe a rota: a borracha tem caminho sem espécie, que é o
// conserto que a fatia 1 fez e que não pode se perder aqui.
func oRetanguloDeTerrenoSolta(v tabuleiroView) string {
	return fmt.Sprintf(
		"if ($%s !== %q) return; const ate = $retanguloatex + '/' + $retanguloatey, de = $%s; "+
			"$%s = ''; "+
			"return $ferramenta === %q "+
			"? @post('/mesa/%d/%d/tabuleiro/terreno/limpar/retangulo/' + de + '/' + ate) "+
			": @post('/mesa/%d/%d/tabuleiro/terreno/' + $ferramenta + '/retangulo/' + de + '/' + ate)",
		sinalDoRetangulo, retanguloDeTerreno, sinalDoRetanguloDe,
		sinalDoRetangulo,
		FerramentaDaBorracha,
		v.CampaignID, v.SessionID,
		v.CampaignID, v.SessionID,
	)
}

// oLacoEstaAberto mostra o desenho do laço.
func oLacoEstaAberto() string {
	return fmt.Sprintf("$%s !== ''", sinalDoRetangulo)
}

// oEstiloDoLaco põe o retângulo na tela, em QUADRADOS do plano.
//
// `min` e `+1` porque o retângulo inclui as duas casas das pontas e o dedo pode
// arrastar para qualquer lado — a mesma regra que o `CasasDoRetangulo` aplica do
// lado do servidor, e é de propósito que as duas existam: esta desenha o que
// aquela vai fazer, e uma promessa que não bate com o resultado é pior que não
// desenhar nada.
const oEstiloDoLaco = "`left: ${Math.min($retangulodex, $retanguloatex) * $quadrado}px; " +
	"top: ${Math.min($retangulodey, $retanguloatey) * $quadrado}px; " +
	"width: ${(Math.abs($retanguloatex - $retangulodex) + 1) * $quadrado}px; " +
	"height: ${(Math.abs($retanguloatey - $retangulodey) + 1) * $quadrado}px`"

// sinalDoCliqueEngolido diz ao `click` que o gesto anterior foi um ARRASTO.
//
// Ele existe porque o navegador dispara `click` depois de um `pointerdown` +
// `pointerup` no mesmo elemento, INCLUSIVE quando o dedo andou entre os dois. Sem
// ele, terminar um laço em cima da camada de repouso também MOVERIA a peça da
// vez para onde o laço terminou — o mestre marca um grupo e a peça do turno anda
// junto, sem ninguém ter pedido.
const sinalDoCliqueEngolido = "engoleoclique"

// oRetanguloDePecasSolta fecha o laço e pergunta ao servidor quem ele pegou.
//
// Só PERGUNTA: marcar não muta a cena, e a resposta é do tamanho de um sinal.
//
// O laço que NÃO ANDOU (mesmo canto nas duas pontas) é um clique, não um laço, e
// segue o caminho do clique — é assim que a mesma camada serve aos dois gestos.
func oRetanguloDePecasSolta(v tabuleiroView) string {
	return fmt.Sprintf(
		"if ($%s !== %q) return; const ate = $retanguloatex + '/' + $retanguloatey, de = $%s; "+
			"$%s = ''; if (de === ate) return; $%s = true; "+
			"return @post('/mesa/%d/%d/tabuleiro/marcar-area/' + de + '/' + ate)",
		sinalDoRetangulo, retanguloDePecas, sinalDoRetanguloDe,
		sinalDoRetangulo, sinalDoCliqueEngolido,
		v.CampaignID, v.SessionID,
	)
}

// aPecaEstaMarcada é a pergunta que acende o anel, e ela é feita UMA VEZ POR
// PEÇA na cena.
//
// `,id,` com vírgulas nas pontas e não um `includes(id)` cru: sem elas, marcar a
// peça `abc` acenderia também a `abcd`. É a armadilha clássica de lista em
// string, e ela aparece exatamente no dia em que dois ids compartilham prefixo.
func aPecaEstaMarcada(id string) string {
	return fmt.Sprintf("(',' + $%s + ',').includes(%q)", sinalDasPecasMarcadas, ","+id+",")
}

// aFraseDoGrupo é o que a barra diz, com o plural certo.
//
// A frase INTEIRA numa expressão só, e não um número num `data-text` ao lado de
// um texto fixo: "1 peças marcadas" apareceu na tela na primeira medição, e a
// única forma de a palavra acompanhar o número é ela estar na mesma conta.
var aFraseDoGrupo = fmt.Sprintf(
	"(() => { const n = $%s.split(',').filter(Boolean).length; "+
		"return n + (n === 1 ? ' peça marcada' : ' peças marcadas') "+
		"+ ' · arraste qualquer uma para mover o grupo' })()",
	sinalDasPecasMarcadas)

// haGrupoMarcado mostra a barra do grupo.
var haGrupoMarcado = fmt.Sprintf("$%s !== ''", sinalDasPecasMarcadas)

// largaOGrupo desmarca tudo.
var largaOGrupo = fmt.Sprintf("$%s = ''", sinalDasPecasMarcadas)

// ── ARRASTAR O GRUPO ─────────────────────────────────────────────────────────
//
// Com peças marcadas, arrastar QUALQUER UMA delas move todas pelo mesmo delta.
// É o gesto que todo editor faz e o motivo inteiro de marcar: chegou uma horda
// de seis zumbis e reposicioná-los hoje custa seis arrastos.

// pegaOGrupo começa o arrasto, e SÓ se a peça estiver marcada.
//
// Sem a guarda, arrastar uma peça qualquer com um grupo marcado em outro canto
// do mapa moveria o grupo distante — o gesto agiria sobre o que a pessoa não
// está olhando, que é a pior classe de surpresa num tabuleiro.
func pegaOGrupo(id string) string {
	return fmt.Sprintf("if (!(%s)) return; %s", aPecaEstaMarcada(id), pegaParaArrastar("peca"))
}

// soltaOGrupo converte o deslocamento em QUADRADOS e move todas.
//
// O arredondamento é o mesmo do arrasto de uma peça só (para o quadrado mais
// próximo, e não para baixo), porque o gesto é o mesmo gesto — o que muda é
// quantas peças ele leva.
func soltaOGrupo(v tabuleiroView) string {
	return fmt.Sprintf(
		"if ($arrastando === 'peca') { "+
			"const dx = Math.round($arrastox / $quadrado), dy = Math.round($arrastoy / $quadrado); "+
			"$arrastando = ''; $arrastox = 0; $arrastoy = 0; "+
			"if (dx || dy) @post('/mesa/%d/%d/tabuleiro/grupo/mover/' + dx + '/' + dy) }",
		v.CampaignID, v.SessionID,
	)
}

// oVestidoDaPeca junta as duas marcas que a peça pode vestir.
//
// UM `data-class` só porque atributo repetido não existe: o navegador guarda o
// primeiro e descarta o segundo, e a marca do grupo nasceria morta — é a mesma
// armadilha do `data-on:keydown__window` duplicado que a fatia 2 registrou.
func oVestidoDaPeca(id string, arrastavel bool) string {
	marcada := fmt.Sprintf("'tabuleiro-peca-marcada': %s", aPecaEstaMarcada(id))
	if !arrastavel {
		return "{" + marcada + "}"
	}
	return fmt.Sprintf("{'tabuleiro-arrastando': $arrastando === 'peca', %s}", marcada)
}

// oGestoDoPincel decide entre TRAÇO e RETÂNGULO no `pointerdown`.
//
// A decisão é no `pointerdown` e vale para o gesto inteiro, como o modo do
// pincel: soltar o `Shift` no meio do arrasto não pode trocar o que o gesto está
// fazendo — o dedo já está a caminho de um canto.
//
// `if/else` e NÃO um ternário, e isto é conserto de um defeito MUDO. Os dois
// ramos são SEQUÊNCIAS DE COMANDOS (`preventDefault(); $sinal = …;
// setPointerCapture(…)`), e sequência de comandos entre parênteses é erro de
// SINTAXE em JavaScript. O Datastar engoliu o erro de parse e o `pointerdown`
// inteiro virou nada — não só o retângulo: o pincel à mão livre, que funcionava,
// parou junto. Medido: o `pointerdown` chegava ao elemento e nenhuma requisição
// saía, sem uma linha no console.
func oGestoDoPincel(v tabuleiroView, modoFixo string) string {
	return fmt.Sprintf("if (evt.shiftKey) { %s } else { %s }",
		oRetanguloPega(retanguloDeTerreno), oPincelPega(v, modoFixo))
}

// oNomeDaCamadaDeRepouso diz os dois gestos que ela aceita.
//
// O nome acessível é onde o gesto de ARRASTO fica descoberto: ele não tem ícone
// nem botão, e quem navega por teclado não tem outro lugar para achá-lo.
func oNomeDaCamadaDeRepouso(v tabuleiroView) string {
	if v.AlvoDoMovimento == "" {
		return "Marcar peças — arraste um retângulo em volta delas"
	}
	if !v.Mestre {
		return "Mover " + v.RotuloDoAlvo + " — escolha a casa"
	}
	return "Mover " + v.RotuloDoAlvo + " — escolha a casa, ou arraste para marcar um grupo"
}

// oCliqueEmRepouso é o clique da camada, com o ENGOLE na frente.
//
// Sem alvo de movimento não há o que o clique faça, e a expressão fica só com o
// engole — que continua precisando existir, porque o `click` vem do mesmo jeito
// depois de um laço.
func oCliqueEmRepouso(v tabuleiroView) string {
	engole := fmt.Sprintf("if ($%s) { $%s = false; return }", sinalDoCliqueEngolido, sinalDoCliqueEngolido)
	if v.AlvoDoMovimento == "" {
		return engole
	}
	return engole + "; " + paradaNoPontoClicado(v)
}
