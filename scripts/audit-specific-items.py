#!/usr/bin/env python3
"""Confere os ITENS ESPECÍFICOS do catálogo contra o livro (ALE-448).

São DUAS TABELAS e UM auditor, como nos encantos: a **8-9** é de armas (p337) e
a **8-11** é de armaduras e escudos (p340). A forma das duas é a mesma — `d%` +
nome + PREÇO na tabela, `Nome. Regra` no verbete — e o que muda entre elas são
coordenadas e três constantes. Rode `audit-specific-items.py arma` ou `protecao`
para uma só.

A terceira âncora é o ITÁLICO, e ela é a razão deste auditor existir
--------------------------------------------------------------------
Um item específico não é regra nova: ele é um item BASE com sobreposições já
postas, e o livro escreve QUAIS em itálico dentro do verbete —

    "Este machado de guerra DE GELO ETERNO CONGELANTE FORMIDÁVEL fornece
     redução de fogo 10" (Avalanche, p337)

— que é a base, o material e dois encantos. Transcrever isso a olho é onde o
erro entra: a composição de 31 verbetes são 31 listas de três a cinco peças,
todas em letra miúda no meio da prosa. Aqui ela é COLHIDA, e o `composedOf` do
catálogo é conferido contra o que a página diz.

O itálico vem de outro leitor (`pdftohtml -xml`), e a conversão de escala é do
`t20pdf.italicos_da_pagina`.

O denominador que a tabela dá de graça
---------------------------------------
As faixas de `d%` TILAM 1..100. Uma leitura que perca uma linha não fecha, e o
auditor para antes de encostar no catálogo — é a mesma rede do auditor de
encantos, e ela já pegou duas leituras tortas lá.
"""
import json
import pathlib
import re
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import t20pdf as P

OFFSET_DO_PDF = 6
FOLGA = 3.0

# A ÚLTIMA LINHA da tabela vem COLAPSADA num `<line>` só — "96-100 Vingadora
# sagrada" —, e é por isso que o padrão aceita cauda. É a mesma armadilha das
# duas tabelas de encanto, na mesma posição.
RE_FAIXA = re.compile(r'^(\d{1,3})(?:-(\d{1,3}))?(?:\s+(.*))?$')
# O verbete de um item específico tem nome COMPOSTO — "Azagaia dos Relâmpagos",
# "Cajado do Poder" —, então ele não se acha por regex de uma palavra: quem diz
# o que procurar é a TABELA, que é a outra âncora.
RE_PRECO = re.compile(r'T\$\s*([\d.]+)')


class Perfil:
    """Uma das duas tabelas de item específico, com onde ela mora na página."""

    def __init__(self, *, chave, tabela, pagina_da_tabela, paginas_dos_verbetes,
                 x_do_dado, x_do_nome, x_do_preco, colunas_da_prosa,
                 regiao_da_tabela, fim_dos_verbetes):
        self.chave = chave
        self.tabela = tabela
        self.pagina_da_tabela = pagina_da_tabela
        self.paginas_dos_verbetes = paginas_dos_verbetes
        self.x_do_dado = x_do_dado
        self.x_do_nome = x_do_nome
        self.x_do_preco = x_do_preco
        self.colunas_da_prosa = colunas_da_prosa
        self.regiao_da_tabela = regiao_da_tabela
        self.fim_dos_verbetes = fim_dos_verbetes


PERFIS = {
    # ARMAS: a Tabela 8-9 mora na coluna direita da p337, e os verbetes
    # atravessam as duas páginas — a p337 nas duas colunas (a direita retoma
    # ABAIXO da tabela) e a p338 só na esquerda, porque a direita dela já é a
    # seção de armaduras.
    'arma': Perfil(
        chave='arma', tabela='8-9',
        pagina_da_tabela=343, paginas_dos_verbetes=(342, 343, 344),
        x_do_dado=(313.0, 320.0), x_do_nome=348.7, x_do_preco=(480.0, 490.0),
        # A MARGEM MUDA DE PÁGINA PAR PARA ÍMPAR: a coluna esquerda começa em
        # 51 na par e em 70,9 na ímpar. Lido do próprio livro, não suposto.
        colunas_da_prosa={342: ((51.0, 68.0), (289.1, 306.1)),
                          343: ((70.9, 87.9), (309.0, 326.0)),
                          344: ((51.0, 68.0),)},
        # A tabela ocupa o alto da coluna direita; a prosa de baixo retoma
        # depois dela. Sem o corte por ALTURA, as células da tabela entrariam
        # como linhas de prosa — foi o que o perfil de armas dos encantos
        # ensinou.
        # Duas tabelas a evitar, e elas estão em LADOS DIFERENTES da página: a
        # 8-8 ocupa a metade esquerda da p336 e a 8-9 o alto da direita da
        # p337. Por isso a região é um RETÂNGULO (x0, x1, y0, y1) e não um
        # canto: um corte por "x menor que" acertaria uma e perderia a outra.
        regiao_da_tabela={342: (0.0, 260.0, 0.0, 530.0),
                          343: (300.0, 581.0, 0.0, 380.0)},
        fim_dos_verbetes=('Armaduras',),
    ),
    # ARMADURAS E ESCUDOS: a Tabela 8-11 mora na coluna esquerda da p340, e os
    # verbetes vêm abaixo dela e na coluna direita, em duas seções com título
    # ("Armaduras Específicas" e "Escudos Específicos").
    'protecao': Perfil(
        chave='protecao', tabela='8-11',
        pagina_da_tabela=346, paginas_dos_verbetes=(346,),
        # Medido: o d% em 59,1 (56,9 na linha colapsada), o nome em 90,7 e o
        # preço entre 223,7 e 225,9 — a coluna de preço é alinhada à direita e
        # o número de seis dígitos empurra o começo dela.
        x_do_dado=(56.0, 60.0), x_do_nome=90.7, x_do_preco=(222.0, 228.0),
        colunas_da_prosa={346: ((51.0, 68.0), (289.1, 306.1))},
        regiao_da_tabela={346: (0.0, 250.0, 0.0, 292.0)},
        # DOIS títulos no meio da lista, um por seção. Sem eles o título entra
        # no corpo do verbete anterior e a cobertura dele cai sem motivo.
        fim_dos_verbetes=('Armaduras Específicas', 'Escudos Específicos'),
    ),
}


def linhas_da_tabela(perfil: Perfil):
    """As linhas da tabela do perfil: (inicio, fim, nome, preço).

    A linha se monta pelo Y, como no auditor de encantos: as três células são
    blocos diferentes, e o que diz que "88-92", "Avalanche" e "T$ 140.000" são a
    mesma linha é a altura das três ser igual.
    """
    celulas = P.linhas_com_coordenada(perfil.pagina_da_tabela)
    dados = [(y, RE_FAIXA.match(t).groups()) for x, y, t in celulas
             if perfil.x_do_dado[0] <= x <= perfil.x_do_dado[1] and RE_FAIXA.match(t)]
    # A célula do dado é alinhada à DIREITA, então "96-100" começa mais à
    # esquerda que "93-95": o x é uma FAIXA, e não um ponto.
    nomes = [(y, t) for x, y, t in celulas if abs(x - perfil.x_do_nome) < FOLGA]
    precos = [(y, t) for x, y, t in celulas
              if perfil.x_do_preco[0] <= x <= perfil.x_do_preco[1] and RE_PRECO.match(t)]

    for y, (inicio, fim, cauda) in sorted(dados):
        nome = cauda or next((t for yn, t in nomes if abs(yn - y) < P.MESMA_LINHA), None)
        preco = next((t for yp, t in precos if abs(yp - y) < P.MESMA_LINHA), None)
        if nome is None:
            continue
        valor = int(RE_PRECO.match(preco).group(1).replace('.', '')) if preco else None
        yield int(inicio), int(fim or inicio), nome.strip(), valor


def faixas_que_nao_fecham(linhas) -> str:
    """A tilagem de 1 a 100, que é o denominador de graça desta página."""
    coberto = {}
    for inicio, fim, nome, _preco in linhas:
        for d in range(inicio, fim + 1):
            if d in coberto:
                return f'{d}% aparece duas vezes ({coberto[d]} e {nome})'
            coberto[d] = nome
    faltando = [d for d in range(1, 101) if d not in coberto]
    if faltando:
        return f'{len(faltando)} valores de d% sem linha: {faltando[:8]}…'
    return ''


def verbetes(perfil: Perfil, nomes_da_tabela):
    """Os verbetes das páginas, com o texto, a página e onde ele COMEÇA.

    Quem diz o que procurar é a TABELA: o nome de um item específico tem três
    palavras e preposição no meio, e um regex de "palavra maiúscula seguida de
    ponto" acharia qualquer frase que termine em nome próprio. Procurar pelos
    nomes conhecidos é o que transforma as duas âncoras numa conferência — nome
    da tabela sem verbete sai NOMEADO no relatório.

    O COMEÇO é guardado para o itálico: a composição está na primeira frase, e
    quem a procura precisa saber em que página, em que coluna e entre que
    alturas olhar. Um verbete que atravessa a virada de página tem o começo numa
    e o resto na outra — e só a primeira interessa.
    """
    achados = {}
    atual = None
    for pagina in perfil.paginas_dos_verbetes:
        limite = perfil.regiao_da_tabela.get(pagina)
        for x, y, texto in P.leitura(pagina):
            coluna = coluna_de(x, perfil.colunas_da_prosa[pagina])
            if coluna is None:
                continue
            if limite and limite[0] <= x <= limite[1] and limite[2] <= y <= limite[3]:
                continue
            if comeca_secao(texto, perfil.fim_dos_verbetes):
                atual = None
                continue
            comeco = comeca_verbete(texto, {P.chave(n): n for n in nomes_da_tabela})
            if comeco:
                nome, resto = comeco
                atual = nome
                achados[nome] = {'corpo': [resto], 'pagina': pagina - OFFSET_DO_PDF,
                                 'pdf': pagina, 'coluna': coluna, 'y0': y, 'y1': y}
            elif atual is not None:
                achados[atual]['corpo'].append(texto)
                if pagina == achados[atual]['pdf'] and coluna == achados[atual]['coluna']:
                    achados[atual]['y1'] = y
    return achados


def coluna_de(x: float, colunas):
    """O par (corpo, recuo) em que esta linha está, ou None se ela está fora.

    A coluna é um PAR porque a primeira linha de um parágrafo é recuada: o
    corpo da esquerda da p337 começa em 70,9 e o recuo em 87,9. Quem compara com
    um x só perde uma das duas — e perder a RECUADA é perder justamente a linha
    que abre o verbete.
    """
    for corpo, recuo in colunas:
        if abs(x - corpo) < FOLGA or abs(x - recuo) < FOLGA:
            return (corpo, recuo)
    return None


def comeca_secao(texto: str, titulos) -> bool:
    """A linha é um TÍTULO de seção, e não prosa de verbete.

    Ela corta o verbete em andamento: na p340 a lista tem dois títulos no meio —
    "Armaduras Específicas" e "Escudos Específicos" —, e sem este corte o título
    vira a última frase do verbete anterior.
    """
    return any(texto.startswith(titulo) for titulo in titulos or ())


def comeca_verbete(texto: str, por_chave):
    """(nome da tabela, resto da linha) quando a linha ABRE um verbete."""
    cabeca, ponto, resto = texto.partition('. ')
    if not ponto:
        return None
    nome = por_chave.get(P.chave(cabeca))
    return (nome, resto) if nome else None


def composicao_do_verbete(verbete, pecas) -> tuple:
    """(frase em itálico, peças) — do que o item é feito, dito pelo livro.

    O livro põe em itálico TRÊS coisas dentro do mesmo verbete, e só a primeira
    é composição:

      - a composição — *"este machado de guerra de gelo eterno congelante
        formidável"*;
      - o próprio item mais adiante — *"quando você usa o machado silvestre"*;
      - o nome de uma MAGIA — *"se transforma em um Relâmpago"* — ou de um
        encanto que o verbete só CONCEDE em certa condição (o Martelo de
        Doherimm *adquire* o arremesso na mão de um anão).

    Duas perguntas separam as quatro, e são as duas que esta função faz:

      - **a frase é CONTÍGUA no verbete?** O itálico do martelo e o do arremesso
        caem em linhas vizinhas, mas entre eles há meia frase de texto normal —
        e "martelo de guerra formidável arremesso" não existe na página;
      - **ela nomeia peça que o catálogo conhece?** "Relâmpago" e "machado
        silvestre" não nomeiam nenhuma.

    Devolve vazio para o item que não tem composição, que é estado legítimo: a
    azagaia dos relâmpagos é uma azagaia comum com um efeito mágico, e a espada
    baronial escreve o +1 dela em prosa, sem encanto nenhum.
    """
    corpo_x, _recuo = verbete['coluna']
    texto = P.normaliza_frase(P.junta(verbete['corpo']))
    na_coluna = [(y, t) for x, y, t in P.italicos_da_pagina(verbete['pdf'])
                 if verbete['y0'] - 2 <= y <= verbete['y1'] + 2
                 and corpo_x - FOLGA <= x <= corpo_x + LARGURA_DA_COLUNA]
    for sequencia in sequencias(na_coluna):
        for inicio in range(len(sequencia)):
            for fim in range(len(sequencia), inicio, -1):
                frase = junta_hifenizado(sequencia[inicio:fim])
                if P.normaliza_frase(frase) not in texto:
                    continue
                achado = compoe(frase, pecas)
                if achado['pecas']:
                    return frase, achado
    return '', {'pecas': [], 'sobra': []}


# A LARGURA de uma coluna de texto deste livro, medida: o corpo começa em 70,9 e
# a linha mais longa acaba perto de 290. O itálico começa onde a FRASE está, e
# não onde a coluna começa, então o que o identifica é estar dentro da largura.
LARGURA_DA_COLUNA = 220.0

# A ALTURA DE UMA LINHA, medida nas páginas deste capítulo: 12,2 pontos. O teto
# aqui é maior de propósito — o que ele separa é "linha vizinha" de "outro
# parágrafo", e não duas linhas de alturas ligeiramente diferentes.
LINHA_VIZINHA = 14.0


def sequencias(trechos):
    """Agrupa trechos em linhas vizinhas; devolve cada grupo em ordem."""
    grupo = []
    for y, texto in trechos:
        if grupo and y - grupo[-1][0] > LINHA_VIZINHA:
            yield [t for _y, t in grupo]
            grupo = []
        grupo.append((y, texto))
    if grupo:
        yield [t for _y, t in grupo]


def junta_hifenizado(trechos) -> str:
    """Remonta a frase que a quebra de linha partiu.

    O HÍFEN às vezes vem SOZINHO num trecho — "armadura completa refor" + "-" +
    "çada defensora de adamante" (Baluarte anão, p340) —, e não só colado no fim
    de um. Ele é gancho dos dois jeitos: entra sem espaço, e o trecho seguinte
    cola nele.
    """
    frase = ''
    for trecho in trechos:
        if frase.endswith('-'):
            frase = frase[:-1] + trecho
        elif trecho == '-':
            frase += '-'
        elif frase:
            frase += ' ' + trecho
        else:
            frase = trecho
    return re.sub(r'\s+', ' ', frase).strip()


CATALOGO = P.RAIZ / 'engine-go/domain/catalog/data/items.json'


def do_catalogo():
    return json.loads(CATALOGO.read_text())


def pecas_conhecidas(itens):
    """Nome normalizado → lista de candidatos, com as variantes de GÊNERO.

    O livro escreve a peça concordando com o item: a cota de malha é
    "defensora" e o escudo é "defensor", o machado é "congelante" e a armadura é
    "macabra". O catálogo guarda UMA grafia por peça, então quem casa as duas é
    esta tabela — e ela GERA as variantes em vez de listá-las, senão a próxima
    peça com gênero entraria sem casar e seria reportada como ausente.

    O valor é uma LISTA porque a mesma palavra é duas peças diferentes: o
    "defensor" do bordão é o encanto de ARMA (p335) e o da cota de malha é o de
    ARMADURA (p338). Quem desempata é a família do item, no `compoe`.
    """
    tabela = {}
    for item in itens:
        if item.get('composedOf'):
            continue
        for variante in variantes_de_genero(item['name']):
            tabela.setdefault(P.chave(variante), []).append(
                (item['id'], item['category'], tuple(item.get('appliesTo') or ())))
    return tabela


# As categorias que são SOBREPOSIÇÃO. O resto é item, e item é o que pode ser
# base.
SOBREPOSICOES = frozenset({'improvement', 'material', 'weapon-enchant', 'armor-enchant'})


def familia_de(categoria: str) -> str:
    """A família do `appliesTo` para a categoria de um item base."""
    if categoria.startswith('weapon'):
        return 'weapon'
    if categoria.startswith('armor'):
        return 'armor'
    return categoria


def variantes_de_genero(nome: str):
    """As grafias possíveis de um nome: "Defensora" também aparece "defensor".

    Três pares, que são os do português e os que o catálogo usa: `-o/-a`
    (macabro/macabra), `-ão/-ã` (guardião/guardiã) e `-or/-ora`
    (defensor/defensora). Gerar MAIS do que isso traria grafia que não existe e
    daria a ela a chance de casar com outra peça.
    """
    saida = {nome}
    palavras = nome.split()
    # A FLEXÃO CAI NA CABEÇA do nome, e nem sempre ela é a última palavra:
    # "Banhado a ouro" vira "banhada a ouro" na Couraça do comando (p340). Por
    # isso as duas pontas são tentadas — a primeira palavra e a última.
    for i in (0, len(palavras) - 1):
        for flexionada in flexoes(palavras[i]):
            troca = list(palavras)
            troca[i] = flexionada
            saida.add(' '.join(troca))
    return saida


def flexoes(palavra: str):
    """As grafias de gênero de UMA palavra: três pares, que são os do português
    e os que o catálogo usa."""
    for um, outro in (('ão', 'ã'), ('or', 'ora'), ('o', 'a')):
        for fim, troca in ((um, outro), (outro, um)):
            if palavra.lower().endswith(fim):
                return [palavra[: len(palavra) - len(fim)] + troca]
    return []


def compoe(frase: str, pecas) -> dict:
    """Quebra a frase em itálico nas peças que o catálogo conhece.

    Casa pelo TRECHO MAIS LONGO primeiro — "machado de guerra" antes de
    "machado" —, porque o nome de uma arma tem até quatro palavras e o de um
    encanto tem uma, e a ordem errada partiria a arma em dois achados.

    O "de" é COLA e não palavra: ele tanto está DENTRO de um nome ("machado de
    guerra") quanto LIGA duas peças ("machado de guerra de gelo eterno"). Por
    isso ele entra nas tentativas e, se nenhuma casar, é descartado em vez de
    virar sobra — sobra é o que o auditor pede para um humano olhar, e "de" não
    é.

    A FAMÍLIA sai da BASE e desempata o resto: "defensor" é encanto de arma
    (p335) e também de armadura (p338), e num bordão é o primeiro. Enquanto a
    base não aparece não há família, e o único candidato aceito é um item — que
    é exatamente o que a base é.
    """
    palavras = [p for p in re.split(r'\s+', frase) if p]
    achados, sobra, i, familia = [], [], 0, ''
    while i < len(palavras):
        casou = False
        for tamanho in range(min(5, len(palavras) - i), 0, -1):
            peca = escolhe(pecas.get(P.chave(' '.join(palavras[i:i + tamanho]))), familia)
            if peca:
                achados.append(peca)
                if not familia and peca[1] not in SOBREPOSICOES:
                    familia = familia_de(peca[1])
                i += tamanho
                casou = True
                break
        if not casou:
            if palavras[i].lower() not in CONECTIVOS:
                sobra.append(palavras[i])
            i += 1
    return {'pecas': achados, 'sobra': sobra, 'familia': familia}


def escolhe(candidatos, familia: str):
    """O candidato que CABE no item: a sobreposição pela família, o resto por
    ser item."""
    for id_, categoria, aplica_a in candidatos or ():
        if categoria not in SOBREPOSICOES:
            if not familia:
                return (id_, categoria)
            continue
        if not familia or not aplica_a or familia in aplica_a:
            return (id_, categoria)
    return None


# As palavras que LIGAM peças sem serem peça. Elas também aparecem DENTRO de um
# nome ("machado de guerra"), e é por isso que o descarte vem depois de todas as
# tentativas de casar, nunca antes.
CONECTIVOS = frozenset({'de', 'do', 'da', 'e'})


# O ITÁLICO QUE A PÁGINA PERDEU, nomeado um a um e com a prova.
#
# O Escudo do Conjurador é *"Este escudo leve defensor tem uma pequena tira…"*, e
# a página italiza "escudo  leve " e deixa "defensor" em redondo: o estilo não
# atravessa a quebra de linha, e o XML mostra `font="4"` numa e `font="3"` na
# seguinte. É deslize de tipografia, não regra — os outros doze verbetes desta
# tabela trazem o encanto DENTRO do itálico, e o preço acompanha: T$ 45.000 para
# este escudo LEVE defensor contra T$ 50.000 do escudo PESADO defensor.
#
# A exceção é ESCRITA e sai no relatório. O auditor continua conferindo o
# catálogo contra ela — o que muda é de onde vem a composição esperada, não se
# ela é conferida.
ITALICO_QUE_A_PAGINA_PERDEU = {
    'Escudo do conjurador': ['escudo-leve', 'encanto-defensor'],
}

# O PISO DA COBERTURA, igual ao do auditor de encantos e pela mesma razão: a
# descrição do catálogo é TRANSCRITA do verbete, então ela nasce em 100%. O que
# o piso prende não é a transcrição de hoje — é o dia em que alguém "melhorar" a
# frase e ela passar a dizer outra coisa.
PISO_DA_COBERTURA = 0.95


def audita(perfil: Perfil) -> int:
    """Audita UMA tabela e devolve quantas falhas achou."""
    print(f'\n══ TABELA {perfil.tabela} ({perfil.chave}) ' + '═' * 40)
    tabela = list(linhas_da_tabela(perfil))

    # ── ÂNCORA 1 contra si mesma, antes de o catálogo ser mencionado ─────
    buraco = faixas_que_nao_fecham(tabela)
    if buraco:
        print(f'A TABELA {perfil.tabela} NÃO FECHA: {buraco}')
        print('Nada do que vem abaixo vale — o auditor leu a página errado.')
        return 1

    # ── ÂNCORA 1 contra ÂNCORA 2 ─────────────────────────────────────────
    corpos = verbetes(perfil, [nome for _i, _f, nome, _p in tabela])
    falhas = []
    for _i, _f, nome, _preco in tabela:
        if nome not in corpos:
            falhas.append(f'{nome}: está na tabela {perfil.tabela} e não tem verbete')

    itens = do_catalogo()
    pecas = pecas_conhecidas(itens)
    por_nome = {P.chave(i['name']): i for i in itens}

    print(f'{"item":26} {"d%":>7} {"preço":>10}  catálogo')
    transcritos = 0
    for inicio, fim, nome, preco in tabela:
        entrada = por_nome.get(P.chave(nome))
        faixa = f'{inicio}-{fim}' if fim != inicio else str(inicio)
        if entrada is None:
            print(f'{nome:26} {faixa:>7} {preco:>10}  — AUSENTE')
            continue
        transcritos += 1
        falhas += confere_entrada(perfil, entrada, corpos.get(nome), nome, preco, pecas)
        print(f'{nome:26} {faixa:>7} {preco:>10}  {entrada["id"]}')

    print(f'\n{transcritos}/{len(tabela)} transcritos')
    for falha in falhas:
        print(f'  ✗ {falha}')
    return len(falhas)


def confere_entrada(perfil, entrada, verbete, nome, preco, pecas):
    """As quatro perguntas que se faz a um item específico já transcrito."""
    falhas = []
    if entrada.get('price') != preco:
        falhas.append(f'{nome}: o catálogo diz T$ {entrada.get("price")} e a tabela '
                      f'{perfil.tabela} diz T$ {preco}')
    if verbete is None:
        return falhas

    pagina = verbete['pagina']
    if entrada.get('bookPage') != pagina:
        falhas.append(f'{nome}: o catálogo cita p{entrada.get("bookPage")} e o verbete '
                      f'está na p{pagina}')

    # A COMPOSIÇÃO, que é a razão deste auditor existir.
    _frase, achado = composicao_do_verbete(verbete, pecas)
    do_livro = [id_ for id_, _categoria in achado['pecas']]
    if nome in ITALICO_QUE_A_PAGINA_PERDEU:
        do_livro = ITALICO_QUE_A_PAGINA_PERDEU[nome]
        print(f'  ! {nome}: usando a composição ESCRITA À MÃO — a página perde o '
              f'itálico no meio da frase (ver o comentário)')
    do_catalogo_ = composicao_declarada(entrada)
    texto_do_verbete = P.junta(verbete['corpo'])
    if do_livro:
        # CONJUNTO e não lista: a ordem é a da PROSA, e ela varia de verbete
        # para verbete — "defensora de mitral" põe o material por último e
        # "de gelo eterno congelante formidável" o põe antes dos encantos. O
        # que o catálogo guarda é a composição, não a ordem em que o livro a
        # escreveu.
        if sorted(do_livro) != sorted(do_catalogo_):
            falhas.append(f'{nome}: o livro compõe {sorted(do_livro)} e o catálogo '
                          f'declara {sorted(do_catalogo_)}')
    else:
        # SEM ITÁLICO NÃO É SEM BASE. O item que não tem encanto nenhum — a
        # azagaia dos relâmpagos, a espada baronial — não ganha itálico, e
        # continua sendo uma azagaia: a base é dita em prosa, e é no texto do
        # verbete que ela se confere. O que o livro proíbe aqui é SOBREPOSIÇÃO
        # declarada sem o livro ter escrito nenhuma.
        falhas += base_sem_italico(nome, entrada, do_catalogo_, texto_do_verbete)
    if achado['sobra']:
        falhas.append(f'{nome}: o itálico tem palavra que não é peça conhecida: '
                      f'{achado["sobra"]}')

    # A COBERTURA, para a descrição não virar outra coisa com o tempo.
    razao, _comuns, _total = P.cobertura(texto_do_verbete, entrada.get('description', ''))
    if razao < PISO_DA_COBERTURA:
        falhas.append(f'{nome}: a descrição cobre {razao:.0%} do verbete '
                      f'(piso {PISO_DA_COBERTURA:.0%})')
    return falhas


def base_sem_italico(nome, entrada, declarado, texto) -> list:
    """O que se cobra de um item que o livro não compôs em itálico."""
    if len(declarado) > 1:
        return [f'{nome}: o catálogo declara {declarado[1:]} e o verbete não escreve '
                f'sobreposição nenhuma em itálico']
    if not declarado:
        return [f'{nome}: o catálogo não diz de que item ele é feito']
    base = next((i for i in do_catalogo() if i['id'] == declarado[0]), None)
    if base is None:
        return [f'{nome}: a base {declarado[0]} não existe no catálogo']
    if P.normaliza_frase(base['name']) not in P.normaliza_frase(texto):
        return [f'{nome}: o catálogo diz que a base é {base["name"]!r}, e o verbete '
                f'não a nomeia']
    return []


def composicao_declarada(entrada) -> list:
    """A composição do catálogo na mesma ORDEM em que o livro a escreve: base,
    depois o que vem em itálico depois dela."""
    composto = entrada.get('composedOf')
    if not composto:
        return []
    return ([composto['base']] + list(composto.get('improvements') or ())
            + ([composto['material']] if composto.get('material') else [])
            + list(composto.get('enchants') or ()))


def main() -> int:
    escolhidos = sys.argv[1:] or list(PERFIS)
    desconhecidos = [c for c in escolhidos if c not in PERFIS]
    if desconhecidos:
        print(f'perfil desconhecido: {desconhecidos}. Conheço {list(PERFIS)}.')
        return 2
    return min(1, sum(audita(PERFIS[chave]) for chave in escolhidos))


if __name__ == '__main__':
    raise SystemExit(main())
