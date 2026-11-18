import { describe, expect, it } from "vitest";

import { formatAmount, parseItemInput, units } from "./quantity";

describe("item quantities", () => {
  it.each([
    ["milk", { text: "milk" }],
    ["  1,5 kg potatoes  ", { text: "potatoes", amount: "1.5", unit: "kg" }],
    ["1.25 l milk", { text: "milk", amount: "1.25", unit: "l" }],
    ["2 pcs apples", { text: "apples", amount: "2", unit: "st" }],
    ["2pk coffee", { text: "coffee", amount: "2", unit: "pkt" }],
    ["1,5 tsk salt", { text: "salt", amount: "1.5", unit: "tsk" }],
    ["2 pkt pasta", { text: "pasta", amount: "2", unit: "pkt" }],
    ["1kg potatis", { text: "potatis", amount: "1", unit: "kg" }],
    ["1,5kg potatis", { text: "potatis", amount: "1.5", unit: "kg" }],
    ["1.25l milk", { text: "milk", amount: "1.25", unit: "l" }],
    ["2bags potatoes", { text: "2bags potatoes" }],
    ["2KG potatoes", { text: "2KG potatoes" }],
    ["2kg", { text: "2kg" }],
    ["7up soda", { text: "7up soda" }],
    ["1,2,3kg potatoes", { text: "1,2,3kg potatoes" }],
    ["3 milk", { text: "milk", amount: "3" }],
    ["2 bags potatoes", { text: "bags potatoes", amount: "2" }],
    ["2 kilograms potatoes", { text: "kilograms potatoes", amount: "2" }],
    ["2 KG potatoes", { text: "KG potatoes", amount: "2" }],
    ["2 kg", { text: "kg", amount: "2" }],
    ["123", { text: "123" }],
    ["0 kg potatoes", { text: "0 kg potatoes" }],
    ["-1 kg potatoes", { text: "-1 kg potatoes" }],
    ["1,2,3 kg potatoes", { text: "1,2,3 kg potatoes" }],
    [
      "9007199254740993.123456789 g salt",
      { text: "salt", amount: "9007199254740993.123456789", unit: "g" },
    ],
  ])("parses %s without losing name text or decimal precision", (input, expected) => {
    expect(parseItemInput(input)).toEqual(expected);
  });

  it("localizes decimals without rounding or dropping trailing zeroes", () => {
    expect(formatAmount("9007199254740993.123450", "sv")).toBe("9007199254740993,123450");
    expect(formatAmount("9007199254740993.123450", "en")).toBe("9007199254740993.123450");
  });

  it.each(units)("recognizes the exact unit %s", (unit) => {
    for (const gap of ["", " "]) {
      expect(parseItemInput(`2${gap}${unit} groceries`)).toEqual({
        text: "groceries",
        amount: "2",
        unit,
      });
    }
  });
});
