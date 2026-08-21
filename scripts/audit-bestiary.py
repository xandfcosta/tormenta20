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
                corpo = corpo[:j + 1]
                break
        out.setdefault(nome, {'pagina': ls[i][0], 'corpo': corpo})
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
RE_TESOURO = re.compile(r'Tesouro\s+([^.]+)\.?')


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


def le(bloco: dict) -> dict:
    texto = ' '.join(bloco['corpo'])
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
        campos['pericias'] = m.group(1).strip().rstrip('.')
    if m := ultimo(RE_EQUIP, texto):
        campos['equipamento'] = m.group(1).strip().rstrip('.')
    if m := ultimo(RE_TESOURO, texto):
        campos['tesouro'] = m.group(1).strip()
    return campos



CONFERIDOS = ('hp', 'defesa', 'fortitude', 'reflexos', 'vontade',
              'forca', 'destreza', 'constituicao', 'inteligencia', 'sabedoria', 'carisma')


EXIGIDAS = (
    re.compile(r'Iniciativa\s*[+–−-]?\d+'),
    re.compile(r'Defesa\s*\d+.*Fort.*Ref.*Von'),
    re.compile(r'Pontos de Vida\s*\d+'),
    re.compile(r'\bFor\s*[+–−-]?[\d—]'),
    re.compile(r'Tesouro'),
)


def bloco_completo(corpo: list[str]) -> bool:
    texto = ' '.join(corpo)
    return all(r.search(texto) for r in EXIGIDAS)


def auditar() -> dict:
    cat = json.load(open(BESTIARIO))
    porNome = {chave(m['name']): m for m in cat}
    bs = blocos_por_nome(set(porNome))
    confiaveis, suspeitos = {}, {}
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
    return {'confiaveis': confiaveis, 'suspeitos': suspeitos}




def relatorio(r: dict) -> list[tuple[str, int, str, object, object]]:
    """As correções que a auditoria propõe, em ordem de página."""
    fora = []
    for nome, (m, lido) in r['confiaveis'].items():
        for c in CONFERIDOS:
            if c in lido and lido[c] != m[c]:
                fora.append((nome, m['bookPage'], c, m[c], lido[c]))
    return sorted(fora, key=lambda x: (x[1], x[0], x[2]))


def aplicar(correcoes) -> None:
    cat = json.load(open(BESTIARIO))
    porNome = {m['name']: m for m in cat}
    for nome, _p, campo, _de, para in correcoes:
        porNome[nome][campo] = para
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
    correcoes = relatorio(r)
    print(f'\ncorreções propostas: {len(correcoes)}')
    for nome, p, campo, de, para in correcoes:
        print(f'  p{p:3d} {nome:32s} {campo:14s} {de!r:>6} -> {para!r}')
    if args.aplicar:
        aplicar(correcoes)
        print(f'\naplicadas em {BESTIARIO}')
