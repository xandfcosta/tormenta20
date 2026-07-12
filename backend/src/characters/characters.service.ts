import {
  BadRequestException,
  ForbiddenException,
  Injectable,
  NotFoundException,
} from '@nestjs/common';
import { PrismaService } from '../prisma/prisma.service';
import {
  CreateCharacterDto,
  UpdateAbilityChoicesDto,
  UpdateClassLevelDto,
  UpdateLevelDto,
  UpdateProficienciesDto,
  UpdateVitalsDto,
} from './dto/character.dto';
import { EXPERTISES } from './t20-constants';
import {
  PROFICIENCY_CATEGORIES,
  characterProficiencies,
  type ProficiencyCategory,
} from '@tormenta20/t20-data';
import {
  assertCharacterRules,
  sanitizeClassChoices,
} from './characters.helpers';
import { computeSheetForRow } from './character-sheet.mapper';
import type { ComputedSheet } from '@tormenta20/t20-data';

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
  spells: {
    select: {
      id: true,
      catalogSpellId: true,
      prepared: true,
      learnedAt: true,
    },
    orderBy: { learnedAt: 'asc' },
  },
} as const;

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

  /**
   * Access guard for a single character — used by every read + mutation
   * path. Granted to the **owner** or to a **campaign GM** (a user who
   * owns a campaign the character is a member of). The GM path is what
   * lets a mestre edit a player's sheet, push loot, or peek stats in the
   * session drawer; the player still owns the row. There is no
   * whole-character delete endpoint, so GM reach can't destroy the sheet.
   */
  async findOne(userId: number, id: number) {
    const character = await this.prisma.character.findUnique({
      where: { id },
      include: characterInclude,
    });
    if (!character) {
      throw new NotFoundException(`Character ${id} not found`);
    }
    if (
      character.ownerId !== userId &&
      !(await this.isCampaignGmForCharacter(userId, id))
    ) {
      throw new ForbiddenException(`Character ${id} belongs to another user`);
    }
    return this.backfillProficiencies(character);
  }

  /** True when `userId` owns a campaign this character has joined —
   * i.e. is the character's GM in at least one table. */
  private async isCampaignGmForCharacter(
    userId: number,
    characterId: number,
  ): Promise<boolean> {
    const membership = await this.prisma.campaignMember.findFirst({
      where: { characterId, campaign: { ownerId: userId } },
      select: { id: true },
    });
    return membership !== null;
  }

  /**
   * Lightweight owner-only guard — throws if the character is missing or
   * not owned by `userId`, without loading the full aggregate. For hot
   * paths (live vitals edits) that only need the ownership rule, not the
   * whole sheet. Encapsulates "who owns this character" so callers (the
   * WS gateway) don't re-query `ownerId` themselves.
   */
  async assertOwner(userId: number, characterId: number): Promise<void> {
    const character = await this.prisma.character.findUnique({
      where: { id: characterId },
      select: { ownerId: true },
    });
    if (!character) {
      throw new NotFoundException(`Character ${characterId} not found`);
    }
    if (character.ownerId !== userId) {
      throw new ForbiddenException(
        `Character ${characterId} belongs to another user`,
      );
    }
  }

  /**
   * Read variant that attaches a derived sheet (`computed`) produced by
   * the t20-data orchestrator. Handy for read-only views (character
   * sheet UI, initiative panels) that should show total attributes,
   * defense and saves without the caller re-implementing the rules.
   *
   * v1 scope: attributes + vitals + defense base + saves + movement.
   * Skills, equipment and active-effect stacking are placeholders until
   * the follow-up phases map those pieces of DB state onto the input.
   */
  async findOneWithComputed(
    ownerId: number,
    id: number,
  ): Promise<Awaited<ReturnType<typeof this.findOne>> & { computed: ComputedSheet }> {
    const character = await this.findOne(ownerId, id);
    const computed = computeSheetForRow(character);
    return { ...character, computed };
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
      classChoices?: string;
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
    if (dto.classChoices !== undefined) {
      data.classChoices = JSON.stringify(sanitizeClassChoices(dto.classChoices));
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
    /* Ownership gate before the tx so we don't waste a transaction on
     * an unauthorized caller. The class + total re-check happen INSIDE
     * the tx so a parallel updateClassLevel on another class can't
     * slip past the L20 cap. */
    await this.findOne(ownerId, characterId);
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    return this.prisma.$transaction(async (tx: any) => {
      const classes = (await tx.characterClass.findMany({
        where: { characterId },
        select: { className: true, level: true },
      })) as { className: string; level: number }[];
      const entry = classes.find((c) => c.className === dto.className);
      if (!entry) {
        throw new BadRequestException({
          statusCode: 400,
          error: 'Bad Request',
          message: `Character does not have class "${dto.className}"`,
          fieldErrors: { className: [`Class not on character`] },
        });
      }
      const otherSum = classes
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
      return tx.character.update({
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

  /**
   * Two concurrent updateVitals calls (same player, two tabs; two GMs
   * co-editing) both read `hpMax` outside the write path and both write
   * `data.hpCurrent`, silently clobbering one write against the other.
   * BC3 fix: wrap the max lookup + validation + write in one tx so a
   * mid-flight max change is seen. Same pattern the future WS→DB
   * write-through (Persistence P3) must adopt — write path already
   * consolidated here so P3 only needs to plug in.
   */
  async updateVitals(
    ownerId: number,
    characterId: number,
    dto: UpdateVitalsDto,
  ) {
    await this.findOne(ownerId, characterId);
    if (dto.hpCurrent === undefined && dto.mpCurrent === undefined) {
      throw new BadRequestException('No fields to update');
    }
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    return this.prisma.$transaction(async (tx: any) => {
      const fresh = (await tx.character.findUnique({
        where: { id: characterId },
        select: { hpMax: true, mpMax: true },
      })) as { hpMax: number; mpMax: number } | null;
      if (!fresh) {
        throw new NotFoundException(`Character ${characterId} not found`);
      }
      const data: { hpCurrent?: number; mpCurrent?: number } = {};
      const fieldErrors: Record<string, string[]> = {};
      if (dto.hpCurrent !== undefined) {
        if (dto.hpCurrent > fresh.hpMax) {
          fieldErrors.hpCurrent = ['HP current cannot exceed HP max'];
        } else {
          data.hpCurrent = dto.hpCurrent;
        }
      }
      if (dto.mpCurrent !== undefined) {
        if (dto.mpCurrent > fresh.mpMax) {
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
      return tx.character.update({
        where: { id: characterId },
        data,
        include: characterInclude,
      });
    });
  }

}

