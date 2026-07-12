import {
  BadRequestException,
  Injectable,
  NotFoundException,
} from '@nestjs/common';
import { PrismaService } from '../prisma/prisma.service';
import {
  ConsumeItemDto,
  CreateItemDto,
  UpdateItemDto,
} from './dto/character.dto';
import { getCatalogItem, isCatalogId } from '@tormenta20/t20-data';
import {
  assertOverlaysCompatible,
  assertSlotsMultiple,
  rollAverage,
} from './characters.helpers';
import { isPrismaUniqueViolation } from '../common/prisma-errors';
import { CharactersService } from './characters.service';

/**
 * Inventory + equipment slice of the Character aggregate — add/update/
 * delete/consume items and the vested/wielded equip-cap invariants.
 * Split out of CharactersService (which was ~970 lines) to keep one
 * responsibility per module. Ownership is delegated to
 * `CharactersService.findOne` (owner-or-campaign-GM), so item writes
 * stay behind the same guard as the rest of the sheet.
 */
@Injectable()
export class CharacterItemsService {
  constructor(
    private readonly prisma: PrismaService,
    private readonly characters: CharactersService,
  ) {}

  async addItem(
    ownerId: number,
    characterId: number,
    dto: CreateItemDto,
  ) {
    await this.characters.findOne(ownerId, characterId);

    const fieldErrors: Record<string, string[]> = {};
    let resolvedName = dto.name?.trim() ?? '';
    let resolvedSlots = dto.slots;

    if (dto.catalogId) {
      if (!isCatalogId(dto.catalogId)) {
        fieldErrors.catalogId = [`Unknown catalog item "${dto.catalogId}"`];
      } else {
        const entry = getCatalogItem(dto.catalogId)!;
        if (!resolvedName) resolvedName = entry.name;
        if (resolvedSlots === undefined) resolvedSlots = entry.slots;
      }
    }

    if (!resolvedName) {
      fieldErrors.name = ['Name is required'];
    }
    if (resolvedSlots === undefined) {
      fieldErrors.slots = ['Slots is required for custom items'];
    } else if (
      !Number.isFinite(resolvedSlots) ||
      !Number.isInteger(resolvedSlots * 2)
    ) {
      fieldErrors.slots = ['Slots must be a multiple of 0.5'];
    }

    if (Object.keys(fieldErrors).length > 0) {
      throw new BadRequestException({
        statusCode: 400,
        error: 'Bad Request',
        message: 'Validation failed',
        fieldErrors,
      });
    }

    assertOverlaysCompatible(dto.catalogId, dto.improvements, dto.material);

    /* BI1: check + create inside one tx so a parallel equip can't slip
     * past the hand/vested limits. Without the tx, two concurrent
     * addItem calls both observe handsUsed=0 and both write wielded2 */
    return this.prisma.$transaction(async (tx) => {
      if (dto.equipped) {
        await this.assertEquipLimits(tx, characterId, null, dto.equipped);
      }
      return tx.characterItem.create({
        data: {
          characterId,
          catalogId: dto.catalogId ?? null,
          name: resolvedName,
          quantity: dto.quantity,
          slots: resolvedSlots!,
          equipped: dto.equipped ?? null,
          improvements: JSON.stringify(dto.improvements ?? []),
          material: dto.material ?? null,
        },
        select: {
          id: true,
          catalogId: true,
          name: true,
          quantity: true,
          slots: true,
          equipped: true,
          improvements: true,
          material: true,
        },
      });
    });
  }

  async updateItem(
    ownerId: number,
    characterId: number,
    itemId: number,
    dto: UpdateItemDto,
  ) {
    await this.characters.findOne(ownerId, characterId);
    if (dto.slots !== undefined) assertSlotsMultiple(dto.slots);
    /* BI1: fold the equipped-limit check + write into a single tx so
     * two concurrent equip calls can't both observe handsUsed=0 and
     * both commit `wielded2`. */
    return this.prisma.$transaction(async (tx) => {
      const item = await tx.characterItem.findUnique({
        where: { id: itemId },
        select: {
          id: true,
          characterId: true,
          equipped: true,
          catalogId: true,
        },
      });
      if (!item || item.characterId !== characterId) {
        throw new NotFoundException(`Item ${itemId} not found`);
      }
      if (dto.equipped !== undefined && dto.equipped !== item.equipped) {
        if (dto.equipped) {
          await this.assertEquipLimits(tx, characterId, itemId, dto.equipped);
        }
      }
      const data: {
        name?: string;
        quantity?: number;
        slots?: number;
        equipped?: string | null;
        improvements?: string;
        material?: string | null;
      } = {};
      if (dto.name !== undefined) data.name = dto.name.trim();
      if (dto.quantity !== undefined) data.quantity = dto.quantity;
      if (dto.slots !== undefined) data.slots = dto.slots;
      if (dto.equipped !== undefined) data.equipped = dto.equipped;
      if (dto.improvements !== undefined || dto.material !== undefined) {
        assertOverlaysCompatible(
          item.catalogId ?? null,
          dto.improvements,
          dto.material,
        );
        if (dto.improvements !== undefined) {
          data.improvements = JSON.stringify(dto.improvements);
        }
        if (dto.material !== undefined) data.material = dto.material;
      }
      if (Object.keys(data).length === 0) {
        throw new BadRequestException('No fields to update');
      }
      return tx.characterItem.update({
        where: { id: itemId },
        data,
        select: {
          id: true,
          catalogId: true,
          name: true,
          quantity: true,
          slots: true,
          equipped: true,
          improvements: true,
          material: true,
        },
      });
    });
  }

  async deleteItem(
    ownerId: number,
    characterId: number,
    itemId: number,
  ) {
    await this.characters.findOne(ownerId, characterId);
    const item = await this.prisma.characterItem.findUnique({
      where: { id: itemId },
      select: { id: true, characterId: true },
    });
    if (!item || item.characterId !== characterId) {
      throw new NotFoundException(`Item ${itemId} not found`);
    }
    await this.prisma.characterItem.delete({ where: { id: itemId } });
    return { id: itemId };
  }

  /**
   * Enforces vested (≤4) and wielded (≤2 hand-slots) totals for the
   * character. Reads through the passed `reader` — either the top-level
   * PrismaService or a `$transaction` client — so callers can wrap
   * check+update atomically (BI1 fix). Reading from the same client
   * the write will use is what closes the TOCTOU window; using the
   * global `this.prisma` here (as the original did) let two equip
   * calls both observe pre-write state.
   */
  private async assertEquipLimits(
    /* PrismaService and tx clients share the shape but their generated
     * types don't line up structurally — accept via a permissive
     * `PrismaLike` alias here to keep the check + write on the same
     * client without a full Prisma import.
     */
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    reader: any,
    characterId: number,
    excludeItemId: number | null,
    newEquipped: 'vested' | 'wielded' | 'wielded2',
  ) {
    const items = (await reader.characterItem.findMany({
      where: {
        characterId,
        ...(excludeItemId !== null ? { id: { not: excludeItemId } } : {}),
        equipped: { not: null },
      },
      select: { equipped: true },
    })) as { equipped: string | null }[];
    const vestedCount = items.filter((i) => i.equipped === 'vested').length;
    const handsUsed = items.reduce(
      (s, i) =>
        s +
        (i.equipped === 'wielded' ? 1 : i.equipped === 'wielded2' ? 2 : 0),
      0,
    );
    const incomingHands =
      newEquipped === 'wielded' ? 1 : newEquipped === 'wielded2' ? 2 : 0;
    const incomingVested = newEquipped === 'vested' ? 1 : 0;
    if (vestedCount + incomingVested > 4) {
      throw new BadRequestException({
        statusCode: 400,
        error: 'Bad Request',
        message: 'Validation failed',
        fieldErrors: {
          equipped: ['Limite de 4 itens vestidos atingido'],
        },
      });
    }
    if (handsUsed + incomingHands > 2) {
      throw new BadRequestException({
        statusCode: 400,
        error: 'Bad Request',
        message: 'Validation failed',
        fieldErrors: {
          equipped: ['Limite de 2 mãos atingido'],
        },
      });
    }
  }

  async consumeItem(
    ownerId: number,
    characterId: number,
    itemId: number,
    dto: ConsumeItemDto,
  ) {
    const character = await this.characters.findOne(ownerId, characterId);
    const item = character.items.find((i) => i.id === itemId);
    if (!item) {
      throw new NotFoundException(`Item ${itemId} not found`);
    }
    if (!item.catalogId) {
      throw new BadRequestException(
        `Item ${itemId} is custom and has no consumable spec`,
      );
    }
    const catalog = getCatalogItem(item.catalogId);
    if (!catalog?.consumable) {
      throw new BadRequestException(
        `Item "${item.name}" is not consumable`,
      );
    }
    if (item.quantity < 1) {
      throw new BadRequestException(`No remaining uses of "${item.name}"`);
    }

    const spec = catalog.consumable;

    if (spec.oncePerDay) {
      const existing = character.activeEffects.find(
        (e) => e.catalogId === catalog.id,
      );
      if (existing) {
        throw new BadRequestException({
          statusCode: 400,
          error: 'Bad Request',
          message: `"${catalog.name}" already active for the day`,
          fieldErrors: { catalogId: ['Apenas uma porção por dia'] },
        });
      }
    }

    /* Roll the instant gain outside the transaction so retries don't
     * bump the dice again — the roll is a single decision, not part of
     * the write-atomicity story. Actual clamp against hp/mp happens
     * against a *fresh* read inside the tx (BC1 audit finding). */
    const hpGain = spec.instant?.hp
      ? (dto.hpRolled ??
        rollAverage(spec.instant.hp.dice, spec.instant.hp.bonus))
      : null;
    const mpGain = spec.instant?.mp
      ? (dto.mpRolled ??
        rollAverage(spec.instant.mp.dice, spec.instant.mp.bonus))
      : null;

    try {
      await this.prisma.$transaction(async (tx) => {
        if (
          spec.scope !== 'instant' &&
          spec.modifiers &&
          spec.modifiers.length > 0
        ) {
          await tx.activeEffect.create({
            data: {
              characterId,
              catalogId: catalog.id,
              scope: spec.scope,
              modifiers: JSON.stringify(spec.modifiers),
            },
          });
        }
        if (item.quantity > 1) {
          await tx.characterItem.update({
            where: { id: itemId },
            data: { quantity: item.quantity - 1 },
          });
        } else {
          await tx.characterItem.delete({ where: { id: itemId } });
        }
        if (hpGain !== null || mpGain !== null) {
          /* Re-read hp/mp inside the tx so a parallel consume doesn't
           * clobber our gain. Without this, two concurrent potions
           * both observe the pre-write HP and both write "old + gain",
           * dropping one of the gains silently. */
          const fresh = await tx.character.findUnique({
            where: { id: characterId },
            select: {
              hpCurrent: true,
              hpMax: true,
              mpCurrent: true,
              mpMax: true,
            },
          });
          if (!fresh) return;
          const vitalsPatch: { hpCurrent?: number; mpCurrent?: number } = {};
          if (hpGain !== null) {
            vitalsPatch.hpCurrent = Math.min(
              fresh.hpMax,
              fresh.hpCurrent + hpGain,
            );
          }
          if (mpGain !== null) {
            vitalsPatch.mpCurrent = Math.min(
              fresh.mpMax,
              fresh.mpCurrent + mpGain,
            );
          }
          await tx.character.update({
            where: { id: characterId },
            data: vitalsPatch,
          });
        }
      });
    } catch (err) {
      /* P2002 = Prisma unique constraint violation. The DB rejected
       * because another consume of the same catalogId already produced
       * an ActiveEffect row for this (character, catalogId, scope) —
       * the app-level check missed it due to concurrency. Translate to
       * the same BadRequestException the pre-check would have raised. */
      if (isPrismaUniqueViolation(err)) {
        throw new BadRequestException({
          statusCode: 400,
          error: 'Bad Request',
          message: `"${catalog.name}" already active for the day`,
          fieldErrors: { catalogId: ['Apenas uma porção por dia'] },
        });
      }
      throw err;
    }

    return this.characters.findOne(ownerId, characterId);
  }
}
