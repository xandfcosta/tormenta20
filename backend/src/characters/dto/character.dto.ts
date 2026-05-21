import { Type } from 'class-transformer';
import {
  ArrayMinSize,
  ArrayUnique,
  IsArray,
  IsBoolean,
  IsIn,
  IsInt,
  IsNumber,
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

export class CreateCharacterDto {
  @IsString()
  @MinLength(1)
  @MaxLength(80)
  name!: string;

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
}
