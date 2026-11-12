import z from "zod";

export const UnitSchema = z.enum(["st", "förp", "kg", "g", "l", "dl", "cl", "ml"]);
export type Unit = z.infer<typeof UnitSchema>;
export const units = UnitSchema.options;

// Decimal strings preserve the entered precision, including values beyond Number's range.
export const AmountSchema = z
  .string()
  .trim()
  .transform((value) => value.replace(",", "."))
  .pipe(
    z
      .string()
      .max(100)
      .regex(/^[0-9]+(?:\.[0-9]+)?$/),
  )
  .refine((value) => /[1-9]/.test(value), "Amount must be greater than zero");

type ParsedItem = { text: string; amount?: string; unit?: Unit };

export const parseItemInput = (input: string): ParsedItem => {
  const text = input.trim();
  const match = /^(\S+)\s+(.+)$/s.exec(text);
  if (!match) return { text };
  const amount = AmountSchema.safeParse(match[1]);
  if (!amount.success) return { text };

  const rest = match[2]!.trim();
  const unitMatch = /^(\S+)\s+(.+)$/s.exec(rest);
  const unit = UnitSchema.safeParse(unitMatch?.[1]);
  if (unit.success && unitMatch) {
    return { text: unitMatch[2]!.trim(), amount: amount.data, unit: unit.data };
  }
  return { text: rest, amount: amount.data };
};
