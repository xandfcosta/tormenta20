import { Type } from 'class-transformer';
import {
  ArrayMinSize,
  ArrayUnique,
  IsArray,
  IsBoolean,
  IsIn,
  IsInt,
  IsNumber,
  IsObject,
  IsOptional,
  IsString,
  Max,
  MaxLength,
  Min,
  MinLength,
  ValidateNested,
} from 'class-validator';
import {
  ATTRIBUTE_KEYS,
  CLASSES,
  EXPERTISE_NAMES,
  GODS,
  ORIGINS,
  RACES,
  SIZES,
} from '../t20-constants';

export class CharacterClassEntryDto {
  @IsIn(CLASSES as readonly string[])
  className!: string;

  @IsInt()
  @Min(1)
  @Max(20)
  level!: number;
}

/**
 * Attribute choices for the primary race, needed to derive the racial mod from
 * BASE attributes (floating +1 placement; subrace ascendência). Stored as JSON
 * on the character; the sheet applies race once from these.
 */
export class RaceAttributeChoicesDto {
  @IsOptional()
  @IsArray()
  @IsString({ each: true })
  floatingPicks?: string[];

  @IsOptional()
  @IsString()
  @MaxLength(40)
  ascendencia?: string;
}

/** One opted-in secondary race (GM-negotiated) + its attribute choices. */
export class SecondaryRaceChoiceDto {
  @IsIn(RACES as readonly string[])
  race!: string;

  @IsOptional()
  @IsArray()
  @IsString({ each: true })
  floatingPicks?: string[];

  @IsOptional()
  @IsString()
  @MaxLength(40)
  ascendencia?: string;
}

export class CreateCharacterDto {
  @IsString()
  @MinLength(1)
  @MaxLength(80)
  name!: string;

  // Homebrew allows multiple races; only the first (primary) is mechanical —
  // secondary races are GM-negotiated flavor. `races[0]` drives the sheet.
  @IsArray()
  @ArrayMinSize(1)
  @ArrayUnique()
  @IsIn(RACES as readonly string[], { each: true })
  races!: string[];

  @IsIn(ORIGINS as readonly string[])
  origin!: string;

  @IsArray()
  @ArrayMinSize(1)
  @ValidateNested({ each: true })
  @Type(() => CharacterClassEntryDto)
  classes!: CharacterClassEntryDto[];

  @IsOptional()
  @IsIn(GODS as readonly string[])
  god?: string;

  @IsInt() @Min(1) @Max(9999) hpMax!: number;
  @IsInt() @Min(0) @Max(9999) hpCurrent!: number;
  @IsInt() @Min(0) @Max(9999) mpMax!: number;
  @IsInt() @Min(0) @Max(9999) mpCurrent!: number;

  @IsInt() @Min(-5) @Max(10) strength!: number;
  @IsInt() @Min(-5) @Max(10) dexterity!: number;
  @IsInt() @Min(-5) @Max(10) constitution!: number;
  @IsInt() @Min(-5) @Max(10) intelligence!: number;
  @IsInt() @Min(-5) @Max(10) wisdom!: number;
  @IsInt() @Min(-5) @Max(10) charisma!: number;

  @IsIn(SIZES as readonly string[])
  size!: string;

  @IsInt() @Min(0) @Max(120) displacement!: number;

  // Optional creation-time ability choices — validated structurally here and
  // against the t20-data catalog in the service. Absent = character starts
  // with empty picks (the sheet's pendências flow then catches them).
  @IsOptional()
  @IsArray()
  @ArrayUnique()
  @IsString({ each: true })
  classPowers?: string[];

  @IsOptional()
  @IsArray()
  @ArrayUnique()
  @IsString({ each: true })
  originChoices?: string[];

  @IsOptional()
  @IsObject()
  classChoices?: Record<string, ClassChoiceBlobDto>;

  @IsOptional()
  @IsArray()
  @ArrayUnique()
  @IsString({ each: true })
  trainedExpertises?: string[];

  @IsOptional()
  @IsObject()
  powerChoices?: Record<string, string[]>;

  @IsOptional()
  @ValidateNested()
  @Type(() => RaceAttributeChoicesDto)
  raceAttributeChoices?: RaceAttributeChoicesDto;

  @IsOptional()
  @IsArray()
  @ValidateNested({ each: true })
  @Type(() => SecondaryRaceChoiceDto)
  secondaryRaceChoices?: SecondaryRaceChoiceDto[];
}

/**
 * Apply a spell's structured buff to a character as a scoped ActiveEffect.
 * `spellId` must reference a SPELL_CATALOG entry that carries a `buff` block;
 * `scope` overrides the spell's default scope (else the buff's `defaultScope`).
 */
export class ApplyEffectDto {
  @IsString()
  @MinLength(1)
  spellId!: string;

  @IsOptional()
  @IsIn(['scene', 'day'])
  scope?: 'scene' | 'day';
}

export class UpdateVitalsDto {
  @IsOptional()
  @IsInt()
  @Min(0)
  @Max(9999)
  hpCurrent?: number;

  @IsOptional()
  @IsInt()
  @Min(0)
  @Max(9999)
  mpCurrent?: number;
}

export class CreateExpertiseDto {
  @IsString()
  @MinLength(1)
  @MaxLength(40)
  name!: string;

  @IsIn(ATTRIBUTE_KEYS as readonly string[])
  attribute!: string;
}

const EQUIPPED_VALUES = ['vested', 'wielded', 'wielded2'] as const;
type EquippedValue = (typeof EQUIPPED_VALUES)[number];

export class CreateItemDto {
  @IsOptional()
  @IsString()
  @MaxLength(80)
  catalogId?: string;

  @IsOptional()
  @IsString()
  @MinLength(1)
  @MaxLength(80)
  name?: string;

  @IsInt()
  @Min(1)
  @Max(9999)
  quantity!: number;

  @IsOptional()
  @IsNumber()
  @Min(0.5)
  @Max(9999)
  slots?: number;

  @IsOptional()
  @IsIn(EQUIPPED_VALUES as readonly string[])
  equipped?: EquippedValue;

  @IsOptional()
  @IsArray()
  @ArrayUnique()
  @IsString({ each: true })
  improvements?: string[];

  @IsOptional()
  @IsString()
  material?: string;
}

export class UpdateItemDto {
  @IsOptional()
  @IsString()
  @MinLength(1)
  @MaxLength(80)
  name?: string;

  @IsOptional()
  @IsInt()
  @Min(1)
  @Max(9999)
  quantity?: number;

  @IsOptional()
  @IsNumber()
  @Min(0.5)
  @Max(9999)
  slots?: number;

  @IsOptional()
  equipped?: EquippedValue | null;

  @IsOptional()
  @IsArray()
  @ArrayUnique()
  @IsString({ each: true })
  improvements?: string[];

  @IsOptional()
  material?: string | null;
}

export class UpdateExpertiseDto {
  @IsIn(EXPERTISE_NAMES as readonly string[])
  name!: string;

  @IsOptional()
  @IsIn(ATTRIBUTE_KEYS as readonly string[])
  attribute?: string;

  @IsOptional()
  @IsBoolean()
  trained?: boolean;
}

export class ConsumeItemDto {
  @IsOptional()
  @IsInt()
  @Min(0)
  hpRolled?: number;

  @IsOptional()
  @IsInt()
  @Min(0)
  mpRolled?: number;
}

export class UpdateProficienciesDto {
  @IsArray()
  @ArrayUnique()
  @IsString({ each: true })
  proficiencies!: string[];
}

export class UpdateLevelDto {
  @IsInt()
  @Min(1)
  @Max(20)
  level!: number;
}

export class UpdateClassLevelDto {
  @IsIn(CLASSES as readonly string[])
  className!: string;

  @IsInt()
  @Min(1)
  @Max(20)
  level!: number;
}

/**
 * Per-class choice blob — devoto/caminho slugs keyed by className.
 * Validated at service layer against the t20-data catalog (caminhos +
 * deuses) since the legal value set depends on the chosen class.
 */
export class ClassChoiceBlobDto {
  @IsOptional()
  @IsString()
  @MaxLength(40)
  devoto?: string;

  @IsOptional()
  @IsString()
  @MaxLength(40)
  caminho?: string;
}

export class UpdateAbilityChoicesDto {
  @IsOptional()
  @IsArray()
  @ArrayUnique()
  @IsString({ each: true })
  raceAbilityChoices?: string[];

  @IsOptional()
  @IsArray()
  @ArrayUnique()
  @IsString({ each: true })
  originChoices?: string[];

  @IsOptional()
  @IsArray()
  @ArrayUnique()
  @IsString({ each: true })
  classPowers?: string[];

  @IsOptional()
  @IsObject()
  classChoices?: Record<string, ClassChoiceBlobDto>;

  @IsOptional()
  @IsObject()
  powerChoices?: Record<string, string[]>;
}
