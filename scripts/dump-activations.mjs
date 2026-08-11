/**
 * Regenerates `engine-go/catalog/data/activations.json` from t20-data, the
 * source of truth.
 *
 * The dump was hand-maintained, and it drifted: `ActivationScaling` used to
 * carry `maxStepsForLevel` as a FUNCTION, which JSON silently drops — the
 * served catalog reached both fronts without it and the sheet crashed on any
 * scaling stance. Generating the file removes the class of bug where the
 * catalog and the rules disagree because someone edited one of them.
 *
 * Usage: node scripts/dump-activations.mjs
 */
import { writeFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { ACTIVATION_SPECS } from '../t20-data/dist/power-activation.js'

const OUT = resolve(
  dirname(fileURLToPath(import.meta.url)),
  '../engine-go/catalog/data/activations.json',
)

/** Guards the whole point of this script: a function cannot cross the wire. */
function assertSerializable(specs) {
  for (const spec of specs) {
    for (const [key, value] of Object.entries(spec.scaling ?? {})) {
      if (typeof value === 'function') {
        throw new Error(
          `activation ${spec.id}: scaling.${key} is a function — the served catalog is JSON, so express the rule as data (see firstStepLevel/stepEveryLevels)`,
        )
      }
    }
  }
}

assertSerializable(ACTIVATION_SPECS)
writeFileSync(OUT, `${JSON.stringify(ACTIVATION_SPECS, null, 2)}\n`)
console.log(`activations.json: ${ACTIVATION_SPECS.length} specs`)
