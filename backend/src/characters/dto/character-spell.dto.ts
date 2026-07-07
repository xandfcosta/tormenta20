import { IsBoolean, IsString, MaxLength, MinLength } from 'class-validator';

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
