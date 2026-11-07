import type { Department } from "./departments";

export const tokenize = (raw: string): string[] =>
  normalize(raw)
    .split(/\s+/)
    .filter((t) => t.length > 1);

export interface ScoredDept {
  id: string;
  name: string;
  score: number;
}

interface DeptIndex {
  terms: Set<string>;
  multiWordKw: string[];
  dl: number;
}

const normalize = (text: string): string => text.toLowerCase().trim();

const buildIndexes = (
  departments: Department[],
): {
  indexes: DeptIndex[];
  avgdl: number;
  n: number;
} => {
  const indexes: DeptIndex[] = [];
  let totalLen = 0;

  for (const dept of departments) {
    const single: string[] = [];
    const multiWordKw: string[] = [];

    for (const kw of dept.keywords) {
      const trimmed = normalize(kw);
      if (trimmed.includes(" ")) {
        multiWordKw.push(trimmed);
      } else {
        single.push(trimmed);
      }
    }

    const terms = new Set(single);
    const dl = terms.size + multiWordKw.length * 0.5;
    totalLen += dl;
    indexes.push({ terms, multiWordKw, dl });
  }

  return {
    indexes,
    avgdl: totalLen / departments.length,
    n: departments.length,
  };
};

const deptIndexCache = new WeakMap<Department[], ReturnType<typeof buildIndexes>>();

const getIndex = (departments: Department[]) => {
  if (!deptIndexCache.has(departments)) {
    deptIndexCache.set(departments, buildIndexes(departments));
  }
  return deptIndexCache.get(departments)!;
};

const computeIdf = (matchTotalDf: number, n: number): number =>
  Math.log(1 + (n - matchTotalDf + 0.5) / (matchTotalDf + 0.5));

interface Bm25Params {
  k1: number;
  b: number;
}

const DEFAULT_PARAMS: Bm25Params = { k1: 1.2, b: 0.75 };

export const score = (
  rawInput: string,
  departments: Department[],
  params: Bm25Params = DEFAULT_PARAMS,
): ScoredDept[] => {
  const queryTokens = tokenize(rawInput);
  if (queryTokens.length === 0) return [];

  const rawNorm = normalize(rawInput);
  const { indexes, avgdl, n } = getIndex(departments);

  const globalDf = new Map<string, number>();
  for (const idx of indexes) {
    for (const t of queryTokens) {
      if (idx.terms.has(t)) {
        globalDf.set(t, (globalDf.get(t) || 0) + 1);
      }
    }
    for (const mw of idx.multiWordKw) {
      if (rawNorm.includes(mw)) {
        globalDf.set(mw, (globalDf.get(mw) || 0) + 1);
      }
    }
  }

  const results: ScoredDept[] = [];

  for (let i = 0; i < departments.length; i++) {
    const dept = departments[i];
    const idx = indexes[i];
    let internalScore = 0;

    for (const token of queryTokens) {
      if (!idx.terms.has(token)) continue;
      const df = globalDf.get(token) ?? 1;
      const idf = computeIdf(df, n);
      const tf = 1;
      internalScore +=
        idf *
        ((tf * (params.k1 + 1)) / (tf + params.k1 * (1 - params.b + params.b * (idx.dl / avgdl))));
    }

    for (const mw of idx.multiWordKw) {
      if (rawNorm.includes(mw)) {
        const df = globalDf.get(mw) ?? 1;
        const idf = computeIdf(df, n);
        const tf = 1;
        internalScore +=
          idf *
          ((tf * (params.k1 + 1)) /
            (tf + params.k1 * (1 - params.b + params.b * (idx.dl / avgdl))));
      }
    }

    results.push({
      id: dept.id,
      name: dept.name,
      score: internalScore,
    });
  }

  results.sort((a, b) => b.score - a.score);
  return results;
};

type ClassificationResult = {
  id: string;
  name: string;
  score: number;
};

const THRESHOLD = 0.3;

export const classify = (rawInput: string, departments: Department[]): ClassificationResult => {
  const results = score(rawInput, departments);
  if (results.length === 0 || results[0].score < THRESHOLD) {
    const other = departments.find((d) => d.id === "other") ?? departments[departments.length - 1];
    return { id: other.id, name: other.name, score: 0 };
  }
  return results[0];
};
