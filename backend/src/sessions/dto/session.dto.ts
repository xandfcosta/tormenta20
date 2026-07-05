import {
  IsInt,
  IsOptional,
  IsString,
  MaxLength,
  Min,
  MinLength,
} from 'class-validator';

export class CreateSessionDto {
  @IsInt()
  @Min(1)
  sessionNumber!: number;

  @IsOptional()
  @IsString()
  @MinLength(1)
  @MaxLength(200)
  title?: string;

  @IsOptional()
  @IsString()
  @MaxLength(10000)
  notes?: string;
}

export class UpdateSessionDto {
  @IsOptional()
  @IsInt()
  @Min(1)
  sessionNumber?: number;

  @IsOptional()
  @IsString()
  @MinLength(1)
  @MaxLength(200)
  title?: string;

  @IsOptional()
  @IsString()
  @MaxLength(10000)
  notes?: string;
}
