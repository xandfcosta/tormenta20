import {
  BadRequestException,
  ForbiddenException,
  Injectable,
  NotFoundException,
} from '@nestjs/common';
import { PrismaService } from '../prisma/prisma.service';
import {
  ConsumeItemDto,
  CreateCharacterDto,
  CreateExpertiseDto,
  CreateItemDto,
  UpdateAbilityChoicesDto,
  UpdateClassLevelDto,
  UpdateExpertiseDto,
  UpdateItemDto,
  UpdateLevelDto,
  UpdateProficienciesDto,
  UpdateVitalsDto,
} from './dto/character.dto';
import { EXPERTISE_NAMES, EXPERTISES } from './t20-constants';
import {
  PROFICIENCY_CATEGORIES,
  characterProficiencies,
  familyFor,
  getCatalogItem,
  isCatalogId,
  type ProficiencyCategory,
} from '@tormenta20/t20-data';

function assertCharacterRules(dto: CreateCharacterDto): void {
  const fieldErrors: Record<string, string[]> = {};
  if (dto.hpCurrent > dto.hpMax) {
    fieldErrors.hpCurrent = ['HP current cannot exceed HP max'];
  }
  if (dto.mpCurrent > dto.mpMax) {
    fieldErrors.mpCurrent = ['MP current cannot exceed MP max'];
  }
  const seen = new Set<string>();
  dto.classes.forEach((entry, index) => {
    if (seen.has(entry.className)) {
      fieldErrors[`classes.${index}.className`] = [
        `Class "${entry.className}" already added — combine levels in one entry instead`,
      ];
    } else {
      seen.add(entry.className);
    }
  });
  if (Object.keys(fieldErrors).length === 0) return;
  throw new BadRequestException({
    statusCode: 400,
    error: 'Bad Request',
    message: 'Validation failed',
    fieldErrors,
  });
}

const characterInclude = {
  races: { select: { race: true } },
  classes: { select: { className: true, level: true } },
  expertises: {
    select: {
      name: true,
      attribute: true,
      trained: true,
      custom: true,
    },
    orderBy: { name: 'asc' },
  },
  items: {
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
    orderBy: { id: 'asc' },
  },
  activeEffects: {
    select: {
      id: true,
      catalogId: true,
      scope: true,
      modifiers: true,
      createdAt: true,
    },
    orderBy: { id: 'asc' },
  },
} as const;

function assertSlotsMultiple(slots: number): void {
  if (!Number.isFinite(slots) || !Number.isInteger(slots * 2)) {
    throw new BadRequestException({
      statusCode: 400,
      error: 'Bad Request',
      message: 'Validation failed',
      fieldErrors: { slots: ['Slots must be a multiple of 0.5'] },
    });
  }
}

/**
 * Validate that every improvement / material id refers to a known catalog
 * overlay AND that its `appliesTo` family list includes the base item's
 * family. Custom items (no catalogId) cannot have overlays.
 */
function assertOverlaysCompatible(
  catalogId: string | null | undefined,
  improvements: string[] | undefined,
  material: string | null | undefined,
): void {
  const hasOverlays =
    (improvements && improvements.length > 0) ||
    (material !== undefined && material !== null && material !== '');
  if (!hasOverlays) return;
  if (!catalogId) {
    throw new BadRequestException({
      statusCode: 400,
      error: 'Bad Request',
      message: 'Validation failed',
      fieldErrors: {
        improvements: ['Custom items cannot have overlays'],
      },
    });
  }
  const base = getCatalogItem(catalogId);
  if (!base) {
    throw new BadRequestException(`Unknown base item "${catalogId}"`);
  }
  if (
    base.category === 'consumable' ||
    base.category === 'meal' ||
    base.category === 'catalyst' ||
    base.category === 'animal' ||
    base.category === 'vehicle'
  ) {
    throw new BadRequestException({
      statusCode: 400,
      error: 'Bad Request',
      message: 'Validation failed',
      fieldErrors: {
        improvements: [`${base.category} items cannot have overlays`],
      },
    });
  }
  const family = familyFor(base);
  const fieldErrors: Record<string, string[]> = {};
  for (const id of improvements ?? []) {
    const overlay = getCatalogItem(id);
    if (!overlay || overlay.category !== 'improvement') {
      (fieldErrors.improvements ??= []).push(`Unknown improvement "${id}"`);
      continue;
    }
    if (!overlay.appliesTo?.some((f) => f === 'any' || f === family)) {
      (fieldErrors.improvements ??= []).push(
        `"${overlay.name}" does not apply to ${family}`,
      );
    }
  }
  if (material) {
    const overlay = getCatalogItem(material);
    if (!overlay || overlay.category !== 'material') {
      (fieldErrors.material ??= []).push(`Unknown material "${material}"`);
    } else if (!overlay.appliesTo?.some((f) => f === 'any' || f === family)) {
      (fieldErrors.material ??= []).push(
        `"${overlay.name}" does not apply to ${family}`,
      );
    }
  }
  if (Object.keys(fieldErrors).length > 0) {
    throw new BadRequestException({
      statusCode: 400,
      error: 'Bad Request',
      message: 'Validation failed',
      fieldErrors,
    });
  }
}

@Injectable()
export class CharactersService {
  constructor(private readonly prisma: PrismaService) {}

  list(ownerId: number) {
    return this.prisma.character.findMany({
      where: { ownerId },
      orderBy: { updatedAt: 'desc' },
      include: characterInclude,
    });
  }

  async findOne(ownerId: number, id: number) {
    const character = await this.prisma.character.findUnique({
      where: { id },
      include: characterInclude,
    });
    if (!character) {
      throw new NotFoundException(`Character ${id} not found`);
    }
    if (character.ownerId !== ownerId) {
      throw new ForbiddenException(`Character ${id} belongs to another user`);
    }
    return this.backfillProficiencies(character);
  }

  /**
   * Existing rows predating the proficiencies column have an empty JSON array.
   * On first read, derive class defaults and persist so the UI sees a non-empty
   * set without users having to open the panel.
   */
  private async backfillProficiencies<
    T extends {
      id: number;
      proficiencies: string;
      classes: { className: string }[];
    },
  >(character: T): Promise<T> {
    if (character.proficiencies !== '[]') return character;
    if (character.classes.length === 0) return character;
    const classNames = character.classes.map((c) => c.className);
    const defaults = characterProficiencies(classNames)
      .filter((p) => p.granted)
      .map((p) => p.category);
    const next = JSON.stringify(defaults);
    await this.prisma.character.update({
      where: { id: character.id },
      data: { proficiencies: next },
    });
    return { ...character, proficiencies: next };
  }

  create(ownerId: number, dto: CreateCharacterDto) {
    assertCharacterRules(dto);
    const totalLevel = dto.classes.reduce((sum, c) => sum + c.level, 0);
    const classNames = dto.classes.map((c) => c.className);
    const grantedDefaults = characterProficiencies(classNames)
      .filter((p) => p.granted)
      .map((p) => p.category);
    return this.prisma.character.create({
      data: {
        ownerId,
        name: dto.name,
        origin: dto.origin,
        god: dto.god ?? null,
        level: totalLevel,
        hpMax: dto.hpMax,
        hpCurrent: dto.hpCurrent,
        mpMax: dto.mpMax,
        mpCurrent: dto.mpCurrent,
        strength: dto.strength,
        dexterity: dto.dexterity,
        constitution: dto.constitution,
        intelligence: dto.intelligence,
        wisdom: dto.wisdom,
        charisma: dto.charisma,
        size: dto.size,
        displacement: dto.displacement,
        proficiencies: JSON.stringify(grantedDefaults),
        races: { create: dto.races.map((race) => ({ race })) },
        classes: {
          create: dto.classes.map((c) => ({
            className: c.className,
            level: c.level,
          })),
        },
        expertises: {
          create: EXPERTISES.map((e) => ({
            name: e.name,
            attribute: e.attribute,
          })),
        },
      },
      include: characterInclude,
    });
  }

  async updateAbilityChoices(
    ownerId: number,
    characterId: number,
    dto: UpdateAbilityChoicesDto,
  ) {
    await this.findOne(ownerId, characterId);
    const data: {
      raceAbilityChoices?: string;
      originChoices?: string;
      classPowers?: string;
    } = {};
    if (dto.raceAbilityChoices !== undefined) {
      data.raceAbilityChoices = JSON.stringify(dto.raceAbilityChoices);
    }
    if (dto.originChoices !== undefined) {
      data.originChoices = JSON.stringify(dto.originChoices);
    }
    if (dto.classPowers !== undefined) {
      data.classPowers = JSON.stringify(dto.classPowers);
    }
    if (Object.keys(data).length === 0) {
      throw new BadRequestException('No fields to update');
    }
    return this.prisma.character.update({
      where: { id: characterId },
      data,
      include: characterInclude,
    });
  }

  async updateLevel(
    ownerId: number,
    characterId: number,
    dto: UpdateLevelDto,
  ) {
    await this.findOne(ownerId, characterId);
    return this.prisma.character.update({
      where: { id: characterId },
      data: { level: dto.level },
      include: characterInclude,
    });
  }

  /**
   * Per-class level update. Patches the CharacterClass row matching
   * `dto.className` and recomputes the character's total level as the sum of
   * class levels — the engine uses both: per-class level for power gating,
   * total level for ½-level expertise bonus and PV/PM scaling.
   */
  async updateClassLevel(
    ownerId: number,
    characterId: number,
    dto: UpdateClassLevelDto,
  ) {
    const character = await this.findOne(ownerId, characterId);
    const entry = character.classes.find((c) => c.className === dto.className);
    if (!entry) {
      throw new BadRequestException({
        statusCode: 400,
        error: 'Bad Request',
        message: `Character does not have class "${dto.className}"`,
        fieldErrors: { className: [`Class not on character`] },
      });
    }
    const otherSum = character.classes
      .filter((c) => c.className !== dto.className)
      .reduce((sum, c) => sum + c.level, 0);
    const newTotal = otherSum + dto.level;
    if (newTotal > 20) {
      throw new BadRequestException({
        statusCode: 400,
        error: 'Bad Request',
        message: `Total level ${newTotal} exceeds 20`,
        fieldErrors: { level: [`Sum of class levels capped at 20`] },
      });
    }
    return this.prisma.character.update({
      where: { id: characterId },
      data: {
        level: newTotal,
        classes: {
          update: {
            where: {
              characterId_className: {
                characterId,
                className: dto.className,
              },
            },
            data: { level: dto.level },
          },
        },
      },
      include: characterInclude,
    });
  }

  async updateProficiencies(
    ownerId: number,
    characterId: number,
    dto: UpdateProficienciesDto,
  ) {
    await this.findOne(ownerId, characterId);
    const valid = new Set<string>(PROFICIENCY_CATEGORIES);
    const fieldErrors: Record<string, string[]> = {};
    for (const cat of dto.proficiencies) {
      if (!valid.has(cat)) {
        (fieldErrors.proficiencies ??= []).push(`Unknown category "${cat}"`);
      }
    }
    if (Object.keys(fieldErrors).length > 0) {
      throw new BadRequestException({
        statusCode: 400,
        error: 'Bad Request',
        message: 'Validation failed',
        fieldErrors,
      });
    }
    const dedup = [...new Set<ProficiencyCategory>(
      dto.proficiencies as ProficiencyCategory[],
    )];
    return this.prisma.character.update({
      where: { id: characterId },
      data: { proficiencies: JSON.stringify(dedup) },
      include: characterInclude,
    });
  }

  async updateVitals(
    ownerId: number,
    characterId: number,
    dto: UpdateVitalsDto,
  ) {
    const character = await this.findOne(ownerId, characterId);
    const data: { hpCurrent?: number; mpCurrent?: number } = {};
    const fieldErrors: Record<string, string[]> = {};
    if (dto.hpCurrent !== undefined) {
      if (dto.hpCurrent > character.hpMax) {
        fieldErrors.hpCurrent = ['HP current cannot exceed HP max'];
      } else {
        data.hpCurrent = dto.hpCurrent;
      }
    }
    if (dto.mpCurrent !== undefined) {
      if (dto.mpCurrent > character.mpMax) {
        fieldErrors.mpCurrent = ['MP current cannot exceed MP max'];
      } else {
        data.mpCurrent = dto.mpCurrent;
      }
    }
    if (Object.keys(fieldErrors).length > 0) {
      throw new BadRequestException({
        statusCode: 400,
        error: 'Bad Request',
        message: 'Validation failed',
        fieldErrors,
      });
    }
    if (Object.keys(data).length === 0) {
      throw new BadRequestException('No fields to update');
    }
    return this.prisma.character.update({
      where: { id: characterId },
      data,
      include: characterInclude,
    });
  }

  async addCustomExpertise(
    ownerId: number,
    characterId: number,
    dto: CreateExpertiseDto,
  ) {
    await this.findOne(ownerId, characterId);
    const trimmed = dto.name.trim();
    const fieldErrors: Record<string, string[]> = {};
    if (!trimmed) {
      fieldErrors.name = ['Name is required'];
    } else if ((EXPERTISE_NAMES as readonly string[]).includes(trimmed)) {
      fieldErrors.name = [`"${trimmed}" is a builtin expertise — pick another name`];
    } else {
      const exists = await this.prisma.characterExpertise.findUnique({
        where: { characterId_name: { characterId, name: trimmed } },
        select: { id: true },
      });
      if (exists) {
        fieldErrors.name = [`Expertise "${trimmed}" already exists`];
      }
    }
    if (Object.keys(fieldErrors).length > 0) {
      throw new BadRequestException({
        statusCode: 400,
        error: 'Bad Request',
        message: 'Validation failed',
        fieldErrors,
      });
    }
    return this.prisma.characterExpertise.create({
      data: {
        characterId,
        name: trimmed,
        attribute: dto.attribute,
        trained: true,
        custom: true,
      },
      select: {
        name: true,
        attribute: true,
        trained: true,
        custom: true,
      },
    });
  }

  async deleteExpertise(
    ownerId: number,
    characterId: number,
    name: string,
  ) {
    await this.findOne(ownerId, characterId);
    const row = await this.prisma.characterExpertise.findUnique({
      where: { characterId_name: { characterId, name } },
      select: { id: true, custom: true },
    });
    if (!row) {
      throw new NotFoundException(`Expertise "${name}" not found`);
    }
    if (!row.custom) {
      throw new BadRequestException(
        `Expertise "${name}" is builtin and cannot be removed`,
      );
    }
    await this.prisma.characterExpertise.delete({ where: { id: row.id } });
    return { name };
  }

  async updateExpertise(
    ownerId: number,
    characterId: number,
    dto: UpdateExpertiseDto,
  ) {
    await this.findOne(ownerId, characterId);
    const data: { attribute?: string; trained?: boolean } = {};
    if (dto.attribute !== undefined) data.attribute = dto.attribute;
    if (dto.trained !== undefined) data.trained = dto.trained;
    if (Object.keys(data).length === 0) {
      throw new BadRequestException('No fields to update');
    }
    return this.prisma.characterExpertise.update({
      where: { characterId_name: { characterId, name: dto.name } },
      data,
      select: {
        name: true,
        attribute: true,
        trained: true,
        custom: true,
      },
    });
  }

  async addItem(
    ownerId: number,
    characterId: number,
    dto: CreateItemDto,
  ) {
    await this.findOne(ownerId, characterId);

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

    if (dto.equipped) {
      await this.assertEquipLimits(characterId, null, dto.equipped);
    }

    assertOverlaysCompatible(dto.catalogId, dto.improvements, dto.material);

    return this.prisma.characterItem.create({
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
  }

  async updateItem(
    ownerId: number,
    characterId: number,
    itemId: number,
    dto: UpdateItemDto,
  ) {
    await this.findOne(ownerId, characterId);
    if (dto.slots !== undefined) assertSlotsMultiple(dto.slots);
    const item = await this.prisma.characterItem.findUnique({
      where: { id: itemId },
      select: { id: true, characterId: true, equipped: true },
    });
    if (!item || item.characterId !== characterId) {
      throw new NotFoundException(`Item ${itemId} not found`);
    }
    if (dto.equipped !== undefined && dto.equipped !== item.equipped) {
      if (dto.equipped) {
        await this.assertEquipLimits(characterId, itemId, dto.equipped);
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
      const fullItem = await this.prisma.characterItem.findUnique({
        where: { id: itemId },
        select: { catalogId: true },
      });
      assertOverlaysCompatible(
        fullItem?.catalogId ?? null,
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
    return this.prisma.characterItem.update({
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
  }

  async deleteItem(
    ownerId: number,
    characterId: number,
    itemId: number,
  ) {
    await this.findOne(ownerId, characterId);
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

  private async assertEquipLimits(
    characterId: number,
    excludeItemId: number | null,
    newEquipped: 'vested' | 'wielded' | 'wielded2',
  ) {
    const items = await this.prisma.characterItem.findMany({
      where: {
        characterId,
        ...(excludeItemId !== null ? { id: { not: excludeItemId } } : {}),
        equipped: { not: null },
      },
      select: { equipped: true },
    });
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
    const character = await this.findOne(ownerId, characterId);
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

    const vitalsPatch: { hpCurrent?: number; mpCurrent?: number } = {};
    if (spec.instant?.hp) {
      const gain =
        dto.hpRolled ?? rollAverage(spec.instant.hp.dice, spec.instant.hp.bonus);
      vitalsPatch.hpCurrent = Math.min(
        character.hpMax,
        character.hpCurrent + gain,
      );
    }
    if (spec.instant?.mp) {
      const gain =
        dto.mpRolled ?? rollAverage(spec.instant.mp.dice, spec.instant.mp.bonus);
      vitalsPatch.mpCurrent = Math.min(
        character.mpMax,
        character.mpCurrent + gain,
      );
    }

    await this.prisma.$transaction(async (tx) => {
      if (spec.scope !== 'instant' && spec.modifiers && spec.modifiers.length > 0) {
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
      if (Object.keys(vitalsPatch).length > 0) {
        await tx.character.update({
          where: { id: characterId },
          data: vitalsPatch,
        });
      }
    });

    return this.findOne(ownerId, characterId);
  }

  async removeActiveEffect(
    ownerId: number,
    characterId: number,
    effectId: number,
  ) {
    await this.findOne(ownerId, characterId);
    const effect = await this.prisma.activeEffect.findUnique({
      where: { id: effectId },
      select: { id: true, characterId: true },
    });
    if (!effect || effect.characterId !== characterId) {
      throw new NotFoundException(`Active effect ${effectId} not found`);
    }
    await this.prisma.activeEffect.delete({ where: { id: effectId } });
    return { id: effectId };
  }

  async endScene(ownerId: number, characterId: number) {
    await this.findOne(ownerId, characterId);
    await this.prisma.activeEffect.deleteMany({
      where: { characterId, scope: 'scene' },
    });
    return this.findOne(ownerId, characterId);
  }

  async endDay(ownerId: number, characterId: number) {
    await this.findOne(ownerId, characterId);
    await this.prisma.activeEffect.deleteMany({
      where: { characterId, scope: { in: ['scene', 'day'] } },
    });
    return this.findOne(ownerId, characterId);
  }
}

/**
 * Deterministic average roll used for AI-rolled consumables when the client
 * does not supply a player-rolled value. Treats `NdF` as `N * (F+1)/2`.
 * Accepts plain integers ('0', '5') as a flat-bonus shorthand.
 */
function rollAverage(dice: string, bonus = 0): number {
  const trimmed = dice.trim();
  if (trimmed === '' || trimmed === '0') return bonus;
  const flat = Number(trimmed);
  if (!Number.isNaN(flat)) return flat + bonus;
  const match = /^(\d+)d(\d+)$/i.exec(trimmed);
  if (!match) {
    throw new BadRequestException(`Invalid dice expression: "${dice}"`);
  }
  const n = Number(match[1]);
  const f = Number(match[2]);
  return Math.floor(n * (f + 1) / 2) + bonus;
}
