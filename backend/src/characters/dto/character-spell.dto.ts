import { Type } from 'class-transformer';
import {
  ArrayMaxSize,
  IsArray,
  IsBoolean,
  IsInt,
  IsOptional,
  IsString,
  MaxLength,
  Min,
  MinLength,
  ValidateNested,
} from 'class-validator';

export class LearnSpellDto {
  @IsString()
  @MinLength(1)
  @MaxLength(64)
  catalogSpellId!: string;
}

export class SetSpellPreparedDto {
  @IsBoolean()
  prepared!: boolean;
}

export class CastAugmentPickDto {
  @IsInt()
  @Min(0)
  augmentIndex!: number;

  @IsInt()
  @Min(1)
  stacks!: number;
}

export class CastSpellDto {
  @IsOptional()
  @IsArray()
  @ArrayMaxSize(16)
  @ValidateNested({ each: true })
  @Type(() => CastAugmentPickDto)
  augments?: CastAugmentPickDto[];
}
