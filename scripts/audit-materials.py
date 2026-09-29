#!/usr/bin/env python3
"""Confere os MATERIAIS ESPECIAIS do catálogo contra o livro (ALE-415).

Por que ele não existia, e o que sobreviveu por isso
-----------------------------------------------------
O `audit-equipment.py` confere PREÇO por tabela do capítulo 3, e os seis
materiais saem dele na lista de "NÃO MEDIDOS" — eles não estão nas tabelas que
ele lê. A REGRA de um material nunca foi medida por nada, e o resultado foram
cinco dos seis errados: um inventando bônus que a página não tem, um com o fato
de OUTRO material, um com o número e a nota discordando entre si, e dois com
metade do verbete faltando.

As duas âncoras
---------------
* a **Tabela 3-9** (p167), que é uma MATRIZ — seis materiais em coluna, cinco
  tipos de item em linha, trinta células de preço;
* o **verbete** de cada material (p166-167), que é um título sozinho na linha
  seguido de prosa com as metades rotuladas: `Arma.`, `Armadura e Escudo.`,
  `Esotérico.`.

A matriz é o instrumento mais forte desta seção porque ela tem DENOMINADOR: são
trinta células, e duas delas são travessão (a madeira Tollon não vira armadura).
Um leitor que perca uma coluna devolve cinco materiais com nomes certos e preços
certos — e trinta é o único número que não fecha.
"""
import json
import pathlib
import re
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import t20pdf as P

OFFSET_DO_PDF = 6
PAGINA_DA_TABELA = 173
PAGINAS_DOS_VERBETES = (172, 173)

# A Tabela 3-9 ocupa o rodapé da p167. O `y` do título dela é o piso.
TOPO_DA_TABELA = 590.0
X_DO_TIPO_DE_ITEM = 79.4
FOLGA = 3.0

# As cinco linhas da matriz, na ordem impressa. Elas são o vocabulário do
# `appliesTo`, e é por isso que a tradução mora aqui e não no leitor.
TIPOS_DE_ITEM = {
    'Arma': 'weapon',
    'Armadura leve': 'armor',
    'Armadura pesada': 'armor',
    'Escudo': 'shield',
    'Esotéricos': 'esoteric',
}

RE_PRECO = re.compile(r'^\+ T\$ ([\d.]+)$')
SEM_PRECO = '—'


def matriz_de_precos(pagina: int):
    """A Tabela 3-9 como {(material, tipo de item): preço}, e os nomes em ordem.

    As colunas se acham pela linha `Arma`, que é a única sem travessão: os x das
    seis células dela são as âncoras, e toda outra célula vai para a mais
    próxima. Achar coluna pelo CABEÇALHO não serve — "Aço-Rubi Adamante" chega
    num `<line>` só, com os dois nomes colados.
    """
    linhas = [(x, y, t) for x, y, t in P.linhas_com_coordenada(pagina) if y > TOPO_DA_TABELA]
    tipos = {y: t for x, y, t in linhas if abs(x - X_DO_TIPO_DE_ITEM) < FOLGA and t in TIPOS_DE_ITEM}
    if not tipos:
        raise SystemExit('não achei nenhuma linha da Tabela 3-9')

    y_da_arma = next(y for y, t in tipos.items() if t == 'Arma')
    colunas = sorted(x for x, y, t in linhas
                     if abs(y - y_da_arma) < P.MESMA_LINHA and RE_PRECO.match(t))

    nomes = nomes_das_colunas(pagina, colunas)
    precos = {}
    for x, y, texto in linhas:
        if y not in tipos or x < min(colunas) - 30:
            continue
        casa = RE_PRECO.match(texto)
        if not casa and texto != SEM_PRECO:
            continue
        perto = min(range(len(colunas)), key=lambda i: abs(colunas[i] - x))
        valor = None if texto == SEM_PRECO else int(casa.group(1).replace('.', ''))
        precos[(nomes[perto], tipos[y])] = valor
    return nomes, precos, len(tipos)


def nomes_das_colunas(pagina: int, colunas: list[float]) -> list[str]:
    """Os seis nomes do cabeçalho, lidos PALAVRA a palavra.

    Por linha não dá: "Aço-Rubi Adamante" vem num elemento só, e "Gelo Eterno"
    vem quebrado em duas alturas. Cada palavra tem x próprio, e é ele que diz a
    coluna — o mesmo motivo pelo qual o `palavras_da_pagina` existe.
    """
    cabecalho = [(x, y, t) for x, y, t in P.palavras_da_pagina(pagina)
                 if TOPO_DA_TABELA < y < min(colunas) * 0 + 620 and x > X_DO_TIPO_DE_ITEM + 40]
    partes: dict[int, list[tuple[float, str]]] = {}
    for x, y, texto in cabecalho:
        perto = min(range(len(colunas)), key=lambda i: abs(colunas[i] - x))
        partes.setdefault(perto, []).append((y, texto))
    return [' '.join(t for _, t in sorted(partes.get(i, [])))
            for i in range(len(colunas))]


# As METADES que o verbete rotula. Elas são a âncora 2, e a escolha tem razão:
# capturar "do título até o próximo título" engole o boxe "Fabricando Itens
# Superiores", que divide a coluna com o Mitral e tem x IDÊNTICO ao da primeira
# linha de parágrafo — 22 linhas sobre preço de fabricação entrariam na regra do
# mitral e inflariam a cobertura dele com palavras de outro assunto.
#
# As metades também são o que se compara: a prosa de sabor ("o mitral é prateado
# e mais leve que aço") não é regra, e medi-la seria responder outra pergunta.
RE_METADE = re.compile(
    r'^(Arma|Armadura|Escudo|Esotérico|Armadura e Escudo|Escudo e Esotérico)\. (.+)$')

# Um TÍTULO de material é uma linha curta, sem ponto final, NO X DO CORPO. As
# duas condições são necessárias e nenhuma é decorativa.
#
# O x separa o título de SEÇÃO do título de BOXE: o boxe "Fabricando Itens
# Superiores" mora no x da primeira linha de parágrafo (326 = 309 + 17), e na
# ordem de leitura ele cai ENTRE o título "Mitral" (coluna esquerda) e as
# metades do Mitral (coluna direita). Sem o recorte, o sexto material saía
# chamado "Fabricando Itens Superiores", com as três metades certas e a
# contagem certa — foi a outra âncora que denunciou.
#
# E "ter metade rotulada abaixo" é o que separa material de título de seção:
# "Materiais Especiais" também é uma linha curta no x do corpo. Isso é regra
# ESTRUTURAL, e não a lista de nomes copiada da outra âncora — copiar faria as
# duas deixarem de se conferir.
RE_TITULO = re.compile(r'^[A-ZÁÉÍÓÚÂÊÔÃÕÇ][\wçãõáéíóúâêô-]*(?: [\wçãõáéíóúâêô-]+){0,2}$')


def verbetes(paginas):
    """{material: {parte: regra}}, lido BLOCO a bloco.

    A parte é `abertura` mais as metades rotuladas do livro. A abertura entra
    porque DUAS regras moram nela e em nenhuma metade: *"Itens de mitral ocupam
    –1 espaço (mínimo 1)"* e o *"–2 em perícias baseadas em Carisma"* da matéria
    vermelha. Um leitor que só colhesse as metades perderia as duas em silêncio.

    E é a abertura que obriga a ler por BLOCO e não por linha: o boxe
    "Fabricando Itens Superiores" divide a coluna com o Mitral, e 22 linhas
    sobre preço de fabricação entrariam na abertura dele. O bloco separa — a
    prosa do verbete começa em x=309 e a do boxe em x=326.
    """
    achados: dict[str, dict[str, str]] = {}
    pagina_de: dict[str, int] = {}
    titulo, parte, corpo = None, None, []

    def fecha():
        if titulo and parte and corpo:
            achados.setdefault(titulo, {})[parte] = P.junta(corpo)

    for pagina in paginas:
        for x0, texto_das_linhas in blocos_em_ordem(pagina):
            rotulada = RE_METADE.match(texto_das_linhas[0])
            # O bloco fora das colunas de corpo é BOXE, e não verbete — a menos
            # que ele seja uma metade rotulada de uma linha só, que nasce no x
            # da primeira linha de parágrafo (a do Adamante e a do Gelo eterno).
            if not rotulada and not e_coluna_de_corpo(pagina, x0):
                continue
            for texto in texto_das_linhas:
                casa = RE_METADE.match(texto)
                if casa:
                    fecha()
                    parte, corpo = casa.group(1), [casa.group(2)]
                elif RE_TITULO.match(texto) and not texto.endswith('.'):
                    fecha()
                    titulo, parte, corpo = texto, ABERTURA, []
                    pagina_de[titulo] = pagina - OFFSET_DO_PDF
                elif parte is not None:
                    corpo.append(texto)
    fecha()
    return ({nome: partes for nome, partes in achados.items()
             if any(p != ABERTURA for p in partes)}, pagina_de)


ABERTURA = 'abertura'
_colunas_por_pagina: dict[int, list[float]] = {}


def e_coluna_de_corpo(pagina: int, x: float) -> bool:
    if pagina not in _colunas_por_pagina:
        _colunas_por_pagina[pagina] = P.colunas(list(P.blocos_da_pagina(pagina)))
    return any(abs(x - coluna) < FOLGA for coluna in _colunas_por_pagina[pagina])


def blocos_em_ordem(pagina: int):
    """(xMin, [linhas]) de cada bloco, na ordem de leitura: coluna, depois y."""
    blocos = list(P.blocos_da_pagina(pagina))
    inicios = P.colunas(blocos)
    ordenados = sorted(blocos, key=lambda b: (P.coluna_de(inicios, b[0]), b[2]))
    for x0, _x1, y, linhas in ordenados:
        if pagina == PAGINA_DA_TABELA and y > TOPO_DA_TABELA:
            continue
        limpas = [t for t in P.sem_lixo(linhas) if t.strip()]
        if limpas:
            yield x0, limpas


PISO_DA_COBERTURA = 0.90
CATALOGO = P.RAIZ / 'engine-go/domain/catalog/data/items.json'
CATEGORIA = 'material'


def do_catalogo() -> dict:
    itens = json.loads(CATALOGO.read_text())
    return {P.chave(i['name']): i for i in itens if i.get('category') == CATEGORIA}


def main() -> int:
    nomes, precos, linhas = matriz_de_precos(PAGINA_DA_TABELA)
    dos_verbetes, paginas = verbetes(PAGINAS_DOS_VERBETES)
    partes = sum(len(p) for p in dos_verbetes.values())
    celulas_vazias = sum(1 for v in precos.values() if v is None)

    print(f'AS DUAS ÂNCORAS  Tabela 3-9: {len(nomes)} materiais x {linhas} tipos de item '
          f'= {len(precos)} células ({celulas_vazias} em travessão)')
    print(f'                 verbetes: {len(dos_verbetes)}, {partes} partes rotuladas')

    # AS DUAS SE CONFEREM ANTES DE O CATÁLOGO SER MENCIONADO. Foi assim que o
    # boxe "Fabricando Itens Superiores" foi pego roubando o nome do Mitral: a
    # contagem batia em seis e o sexto nome era o do boxe.
    por_chave = {P.chave(n): n for n in nomes}
    so_na_tabela = sorted(set(por_chave) - {P.chave(n) for n in dos_verbetes})
    so_no_verbete = sorted({P.chave(n) for n in dos_verbetes} - set(por_chave))
    if so_na_tabela or so_no_verbete:
        print(f'  AS DUAS DISCORDAM — só na tabela: {so_na_tabela}; só no verbete: {so_no_verbete}')
        print('  É defeito de LEITURA, e ele vem antes de qualquer correção de catálogo.')
        return 1
    if len(precos) != len(nomes) * linhas:
        print(f'  A MATRIZ NÃO FECHA: {len(precos)} células para {len(nomes)}x{linhas}')
        return 1
    print(f'                 as duas dizem os MESMOS {len(nomes)} nomes')

    catalogo = do_catalogo()
    falhas = []
    print(f'\nCATÁLOGO         {len(catalogo)} entradas em `{CATEGORIA}`')
    com_modelo, sem_modelo, menor = 0, [], (2.0, '')

    for nome in nomes:
        entrada = catalogo.get(P.chave(nome))
        if entrada is None:
            falhas.append(f'{nome}: está na Tabela 3-9 e não está no catálogo')
            continue
        do_livro = dos_verbetes[por_verbete(dos_verbetes, nome)]

        quanto, juntas, total = P.cobertura(' '.join(do_livro.values()),
                                            entrada.get('description', ''))
        menor = min(menor, (quanto, nome))
        if quanto < PISO_DA_COBERTURA:
            falhas.append(
                f'{nome}: a descrição repete {juntas} das {total} palavras de conteúdo do '
                f'verbete ({quanto:.0%}), abaixo do piso de {PISO_DA_COBERTURA:.0%}')

        # A MATRIZ INTEIRA, célula a célula. É o instrumento mais forte da
        # seção: o preço de um material depende do TIPO DE ITEM, e um catálogo
        # com um número só não tem como estar certo para os cinco.
        do_catalogo_precos = entrada.get('priceByItem') or {}
        for tipo_impresso, tipo in TIPOS_DE_ITEM.items():
            esperado = precos[(nome, tipo_impresso)]
            chave = chave_de_preco(tipo_impresso)
            tem = do_catalogo_precos.get(chave, 'AUSENTE')
            if esperado is None and chave in do_catalogo_precos:
                falhas.append(f'{nome}: o catálogo dá preço de {tipo_impresso} e a '
                              f'Tabela 3-9 traz travessão — este material não vira isso')
            elif esperado is not None and tem != esperado:
                falhas.append(f'{nome}/{tipo_impresso}: o catálogo diz {tem} e a '
                              f'Tabela 3-9 diz {esperado}')

        familias = {TIPOS_DE_ITEM[t] for t in TIPOS_DE_ITEM if precos[(nome, t)] is not None}
        if set(entrada.get('appliesTo') or []) != familias:
            falhas.append(f'{nome}: o catálogo aceita {sorted(entrada.get("appliesTo") or [])} '
                          f'e a Tabela 3-9 dá preço para {sorted(familias)}')
        if entrada.get('price') != precos[(nome, 'Arma')]:
            falhas.append(f'{nome}: o `price` diz {entrada.get("price")} e o de ARMA é '
                          f'{precos[(nome, "Arma")]}. O campo é um número só e a Tabela 3-9 é '
                          f'uma matriz: ele carrega a coluna de arma, e a verdade inteira '
                          f'mora no `priceByItem`')
        if entrada.get('bookPage') != paginas[por_verbete(dos_verbetes, nome)]:
            falhas.append(f'{nome}: o catálogo cita a p{entrada.get("bookPage")} e o verbete '
                          f'está na p{paginas[por_verbete(dos_verbetes, nome)]}')

        # O DENOMINADOR É POR METADE, e não por material. Um material com
        # modificador não está pronto por isso: o mitral aplica a margem de
        # ameaça e não aplica o limite de Destreza, e "tem modifiers?" daria
        # verde sobre os dois.
        #
        # As duas listas são PERMITIDOS, e a união delas tem de dar exatamente
        # as partes que o verbete tem. Uma parte pode estar nas DUAS — é o caso
        # do "Armadura e Escudo" do mitral, em que metade entra e metade não.
        aplicado = set(entrada.get('modeled') or [])
        calado = set(entrada.get('unmodeled') or {})
        do_livro_partes = set(do_livro)
        if aplicado | calado != do_livro_partes:
            faltam = sorted(do_livro_partes - (aplicado | calado))
            sobram = sorted((aplicado | calado) - do_livro_partes)
            falhas.append(
                f'{nome}: as partes declaradas não fecham com o verbete — '
                f'sem declaração: {faltam}; declaradas e inexistentes: {sobram}')
        if aplicado and not entrada.get('modifiers'):
            falhas.append(f'{nome}: diz aplicar {sorted(aplicado)} e não tem modificador nenhum')
        if entrada.get('modifiers') and not aplicado:
            falhas.append(f'{nome}: tem modificador e não diz QUAL metade ele aplica')
        for parte, motivo in (entrada.get('unmodeled') or {}).items():
            if not motivo:
                falhas.append(f'{nome}/{parte}: está na lista do que não foi modelado sem motivo')
        com_modelo += len(aplicado)
        for parte in sorted(calado):
            sem_modelo.append((f'{nome}/{parte}', entrada['unmodeled'][parte]))

    print(f'  metades que o motor APLICA: {com_modelo}')
    print(f'  metades que ele NÃO aplica: {len(sem_modelo)}, e cada uma diz por quê:')
    for nome, motivo in sem_modelo:
        print(f'    {nome:36s} {motivo[:92]}')
    print(f'\n  cobertura mínima: {menor[0]:.0%} ({menor[1]})')
    print(f'\nFALHAS: {len(falhas)}   (medidos: {len(nomes)} materiais, {len(precos)} células '
          f'de preço, {partes} partes de verbete)')
    for f in falhas:
        print(f'  {f}')
    return 1 if falhas else 0


def por_verbete(dos_verbetes: dict, nome: str) -> str:
    """O nome como o VERBETE o escreve. A tabela e o verbete divergem na caixa —
    "Gelo Eterno" contra "Gelo eterno" —, e as duas grafias são do livro."""
    return next(n for n in dos_verbetes if P.chave(n) == P.chave(nome))


def chave_de_preco(tipo_impresso: str) -> str:
    """O tipo de item como chave de JSON, em inglês: a fronteira é o campo."""
    return {'Arma': 'weapon', 'Armadura leve': 'armorLight',
            'Armadura pesada': 'armorHeavy', 'Escudo': 'shield',
            'Esotéricos': 'esoteric'}[tipo_impresso]


if __name__ == '__main__':
    sys.exit(main())
