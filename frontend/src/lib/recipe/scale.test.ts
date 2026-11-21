import { describe, expect, it } from "vitest";

import { parsePortions, scaleAmount } from "./scale";

describe("recipe portions", () => {
  it.each(["", "0", "1.5", "+1", "2e3", "9007199254740992"])("rejects %s", (value) => {
    expect(parsePortions(value)).toBeUndefined();
  });

  it("accepts positive safe integers", () => {
    expect(parsePortions("9007199254740991")).toBe(Number.MAX_SAFE_INTEGER);
  });
});

describe("recipe quantities", () => {
  it.each([
    ["400", 2, 4, "200", false],
    ["1", 1, 2, "0.5", false],
    ["1.200", 4, 4, "1.200", false],
    ["9007199254740993.0000002", 1, 2, "4503599627370496.5000001", false],
    ["2", 1, 3, "0.666667", true],
    ["0.000000000001", 1, 3, "0.000000000000333333", true],
  ])("scales %s by %d/%d", (amount, target, base, expected, approximate) => {
    expect(scaleAmount(amount, target, base)).toEqual({ amount: expected, approximate });
  });
});
