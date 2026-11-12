import z from "zod";

import { AmountSchema, UnitSchema } from "./quantity";

const isoDatetimeToDate = z.codec(z.iso.datetime(), z.date(), {
  decode: (isoString: string) => new Date(isoString),
  encode: (date: Date) => date.toISOString(),
});

export const ItemSchema = z.object({
  id: z.string(),
  text: z.string(),
  amount: AmountSchema.optional(),
  unit: UnitSchema.optional(),
  checkedAt: isoDatetimeToDate.optional(),
  createdAt: isoDatetimeToDate,
  department: z.string().optional(),
});

export type Item = z.infer<typeof ItemSchema>;
