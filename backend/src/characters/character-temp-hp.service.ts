import { BadRequestException, Injectable } from '@nestjs/common';
import { PrismaService } from '../prisma/prisma.service';
import { CharactersService } from './characters.service';
import {
  parseTempHpPools,
  planDamage,
  planPoolSupremacy,
  type DamageDrainStep,
  type DisplacedPool,
} from './temp-hp.helpers';

/** Delta row shape every effect mutation returns to the client cache.
 *  (Shared with CharacterEffectsService, which imports it from here.) */
export const ACTIVE_EFFECT_SELECT = {
  id: true,
  catalogId: true,
  scope: true,
  modifiers: true,
  createdAt: true,
} as const;

/** catalogId of the GM-entered ad-hoc pool (F3) — no power/spell behind it. */
export const MANUAL_TEMP_HP_CATALOG_ID = 'manual-temp-hp';

type ActiveEffectDelta = {
  id: number;
  catalogId: string;
  scope: string;
  modifiers: string;
  createdAt: Date;
};

/**
 * Result of applying a temp-PV pool under vale-o-maior (p256):
 *  - `superseded` — an existing pool is bigger or equal; nothing was written.
 *  - otherwise the upserted row + the pools it displaced (`removed: true`
 *    rows were deleted; kept rows had their tempHp amount zeroed).
 */
export type PoolApplyResult =
  | { superseded: true; keptEffectId: number; keptAmount: number }
  | { effect: ActiveEffectDelta; displaced: DisplacedPool[] };

/** `manualTempHp: 0` cleared the manual pool (ids for the client to drop). */
export type ManualPoolClearedResult = { cleared: true; removedEffectIds: number[] };

/** POST :id/damage response — hp after routing + per-pool drain deltas. */
export type ApplyDamageResult = {
  hpCurrent: number;
  tempHpRemaining: number;
  drained: DamageDrainStep[];
};

/**
 * Temp-PV pool lifecycle of the Character aggregate (livro p256: PV
 * temporários não acumulam — vale o maior). Owns the vale-o-maior apply used
 * by power grants (F1) and the manual GM pool (F3), plus the atomic damage
 * routing endpoint (F2). Authorization stays on `CharactersService.findOne`;
 * `applyPool`/`setManualPool` trust the caller already passed it.
 */
@Injectable()
export class CharacterTempHpService {
  constructor(
    private readonly prisma: PrismaService,
    private readonly characters: CharactersService,
  ) {}

  /**
   * Upsert a temp-PV pool under vale-o-maior, in one transaction: same
   * catalogId+scope replaces its own row (amount refreshed); a bigger
   * existing pool wins (superseded, no write); a winning new pool deletes
   * smaller pure pools and zeroes mixed ones.
   *
   * @example tempHp.applyPool(1, 'power', 'class.barbaro.alma-de-bronze', 'scene', 10)
   */
  applyPool(
    characterId: number,
    source: string,
    catalogId: string,
    scope: 'scene' | 'day',
    amount: number,
    note = 'PV temporários',
  ): Promise<PoolApplyResult> {
    const modifiers = JSON.stringify([
      { target: { k: 'tempHp' }, amount, bonusType: 'untyped', note },
    ]);
    return this.prisma.$transaction(async (tx) => {
      const rows = await tx.activeEffect.findMany({
        where: { characterId },
        select: { id: true, catalogId: true, scope: true, modifiers: true },
      });
      const plan = planPoolSupremacy(parseTempHpPools(rows), { catalogId, scope }, amount);
      if (plan.kind === 'superseded') {
        return {
          superseded: true as const,
          keptEffectId: plan.keptEffectId,
          keptAmount: plan.keptAmount,
        };
      }
      await this.displacePools(tx, plan);
      const effect = await tx.activeEffect.upsert({
        where: { characterId_catalogId_scope: { characterId, catalogId, scope } },
        create: { characterId, source, catalogId, scope, modifiers },
        update: { modifiers, source },
        select: ACTIVE_EFFECT_SELECT,
      });
      return { effect, displaced: plan.displaced };
    });
  }

  private async displacePools(
    tx: { activeEffect: PrismaService['activeEffect'] },
    plan: { zeroWrites: { effectId: number; modifiers: string }[]; deleteIds: number[] },
  ): Promise<void> {
    for (const zero of plan.zeroWrites) {
      await tx.activeEffect.update({
        where: { id: zero.effectId },
        data: { modifiers: zero.modifiers },
      });
    }
    if (plan.deleteIds.length > 0) {
      await tx.activeEffect.deleteMany({ where: { id: { in: plan.deleteIds } } });
    }
  }

  /**
   * Set (or clear, with 0) the GM-entered manual pool. Same vale-o-maior
   * path as power grants so the rule lives once. Caller (effects service)
   * already authorized.
   *
   * @example tempHp.setManualPool(1, 12) // rolled Fortificação, etc.
   */
  async setManualPool(
    characterId: number,
    amount: number,
    scope: 'scene' | 'day' = 'scene',
  ): Promise<PoolApplyResult | ManualPoolClearedResult> {
    if (!Number.isInteger(amount) || amount < 0) {
      throw new BadRequestException({
        statusCode: 400,
        error: 'Bad Request',
        message: `manualTempHp must be an integer >= 0 — got ${amount}`,
        fieldErrors: { manualTempHp: ['Informe um valor inteiro ≥ 0'] },
      });
    }
    if (amount === 0) return this.clearManualPool(characterId);
    return this.applyPool(
      characterId,
      'manual',
      MANUAL_TEMP_HP_CATALOG_ID,
      scope,
      amount,
      'PV temporários (manual)',
    );
  }

  private async clearManualPool(
    characterId: number,
  ): Promise<ManualPoolClearedResult> {
    const rows = await this.prisma.activeEffect.findMany({
      where: { characterId, catalogId: MANUAL_TEMP_HP_CATALOG_ID },
      select: { id: true },
    });
    if (rows.length > 0) {
      await this.prisma.activeEffect.deleteMany({
        where: { characterId, catalogId: MANUAL_TEMP_HP_CATALOG_ID },
      });
    }
    return { cleared: true, removedEffectIds: rows.map((r) => r.id) };
  }

  /**
   * Atomic damage application (F2): fresh-read hp + pools inside one
   * transaction, drain pools first (highest first), remainder lowers
   * hpCurrent with a floor of 0. Owner-or-GM guard via `findOne`.
   *
   * @example tempHp.applyDamage(userId, charId, 7) // { hpCurrent, tempHpRemaining, drained }
   */
  async applyDamage(
    callerId: number,
    characterId: number,
    amount: number,
  ): Promise<ApplyDamageResult> {
    if (!Number.isInteger(amount) || amount <= 0) {
      throw new BadRequestException({
        statusCode: 400,
        error: 'Bad Request',
        message: `Damage amount must be a positive integer — got ${amount}`,
        fieldErrors: { amount: ['Informe um dano inteiro ≥ 1'] },
      });
    }
    await this.characters.findOne(callerId, characterId);
    return this.prisma.$transaction(async (tx) => {
      const fresh = await tx.character.findUnique({
        where: { id: characterId },
        select: { hpCurrent: true },
      });
      const rows = await tx.activeEffect.findMany({
        where: { characterId },
        select: { id: true, catalogId: true, scope: true, modifiers: true },
      });
      const plan = planDamage(parseTempHpPools(rows), fresh?.hpCurrent ?? 0, amount);
      for (const update of plan.updates) {
        await tx.activeEffect.update({
          where: { id: update.effectId },
          data: { modifiers: update.modifiers },
        });
      }
      if (plan.deleteIds.length > 0) {
        await tx.activeEffect.deleteMany({ where: { id: { in: plan.deleteIds } } });
      }
      if (plan.hpCurrent !== (fresh?.hpCurrent ?? 0)) {
        await tx.character.update({
          where: { id: characterId },
          data: { hpCurrent: plan.hpCurrent },
        });
      }
      return {
        hpCurrent: plan.hpCurrent,
        tempHpRemaining: plan.tempHpRemaining,
        drained: plan.drained,
      };
    });
  }
}
