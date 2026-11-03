# Testes

Como testar neste repositório, e como um instrumento mente. Faz parte do [CLAUDE.md](../CLAUDE.md).

**O objetivo é confiança de que o app produz o resultado que a gente quer — não
cobertura de cada pedacinho de código.** Um teste ganha o lugar dele protegendo
um resultado que alguém notaria quebrar. Tudo abaixo decorre disso.

- **Prefira INTEGRAÇÃO.** O teste padrão monta uma página de verdade (ou bate num
  handler de verdade, pelo roteador de verdade) com o I/O trocado na borda, e
  afirma o que a pessoa ou o chamador recebe. É a faixa que pega defeito de
  COMPOSIÇÃO — que é onde os defeitos deste repositório de fato estiveram — e é
  para lá que a cobertura nova vai primeiro.
- **Unitário para o que carrega REGRA**, não para o que carrega encanamento.
  Empilhamento de modificador, PV/PM, arredondamento, limites, rollback otimista,
  formato de fio: sim. Getter, formatador de uma linha, um `Set` + `sort` que a
  asserção reimplementa, e tudo que o typechecker já garante: não.
- **E2E é o menor conjunto possível.** Um teste de Playwright precisa se
  justificar com um mecanismo que só um navegador real tem — linha do tempo de
  animação, leiaute e overflow reais, lista virtualizada que mede zero no jsdom,
  gesto de ponteiro com passos intermediários, fluxo ao vivo entre dois
  servidores. **"É uma jornada do usuário" NÃO é justificativa**: jornada é mais
  barata e mais firme como teste de integração. E2E é a coisa mais cara e mais
  frágil deste repositório; gaste com intenção.
- **Empurre cada garantia para a camada mais barata que a segura.** Regra de
  servidor pertence a um teste de handler, não a uma asserção de que o botão
  sumiu — travar na UI é UX, a fronteira de segurança é o servidor.
- **Uma regra, uma camada.** Uma regra é prendida UMA vez, onde ela mora; as
  outras camadas afirmam presença e ligação, nunca a fronteira de novo. **Se
  apagar um teste não muda nada que outra camada já não acuse, apague.**
- **Correção de defeito ganha teste de regressão, e ele nasce VERMELHO.** Um
  teste que nunca foi visto falhando é um palpite. Quando o conserto e o teste
  entram juntos, o commit diz como o teste foi provado falho.
- **Nunca derive o esperado do código sob teste.** Não importe o helper que está
  sendo testado para montar o valor que se espera dele, e não espelhe a
  implementação na asserção — os dois andam juntos com o defeito. Escreva o
  número e a data na mão. Medido: um teste de componente que não passava o
  tamanho comparava o valor do servidor com o **default** da SPA, e passava verde
  sobre nada.
- **Unitário prova DECISÃO, não consulta.** Um spec que troca a borda protege
  exatamente duas coisas: a ramificação em volta da chamada e os argumentos dela.
  Um spec que não afirma nenhuma das duas é um *mock echo* — arranje o dublê para
  devolver X, afirme que o resultado é X — e não deve ser escrito. Ele prova que
  a linguagem devolve valores.
- **Quando o PREDICADO decide quem é afetado, prenda o predicado.** O `where` de
  uma consulta, o papel num `BoardForRole`, o filtro que escolhe QUAIS peças a
  mesa vê: isso não é encanamento, é a regra. Arranjar o resultado por ordem de
  chamada diz o que acontece *com* as linhas achadas e nada sobre *quais* linhas
  são essas.

## O INSTRUMENTO MENTE COM CARA DE RESULTADO

A infraestrutura em volta da medição destrói a medição, e o que sobra parece um
dado. **A pergunta que abre todas: "o que este instrumento DESLIGA para
funcionar, e o que ele não mede?"** Os casos que produziram cada linha estão no
[APRENDIZADOS.md](../APRENDIZADOS.md).

- **Limpeza não fala mais alto que o defeito**: `finally` que estoura esconde o
  erro de verdade. Limpeza ganha `catch`.
- **Nunca rode a suíte com `| tail`**: não achar a linha no que sobrou parece
  "não aconteceu".
- **Sonda instalada cedo demais mede o nada**: um `MutationObserver` num
  `addInitScript` observa um `body` que ainda é `null`.
- **`boundingBox` devolve a caixa de um elemento coberto**, e o clique acerta
  quem está por cima.
- **Parser que ignora o que não entende inventa achados.** Falhe no que não
  sabe ler.
- **Comparar tamanho responde sobre tamanho.** A pergunta costuma ser o LIMIAR
  da norma.
- **Captura de tela não diz nada sobre animação**: o `screenshot()` finaliza toda
  animação finita antes de fotografar.
- **Sonda de vida longa mede tudo na janela dela**: armada antes do gesto, ela
  conta a mão. A janela é parte do desenho.
- **"Existe agora" e "existe quando importa" são perguntas diferentes.**
- **Varredura de um nível não é varredura**: o que vale é o fecho transitivo.
- **Leia a norma até o fim**: o WCAG tem exceções que medir tamanho não enxerga.
- **Meça o que é DESENHADO, não o que foi escrito**: herança é a regra do CSS.
- **Transição em curso faz o computado mentir.** Pergunte com `getAnimations()`
  se há transição, em vez de escolher outro instante.
- **Mostrador cujo repouso é igual ao sucesso não testemunha o sucesso.** Ele
  mostra algo diferente antes e depois?
- **Lista de PROIBIDOS subconta em silêncio.** Use uma de PERMITIDOS e falhe no
  que não conhece.
- **O medidor de contraste ignora o ALFA**: sabotar a opacidade é sabotagem
  inerte, e não prova guarda nenhum.

**Antes de ler AUSÊNCIA como evidência, prove que o canal estaria lá se o evento
tivesse acontecido**: uma linha que sai sempre, ou o caso positivo conhecido. E
o canal pode morrer depois de instalado — navegação descarta o documento e o
observador junto, e a lista vazia diz "nada mudou". **Afirme o documento antes
de afirmar o silêncio.**

## Instrumento que DECOMPÕE tem denominador de graça: a soma

**Toda decomposição afirma a soma antes das parcelas.** Uma tabela de parcelas
com nomes e alturas certas pode somar o dobro do todo sem nada reclamar; a soma
que não fecha é o que denuncia. Monte a medição de forma que cada parte tenha um
dono só (uma coluna de sondas `elementFromPoint` fecha por construção). E o
EFEITO de um corte se mede cortando de verdade (`display:none` e medir de
novo), nunca subtraindo.

## Um guarda só mede o que ele VISITA

Cobertura de contraste, tipografia e leiaute depende de onde o teste NAVEGA, não
de quantas asserções ele tem. Quatro formas de não visitar:

1. **A cena não está na lista.** Enumerar é remendo. O que restaura a amostragem
   é um guarda que force a VARREDURA ("alguém escreveu a receita à mão?"), que
   vale para a cena que nascer amanhã.
2. **O medidor não é importável.** Instrumento privado de um arquivo de teste
   tem exatamente um chamador.
3. **A tela tem UM item.** O defeito de N>1 é invisível com N=1. Quantos itens o
   caso põe na tela, e de quantos o defeito precisa?
4. **Todas as telas, um só DADO.** Tela que ramifica pelo dado pede um caso por
   ramo, com o ramo nomeado e o controle afirmando que ele apareceu.

**O controle que fecha as quatro é o DENOMINADOR**: todo medidor devolve
`{falhas, medidos}`. Sem isso, "verde" e "não mediu" são a mesma cor.

## O resto

- **Apague teste que custa mais do que protege**: asserção sobre nome de classe e
  forma de DOM que ninguém prometeu, teste que redescobre o esperado rodando a
  implementação, teste sobre código morto ou fora do bundle. Teste verde sobre
  código que ninguém usa é a pior dívida — cobra manutenção e não protege nada.
- Dado transcrito do livro (catálogos) é validado por SCHEMA, não por
  um `expect` por campo repetindo o mesmo número. Prenda a *exceção* (a armadilha
  da tabela), nunca a tabela inteira.
- Troque I/O externo (API, banco, sistema de arquivos) por classes dublê nomeadas, não por stub inline. Testes F.I.R.S.T: rápidos, independentes,
  repetíveis, auto-verificáveis, oportunos.
- **Teste nasce antes do código.** Sem o teto de linha por função (ver "Estilo
  de código"), uma função pode ser o comportamento inteiro, e o teste que a
  define cabe ser escrito primeiro — na camada que a seção já manda usar
  (integração por padrão, unitário só pra regra). Caçando defeito, reproduza no
  navegador ou num handler antes: o defeito reproduzido É o teste vermelho, e
  ele nasce antes do conserto por definição.
- **Ordem do corte: escreva o substituto, veja-o verde, DEPOIS apague.** Apagar
  primeiro abre uma janela cega.
