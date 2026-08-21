#!/usr/bin/env python3
"""Confere o bestiário do app contra o livro, verbete por verbete (ALE-151).

Rodar é ATO DELIBERADO, como o `genoracle`: a ferramenta PROPÕE, o diff se
revisa contra o PDF, e é a revisão que decide. Ela nunca escreve sozinha.

    python3 scripts/audit-bestiary.py            # relatório
    python3 scripts/audit-bestiary.py --aplicar  # escreve as correções

Como ela sabe que está lendo a criatura certa
---------------------------------------------
Não por confiança no parser. Um bloco só conta quando é CONTÍGUO e COMPLETO —
tipo e tamanho, Iniciativa/Percepção, Defesa com as três resistências, Pontos
de Vida, uma linha de atributos, e termina em Tesouro — E quando a Defesa e os
Pontos de Vida batem com o que o catálogo já tem. Essas duas âncoras dizem "é
esta criatura"; o que divergir depois disso é erro do catálogo.

Três armadilhas que custaram caro e estão consertadas aqui
----------------------------------------------------------
1. `pdftotext -layout` junta colunas VIZINHAS na mesma linha de texto. Uma
   linha de atributos aparece colada à criatura errada, e eu quase "corrigi" a
   Hidra ao contrário por causa disso. A leitura é por COORDENADA.
2. A geometria varia: há página de duas e de três colunas, e o título de seção
   é centralizado e atravessa a calha. As colunas são achadas por FREQUÊNCIA do
   x onde o corpo começa, não por corte no meio nem por fusão de faixas.
3. O livro escreve o nome de quatro jeitos: numa linha, junto do nível de
   desafio, quebrado em duas linhas, e quebrado COM o nível no meio
   ("Cavaleiro do Leopardo" / "ND 9" / "Sangrento"). O casamento vai do nome
   mais longo para o mais curto, senão "Troll" come "Troll das Cavernas".

Precisa do `pdftotext` (poppler) e do PDF do livro na raiz do repositório.
"""
import argparse
import html
import json
import re
import subprocess
import unicodedata
from collections import Counter

PDF = '/mnt/HD/projects/tormenta20/t20-book.pdf'
RE_BLOCO = re.compile(
    r'<block xMin="([\d.]+)" yMin="([\d.]+)" xMax="([\d.]+)"[^>]*>(.*?)</block>', re.S)
RE_LINHA = re.compile(r'<line[^>]*>(.*?)</line>', re.S)
RE_PALAVRA = re.compile(
    r'<word xMin="([\d.]+)" yMin="([\d.]+)"[^>]*>(.*?)</word>', re.S)



def blocos_da_pagina(pagina: int):
    """(xMin, yMin, [linhas de texto]) de cada bloco declarado pelo PDF."""
    xml = subprocess.run(
        ['pdftotext', '-bbox-layout', '-f', str(pagina), '-l', str(pagina), PDF, '-'],
        capture_output=True, text=True).stdout
    for m in RE_BLOCO.finditer(xml):
        x0, ybloco, x1 = float(m.group(1)), float(m.group(2)), float(m.group(3))
        linhas, y0 = [], None
        for lm in RE_LINHA.finditer(m.group(4)):
            palavras = [
                (float(x), float(y), html.unescape(t))
                for x, y, t in RE_PALAVRA.findall(lm.group(1))
            ]
            if not palavras:
                continue
            if y0 is None:
                y0 = palavras[0][1]
            palavras.sort(key=lambda w: w[0])
            linhas.append(' '.join(t for _x, _y, t in palavras))
        if linhas:
            yield x0, x1, y0 if y0 is not None else ybloco, linhas


from collections import Counter

MINIMO_DE_BLOCOS_POR_COLUNA = 3


def inicios_das_colunas(bs) -> list[float]:
    """Os x onde as colunas COMEÇAM, achados por frequência.

    Não serve fundir faixas de x: o título de seção é centralizado e atravessa
    a calha, e uma única linha dessas funde as duas colunas numa só — foi assim
    que o Orc e o Glop saíram interfoliados. O que é estável é o x onde o corpo
    começa: numa página de duas colunas ele aparece dezenas de vezes em dois
    valores, e tudo que começa mais à direita (o "ND 1/4" alinhado à direita, a
    linha de atributos centralizada) pertence à coluna que vem ANTES dele.
    """
    contagem = Counter(round(b[0], 1) for b in bs)
    inicios = sorted(x for x, n in contagem.items() if n >= MINIMO_DE_BLOCOS_POR_COLUNA)
    return inicios or [min((b[0] for b in bs), default=0.0)]


def texto_da_pagina(pagina: int) -> str:
    bs = list(blocos_da_pagina(pagina))
    if not bs:
        return ''
    inicios = inicios_das_colunas(bs)

    def coluna(x0: float) -> int:
        cabem = [i for i, ini in enumerate(inicios) if x0 >= ini - 2]
        return cabem[-1] if cabem else 0

    bs.sort(key=lambda b: (coluna(b[0]), b[2]))
    saida: list[str] = []
    atual = None
    for x0, _x1, _y, linhas in bs:
        c = coluna(x0)
        if atual is not None and c != atual:
            saida.append('--- fim de coluna ---')
        atual = c
        saida.extend(linhas)
    return '\n'.join(saida)




PRIMEIRA, ULTIMA = 292, 322  # PDF; livro = PDF - 6
BESTIARIO = '/mnt/HD/projects/tormenta20/engine-go/catalog/data/bestiary.json'
LIXO = re.compile(r'Mateus Santos|mateush\.santos|^Capítulo|^\d{1,3}$|fim de coluna')
MENOS = '–−—'  # o livro usa travessão, não hífen, nos negativos


def num(t: str) -> int:
    return int(t.strip().translate({ord(c): '-' for c in MENOS}).replace('+', ''))


def chave(nome: str) -> str:
    s = unicodedata.normalize('NFD', nome.lower())
    return re.sub(r'[^a-z0-9]', '', ''.join(c for c in s if not unicodedata.combining(c)))


def linhas() -> list[tuple[int, str]]:
    out = []
    for pdf in range(PRIMEIRA, ULTIMA + 1):
        for linha in texto_da_pagina(pdf).split('\n'):
            t = ' '.join(linha.split())
            if t and not LIXO.search(t):
                out.append((pdf - 6, t))
    return out


# Teto de linhas para a frase do tesouro. Um verbete passa disso — o Reishid,
# cujo tesouro é seguido da DESCRIÇÃO do item mágico —, e nesse caso a
# ferramenta avisa em vez de cortar calada.
LINHAS_DE_TESOURO = 4

RE_ND_LINHA = re.compile(r'^\s*ND\s+\d+(?:/\d+)?\s*$', re.I)
RE_ND_FIM = re.compile(r'\s*ND\s+\d+(?:/\d+)?\s*$', re.I)


def blocos_por_nome(nomes: set[str]) -> dict[str, dict]:
    """Do nome de cada verbete até a linha de Tesouro dele.

    O casamento é tolerante porque o livro escreve o nome de três jeitos: numa
    linha só ("Bandido"), na mesma linha do nível de desafio ("Lobo-das-Cavernas
    ND 2") e quebrado em duas ("Guerreiro" / "de Chifres"). Exigir linha inteira
    igual perdia catorze verbetes.
    """
    ls = linhas()
    inicios: list[tuple[int, int, str]] = []  # (linha, quantas linhas do nome, nome)
    i = 0
    while i < len(ls):
        casou = None
        # Do MAIS LONGO para o mais curto: o catálogo tem "Troll" e "Troll das
        # Cavernas", e a janela de uma linha comia o primeiro, deixando "das
        # Cavernas" órfão e o segundo sem bloco.
        for n in (3, 2, 1):
            if i + n > len(ls):
                continue
            # O nível de desafio pode cair NO MEIO do nome quando ele quebra em
            # duas linhas: "Cavaleiro do Leopardo" / "ND 9" / "Sangrento".
            pedacos = [t for _p, t in ls[i:i + n] if not RE_ND_LINHA.match(t)]
            k = chave(RE_ND_FIM.sub('', ' '.join(pedacos)))
            if k in nomes:
                casou = (n, k)
                break
        if casou:
            inicios.append((i, casou[0], casou[1]))
            i += casou[0]
        else:
            i += 1

    out: dict[str, dict] = {}
    for k, (i, n, nome) in enumerate(inicios):
        fim = inicios[k + 1][0] if k + 1 < len(inicios) else len(ls)
        corpo = [t for _p, t in ls[i + n:fim]]
        truncado = False
        # Todo bloco do livro TERMINA na linha de Tesouro. Sem esse corte ele
        # corre até o próximo verbete do CATÁLOGO — e o livro tem criaturas que
        # o catálogo não tem, então engolia os números do vizinho.
        # O corte é no Tesouro que vem DEPOIS da linha de atributos: num bloco
        # longo, uma habilidade pode citar tesouro antes dela, e cortar ali
        # deixava o verbete sem os seis atributos (Thuwarokk).
        vistos_atributos = False
        for j, t in enumerate(corpo):
            if RE_ATTR.search(t):
                vistos_atributos = True
            if 'Tesouro' in t and vistos_atributos:
                # O tesouro é a última FRASE do bloco, e ela atravessa linhas:
                # "Tesouro Um ninho de grifo tem 25% de chance de conter" /
                # "1d4 ovos no valor de T$ 2.500 cada." Cortar na palavra
                # deixava seis verbetes sem tesouro e dezesseis com a frase
                # decepada no meio. Vai até a linha que FECHA a frase.
                fim = j + 1
                while (fim <= j + LINHAS_DE_TESOURO and fim <= len(corpo)
                       and not corpo[fim - 1].rstrip().endswith('.')):
                    fim += 1
                truncado = not corpo[min(fim, len(corpo)) - 1].rstrip().endswith('.')
                corpo = corpo[:fim]
                break
        out.setdefault(nome, {'pagina': ls[i][0], 'corpo': corpo, 'truncado': truncado})
    return out


RE_INI = re.compile(r'Iniciativa\s*([+–−-]?\d+)\s*,\s*Percepção\s*([+–−-]?\d+)')
RE_DEF = re.compile(
    r'Defesa\s*(\d+)\s*,\s*Fort\s*([+–−-]?\d+)\s*,\s*Ref\s*([+–−-]?\d+)\s*,\s*Von\s*([+–−-]?\d+)')
RE_PV = re.compile(r'Pontos de Vida\s*(\d+)')
RE_PM = re.compile(r'Pontos de Mana\s*(\d+)')
RE_ATTR = re.compile(
    r'For\s*([+–−-]?\d+|—)\s*,?\s*Des\s*([+–−-]?\d+|—)\s*,?\s*Con\s*([+–−-]?\d+|—)\s*,?\s*'
    r'Int\s*([+–−-]?\d+|—)\s*,?\s*Sab\s*([+–−-]?\d+|—)\s*,?\s*Car\s*([+–−-]?\d+|—)')
RE_PERICIAS = re.compile(r'Perícias\s+(.+?)(?=\s*(?:Equipamento|Tesouro)\b|\.$|$)')
RE_EQUIP = re.compile(r'Equipamento\s+(.+?)(?=\s*Tesouro\b|$)')
# Até o FIM: o valor pode ter ponto dentro ("T$ 2.500") e o bloco já termina
# na frase do tesouro, então não há o que engolir depois dela.
RE_TESOURO = re.compile(r'Tesouro\s+(.+)$')


def ultimo(regex: re.Pattern, texto: str):
    """A ÚLTIMA ocorrência, para os campos da CAUDA do bloco.

    A linha de atributos e as de Perícias/Equipamento/Tesouro ficam sempre no
    fim, coladas umas nas outras. Quando um bloco engole conteúdo de outra
    criatura — acontece onde a ordem das colunas não é a de leitura —, a
    PRIMEIRA linha de atributos é a do intruso: foi assim que o Otyugh quase
    ganhou os 17 de Força do Sacerdote de Aharadak.
    """
    m = None
    for m in regex.finditer(texto):
        pass
    return m


RE_HIFEN = re.compile(r'(\w)-\s+(\w)')


def junta(corpo: list[str]) -> str:
    """Junta as linhas do bloco DESFAZENDO a hifenização de quebra.

    O livro quebra palavra no fim da linha ("subter-" / "râneos"), e juntar sem
    cuidado produz "subter- râneos" dentro de uma nota de perícia.
    """
    return RE_HIFEN.sub(r'\1\2', ' '.join(corpo))


def le(bloco: dict) -> dict:
    texto = junta(bloco['corpo'])
    campos: dict = {}
    if m := RE_INI.search(texto):
        campos['iniciativa'], campos['percepcao'] = num(m.group(1)), num(m.group(2))
    if m := RE_DEF.search(texto):
        campos['defesa'] = int(m.group(1))
        campos['fortitude'], campos['reflexos'], campos['vontade'] = (
            num(m.group(2)), num(m.group(3)), num(m.group(4)))
    if m := RE_PV.search(texto):
        campos['hp'] = int(m.group(1))
    if m := RE_PM.search(texto):
        campos['pm'] = int(m.group(1))
    if m := ultimo(RE_ATTR, texto):
        for nome, g in zip(
            ('forca', 'destreza', 'constituicao', 'inteligencia', 'sabedoria', 'carisma'),
            m.groups()):
            campos[nome] = None if g == '—' else num(g)
    if m := ultimo(RE_PERICIAS, texto):
        campos['skills'] = le_pericias(m.group(1))
    if m := ultimo(RE_EQUIP, texto):
        campos['equipamento'] = m.group(1).strip().rstrip('.')
    if m := ultimo(RE_TESOURO, texto):
        # Sem o ponto final: ele é pontuação da frase do livro, não parte do
        # valor — "Padrão", e não "Padrão.". Quem exibe põe a sua.
        campos['tesouro'] = m.group(1).strip().rstrip('.')
    return campos



RE_PERICIA = re.compile(
    r'^(?P<nome>[^,+]+?(?:\([^)]*\))?)\s*(?P<bonus>[+–−-]\d+)\s*(?:\((?P<nota>[^)]*)\))?$')


def le_pericias(linha: str) -> list[dict]:
    """"Furtividade +4 (+14 em pântanos), Ofício (armeiro) +2" → estruturado.

    Os dois parênteses do livro querem coisas OPOSTAS e a diferença é a posição:
    antes do bônus ele faz parte do NOME da perícia (`Ofício (armeiro)`, que é
    a perícia com especialização do livro); depois do bônus ele é uma condição
    (`+14 em pântanos`), que vai para a nota — a mesma escolha que a linha de
    ataque já fazia com `special`.
    """
    fora: list[dict] = []
    # Vírgula separa perícias, mas NÃO a que estiver dentro de parênteses.
    partes = re.split(r',\s*(?![^(]*\))', linha.strip().rstrip('.'))
    for parte in partes:
        m = RE_PERICIA.match(parte.strip())
        if not m:
            continue
        pericia = {'name': m.group('nome').strip(), 'bonus': num(m.group('bonus'))}
        if m.group('nota'):
            pericia['nota'] = m.group('nota').strip()
        fora.append(pericia)
    return fora


CONFERIDOS = ('hp', 'defesa', 'fortitude', 'reflexos', 'vontade',
              'forca', 'destreza', 'constituicao', 'inteligencia', 'sabedoria', 'carisma')


# O que IDENTIFICA a criatura: o bloco tem cabeça de bloco de criatura.
EXIGIDAS = (
    re.compile(r'Iniciativa\s*[+–−-]?\d+'),
    re.compile(r'Defesa\s*\d+.*Fort.*Ref.*Von'),
    re.compile(r'Pontos de Vida\s*\d+'),
    re.compile(r'Tesouro'),
)
# A linha de atributos é exigida à parte: sem ela dá para PREENCHER os campos do
# livro, mas não dá para AUDITAR os seis atributos.
RE_TEM_ATRIBUTOS = re.compile(r'\bFor\s*[+–−-]?[\d—]')


def bloco_completo(corpo: list[str]) -> bool:
    texto = junta(corpo)
    return all(r.search(texto) for r in EXIGIDAS)


def auditar() -> dict:
    cat = json.load(open(BESTIARIO))
    porNome = {chave(m['name']): m for m in cat}
    bs = blocos_por_nome(set(porNome))
    confiaveis, sem_atributos, suspeitos, truncados = {}, {}, {}, []
    for k, m in porNome.items():
        b = bs.get(k)
        if not b or not bloco_completo(b['corpo']):
            suspeitos[m['name']] = 'bloco incompleto' if b else 'sem bloco'
            continue
        lido = le(b)
        # As âncoras: se elas não batem, o bloco é de outra criatura.
        if lido.get('defesa') != m['defesa'] or lido.get('hp') != m['hp']:
            suspeitos[m['name']] = (
                f"âncora não bate (Defesa {lido.get('defesa')}/{m['defesa']}, "
                f"PV {lido.get('hp')}/{m['hp']})")
            continue
        confiaveis[m['name']] = (m, lido)
        if not RE_TEM_ATRIBUTOS.search(junta(b['corpo'])):
            sem_atributos[m['name']] = 'linha de atributos noutra coluna'
        if b.get('truncado'):
            truncados.append(m['name'])
    return {'confiaveis': confiaveis, 'sem_atributos': sem_atributos,
            'suspeitos': suspeitos, 'truncados': truncados}


RE_PERICIAS_SOLTAS = re.compile(r'^Perícias\b')
RE_PM_SOLTO = re.compile(r'^PM\s+\d+\s*[;.]\s*')


def preenchimento(lido: dict) -> dict:
    """Os campos que o verbete passa a ter, lidos do bloco (ALE-151).

    `skills` e `equipamento` entram SEMPRE, vazios quando o livro não dá — a
    ausência do campo diria "não sei", e aqui se sabe: o bloco foi lido inteiro
    e não tinha. `pm` é o contrário: a linha "Pontos de Mana" só existe em
    conjurador, e um zero diria "tem mana e está sem", que é outro estado.
    """
    novo: dict = {
        'skills': lido.get('skills', []),
        'equipamento': lido.get('equipamento', ''),
    }
    for campo in ('iniciativa', 'percepcao', 'tesouro'):
        if campo in lido:
            novo[campo] = lido[campo]
    if 'pm' in lido:
        novo['pm'] = lido['pm']
    return novo


def limpa_habilidades(habilidades: list[str]) -> list[str]:
    """Tira das habilidades o que virou CAMPO.

    Duas coisas foram parar aqui na importação. As perícias, como a frase
    "Perícias: Furtividade +5." em 37 verbetes — agora são `skills`, e manter a
    frase seria a mesma informação em dois lugares, livre para divergir. E os
    pontos de mana, grudados noutra coisa: o Centauro Xamã tinha
    "PM 20; Medo de Altura.", que são duas informações diferentes na mesma
    frase. O `PM 20` vira campo e "Medo de Altura." continua sendo o que é.
    """
    fora = []
    for h in habilidades:
        if RE_PERICIAS_SOLTAS.match(h):
            continue
        resto = RE_PM_SOLTO.sub('', h).strip()
        if resto:
            fora.append(resto[0].upper() + resto[1:])
    return fora


def relatorio(r: dict) -> list[tuple[str, int, str, object, object]]:
    """As correções que a auditoria propõe, em ordem de página."""
    fora = []
    for nome, (m, lido) in r['confiaveis'].items():
        for c in CONFERIDOS:
            if c in lido and lido[c] != m[c]:
                fora.append((nome, m['bookPage'], c, m[c], lido[c]))
    return sorted(fora, key=lambda x: (x[1], x[0], x[2]))


def aplicar(correcoes, r: dict) -> None:
    cat = json.load(open(BESTIARIO))
    porNome = {m['name']: m for m in cat}
    for nome, _p, campo, _de, para in correcoes:
        porNome[nome][campo] = para
    for nome, (_m, lido) in r['confiaveis'].items():
        alvo = porNome[nome]
        campos = preenchimento(lido)
        # Não reescreve o que ela SABE que não conseguiu ler inteiro: o tesouro
        # do Reishid é o valor mais a descrição da adaga, sete linhas. Ela
        # avisa e deixa a mão decidir — sobrescrever seria trocar um texto
        # conferido por um pela metade a cada execução.
        if nome in r['truncados']:
            campos.pop('tesouro', None)
        alvo.update(campos)
        alvo['specialAbilities'] = limpa_habilidades(alvo['specialAbilities'])
        # `treasureXp` sai: era `nd * 1000` em TODOS os 80, a mesma conta do
        # `xpForNd`, e o nome mentia — não tem relação com o Tesouro do livro,
        # que é "Nenhum"/"Metade"/"Padrão" e agora tem campo próprio.
        alvo.pop('treasureXp', None)
    # Mesmo formato do arquivo: compacto, numa linha. Reindentar trocaria 80
    # verbetes por 80 verbetes no diff e esconderia as correções.
    with open(BESTIARIO, 'w') as f:
        json.dump(cat, f, ensure_ascii=False, separators=(',', ':'))


if __name__ == '__main__':
    ap = argparse.ArgumentParser()
    ap.add_argument('--aplicar', action='store_true',
                    help='escreve as correções em bestiary.json')
    args = ap.parse_args()

    r = auditar()
    print(f"blocos confiáveis: {len(r['confiaveis'])}/{len(r['confiaveis']) + len(r['suspeitos'])}")
    for nome, motivo in r['suspeitos'].items():
        print(f'  SUSPEITO {nome}: {motivo} — confira à mão')
    for nome in r['truncados']:
        print(f'  TESOURO TRUNCADO {nome}: a frase passa de {LINHAS_DE_TESOURO} linhas — confira à mão')
    for nome, motivo in r['sem_atributos'].items():
        print(f'  SEM AUDITORIA DE ATRIBUTOS {nome}: {motivo} — os outros campos valem')
    correcoes = relatorio(r)
    print(f'\ncorreções propostas: {len(correcoes)}')
    for nome, p, campo, de, para in correcoes:
        print(f'  p{p:3d} {nome:32s} {campo:14s} {de!r:>6} -> {para!r}')
    faltando = {
        campo: [n for n, (_m, l) in r['confiaveis'].items() if campo not in l]
        for campo in ('iniciativa', 'percepcao', 'tesouro')
    }
    for campo, quem in faltando.items():
        if quem:
            print(f'\n{campo}: {len(quem)} sem valor no bloco — {quem[:6]}')

    if args.aplicar:
        aplicar(correcoes, r)
        print(f'\naplicadas em {BESTIARIO}')
