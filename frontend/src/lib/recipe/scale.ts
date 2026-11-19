export const parsePortions = (value: string) => {
  const portions = Number(value);
  return /^[0-9]+$/.test(value) && Number.isSafeInteger(portions) && portions > 0
    ? portions
    : undefined;
};

const gcd = (a: bigint, b: bigint): bigint => (b === 0n ? a : gcd(b, a % b));

export const scaleAmount = (amount: string, target: number, base: number) => {
  if (target === base) return { amount, approximate: false };

  const [whole, fractional = ""] = amount.split(".");
  const numerator = BigInt(whole + fractional) * BigInt(target);
  const denominator = 10n ** BigInt(fractional.length) * BigInt(base);

  let reducedDenominator = denominator / gcd(numerator, denominator);
  let twos = 0;
  let fives = 0;
  while (reducedDenominator % 2n === 0n) {
    reducedDenominator /= 2n;
    twos++;
  }
  while (reducedDenominator % 5n === 0n) {
    reducedDenominator /= 5n;
    fives++;
  }

  const approximate = reducedDenominator !== 1n;
  let precision = Math.max(twos, fives);
  if (approximate) {
    precision = 6;
    // Keep six digits starting at the first nonzero decimal for very small amounts.
    let magnitude = numerator * 10n;
    while (magnitude < denominator) {
      magnitude *= 10n;
      precision++;
    }
  }

  const scaled = numerator * 10n ** BigInt(precision);
  const rounded = (scaled + denominator / 2n) / denominator;
  const digits = rounded.toString().padStart(precision + 1, "0");
  if (precision === 0) return { amount: digits, approximate };

  const wholeDigits = digits.slice(0, -precision);
  const fractionalDigits = digits.slice(-precision).replace(/0+$/, "");
  return {
    amount: fractionalDigits ? `${wholeDigits}.${fractionalDigits}` : wholeDigits,
    approximate,
  };
};
