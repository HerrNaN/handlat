import z from "zod";

import type { Language } from "../i18n";
import { en } from "../i18n/en";

export const UnitSchema = z.enum([
  "st",
  "pkt",
  "kg",
  "g",
  "l",
  "dl",
  "cl",
  "ml",
  "msk",
  "tsk",
  "krm",
]);
export type Unit = z.infer<typeof UnitSchema>;
export const units = UnitSchema.options;

const unitAliases = new Map<string, Unit>(units.map((unit) => [en.units[unit], unit]));

const parseUnit = (value: string): Unit | undefined => {
  const result = UnitSchema.safeParse(value);
  return result.success ? result.data : unitAliases.get(value);
};

export const formatAmount = (amount: string | undefined, language: Language) =>
  language === "sv" ? amount?.replace(".", ",") : amount;

export const formatQuantity = (
  amount: string | undefined,
  unitLabel: string | undefined,
  language: Language,
) => [formatAmount(amount, language), unitLabel].filter(Boolean).join(" ");

// Decimal strings preserve the entered precision, including values beyond Number's range.
export const maxAmountLength = 100;
export const AmountSchema = z
  .string()
  .trim()
  .transform((value) => value.replace(",", "."))
  .pipe(
    z
      .string()
      .max(maxAmountLength)
      .regex(/^[0-9]+(?:\.[0-9]+)?$/),
  )
  .refine((value) => /[1-9]/.test(value), "Amount must be greater than zero");

type ParsedItem = { text: string; amount?: string; unit?: Unit };

export const parseItemInput = (input: string): ParsedItem => {
  const text = input.trim();
  const match = /^([0-9]+(?:[.,][0-9]+)?)(\S*)\s+(.+)$/s.exec(text);
  if (!match) return { text };
  const amount = AmountSchema.safeParse(match[1]);
  if (!amount.success) return { text };

  const rest = match[3]!.trim();
  if (match[2]) {
    const unit = parseUnit(match[2]);
    return unit ? { text: rest, amount: amount.data, unit } : { text };
  }
  const unitMatch = /^(\S+)\s+(.+)$/s.exec(rest);
  if (unitMatch) {
    const unit = parseUnit(unitMatch[1]!);
    if (unit) return { text: unitMatch[2]!.trim(), amount: amount.data, unit };
  }
  return { text: rest, amount: amount.data };
};
