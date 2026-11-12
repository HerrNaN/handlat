import { describe, expect, it } from "vitest";

import { parseItemInput, units } from "./quantity";

describe("item quantities", () => {
  it.each([
    ["milk", { text: "milk" }],
    ["  1,5 kg potatoes  ", { text: "potatoes", amount: "1.5", unit: "kg" }],
    ["1.25 l milk", { text: "milk", amount: "1.25", unit: "l" }],
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

  it.each(units)("recognizes the exact unit %s", (unit) => {
    expect(parseItemInput(`2 ${unit} groceries`)).toEqual({ text: "groceries", amount: "2", unit });
  });
});
