import z from "zod";

const isoDatetimeToDate = z.codec(z.iso.datetime(), z.date(), {
  decode: (isoString: string) => new Date(isoString),
  encode: (date: Date) => date.toISOString(),
});

export const ItemSchema = z.object({
  id: z.string(),
  text: z.string(),
  checkedAt: isoDatetimeToDate.optional(),
  createdAt: isoDatetimeToDate,
  department: z.string().optional(),
});

export type Item = z.infer<typeof ItemSchema>;
