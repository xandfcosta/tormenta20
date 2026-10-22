#!/usr/bin/env python3
"""Confere as POÇÕES do catálogo contra a Tabela 8-12 (p341, ALE-451).

Uma poção é uma MAGIA ENGARRAFADA
----------------------------------
A tabela não tem verbete: ela é `d% | <nome da MAGIA> (tipo; aprimoramento) |
preço`, e a regra da poção É a da magia — *"poções e pergaminhos contêm o efeito
de uma magia. Quando são ativados, geram o efeito dessa magia e então
desaparecem"* (p341). Por isso este auditor não procura prosa: ele resolve cada
linha contra o CATÁLOGO DE MAGIAS, que já tem as 198 com círculo e
aprimoramentos.

O que o parêntese carrega, e são três coisas
---------------------------------------------
* o TIPO de entrega: *óleo* (afeta objeto) e *granada* (afeta área) — sem
  parêntese, é poção de beber;
* o APRIMORAMENTO que aquela linha compra: *"(aprimoramento para duração
  cena)"*, *"(granada; aprimoramento para 10d6 de dano)"*;
* o EFEITO JÁ CALCULADO, quando a magia é de cura: *"Curar Ferimentos (4d8+4
  PV)"* — quatro linhas da tabela são a mesma magia em degraus diferentes de
  aprimoramento.

O denominador é o mesmo das outras tabelas: o `d%` tila 1..100.
"""
import json
import pathlib
import re
import sys

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import t20pdf as P

PAGINA_DA_TABELA = 347
PAGINA_DO_LIVRO = 341
# A célula do d% é alinhada à DIREITA: "97-100" começa em 314,9 e "80" em
# 322,9. O x é FAIXA, e a borda de baixo custou a última linha da tabela.
X_DO_DADO = (313.0, 325.0)
X_DO_NOME = 349.8
X_DO_PRECO = (490.0, 502.0)
FOLGA = 3.0

RE_PRECO = re.compile(r'T\$\s*([\d.]+)')
# "Curar Ferimentos (4d8+4 PV)" → nome e parêntese. O parêntese pode ter duas
# coisas separadas por ponto e vírgula.
RE_CELULA = re.compile(r'^(.+?)(?:\s*\((.*)\))?$')
RE_CURA = re.compile(r'^(\d+d\d+)\+(\d+)\s*PV$')

CATALOGO = P.RAIZ / 'engine-go/domain/catalog/data'


def linhas_da_tabela():
    """As linhas da Tabela 8-12 — a LEITURA mora no `t20pdf`, como nas outras
    quatro tabelas de saque deste capítulo."""
    return P.linhas_de_tabela(PAGINA_DA_TABELA, X_DO_DADO, X_DO_NOME,
                              X_DO_PRECO, FOLGA)


def partes_da_celula(celula: str) -> dict:
    """{magia, tipo, aprimoramento, cura} do que a célula escreve."""
    casa = RE_CELULA.match(celula.strip())
    magia, dentro = casa.group(1).strip(), (casa.group(2) or '').strip()
    partes = {'magia': magia, 'tipo': 'pocao', 'aprimoramento': '', 'cura': None}
    for pedaco in (p.strip() for p in dentro.split(';') if p.strip()):
        if pedaco in ('óleo', 'granada'):
            partes['tipo'] = 'oleo' if pedaco == 'óleo' else 'granada'
        elif RE_CURA.match(pedaco):
            dado, bonus = RE_CURA.match(pedaco).groups()
            partes['cura'] = {'dice': dado, 'bonus': int(bonus)}
            partes['aprimoramento'] = pedaco
        else:
            partes['aprimoramento'] = pedaco
    return partes


def magias_por_nome():
    cru = json.loads((CATALOGO / 'spells.json').read_text())
    lista = cru if isinstance(cru, list) else list(cru.values())
    return {P.chave(m['name']): m for m in lista}


def pocoes_do_catalogo():
    itens = json.loads((CATALOGO / 'items.json').read_text())
    return [i for i in itens if (i.get('consumable') or {}).get('spell')]


def audita() -> int:
    print(f'══ TABELA 8-12 (poções, p{PAGINA_DO_LIVRO}) ' + '═' * 30)
    tabela = list(linhas_da_tabela())
    buraco = P.faixas_que_nao_fecham(tabela)
    if buraco:
        print(f'A TABELA NÃO FECHA: {buraco}')
        print('Nada do que vem abaixo vale — o auditor leu a página errado.')
        return 1

    magias = magias_por_nome()
    por_id = {i['id']: i for i in pocoes_do_catalogo()}
    falhas = []
    print(f'{"d%":>7} {"magia":28} {"tipo":8} {"preço":>7}  catálogo')
    for inicio, fim, celula, preco in tabela:
        partes = partes_da_celula(celula)
        magia = magias.get(P.chave(partes['magia']))
        faixa = f'{inicio}-{fim}' if fim != inicio else str(inicio)
        if magia is None:
            falhas.append(f'{celula}: o catálogo de magias não tem {partes["magia"]!r}')
            print(f'{faixa:>7} {partes["magia"]:28} {partes["tipo"]:8} {preco:>7}  — MAGIA AUSENTE')
            continue
        entrada = casa_a_pocao(por_id, magia['id'], partes, preco)
        print(f'{faixa:>7} {magia["id"]:28} {partes["tipo"]:8} {preco:>7}  '
              f'{entrada["id"] if entrada else "— AUSENTE"}')
        if entrada:
            falhas += confere(entrada, magia, partes, preco)

    transcritas = sum(1 for linha in tabela
                      if casa_a_pocao(por_id, magias.get(P.chave(partes_da_celula(linha[2])['magia']), {}).get('id'),
                                      partes_da_celula(linha[2]), linha[3]))
    print(f'\n{transcritas}/{len(tabela)} transcritas')
    for falha in falhas:
        print(f'  ✗ {falha}')
    return len(falhas)


def casa_a_pocao(por_id, spell_id, partes, preco):
    """A entrada do catálogo que é ESTA linha da tabela.

    A magia não basta para achar: a Curar Ferimentos tem QUATRO linhas e a Arma
    Mágica tem duas. O que as separa é o preço, que é o degrau de aprimoramento.
    """
    if not spell_id:
        return None
    for entrada in por_id.values():
        consumivel = entrada['consumable']
        if consumivel['spell'] == spell_id and entrada.get('price') == preco:
            return entrada
    return None


def confere(entrada, magia, partes, preco) -> list:
    falhas = []
    consumivel = entrada['consumable']
    if consumivel.get('kind') != partes['tipo']:
        falhas.append(f'{entrada["id"]}: o catálogo diz tipo {consumivel.get("kind")!r} '
                      f'e a tabela diz {partes["tipo"]!r}')
    if entrada.get('bookPage') != PAGINA_DO_LIVRO:
        falhas.append(f'{entrada["id"]}: cita p{entrada.get("bookPage")} e a tabela '
                      f'está na p{PAGINA_DO_LIVRO}')
    if partes['aprimoramento'] and not consumivel.get('augment'):
        falhas.append(f'{entrada["id"]}: a tabela escreve "{partes["aprimoramento"]}" '
                      f'e o catálogo não diz aprimoramento nenhum')
    if not partes['aprimoramento'] and consumivel.get('augment'):
        falhas.append(f'{entrada["id"]}: o catálogo diz o aprimoramento '
                      f'{consumivel["augment"]!r} e a tabela não escreve nenhum')
    # A CURA JÁ CALCULADA tem de bater dado a dado: é o único número que a
    # tabela imprime, e é o que a ficha vai somar no PV de alguém.
    if partes['cura']:
        instante = ((consumivel.get('instant') or {}).get('hp') or {})
        if instante.get('dice') != partes['cura']['dice'] or \
                instante.get('bonus', 0) != partes['cura']['bonus']:
            falhas.append(f'{entrada["id"]}: a tabela diz {partes["cura"]["dice"]}+'
                          f'{partes["cura"]["bonus"]} PV e o catálogo diz {instante}')
    return falhas


def main() -> int:
    return min(1, audita())


if __name__ == '__main__':
    raise SystemExit(main())
