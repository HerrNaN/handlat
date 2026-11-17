import z from "zod";

const errorSchema = z.object({
  code: z.string(),
  params: z.record(z.string(), z.number().int()).optional(),
});

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code?: string,
    public readonly params?: Record<string, number>,
  ) {
    super(code ?? `HTTP ${status}`);
  }
}

export const readApiError = async (response: Response): Promise<ApiError> => {
  if (response.headers.get("content-type")?.includes("application/json")) {
    const body: unknown = await response.json().catch(() => undefined);
    const parsed = errorSchema.safeParse(body);
    if (parsed.success) {
      return new ApiError(response.status, parsed.data.code, parsed.data.params);
    }
  }
  return new ApiError(response.status);
};

export const post = async (url: string, body?: unknown) => {
  const response = await fetch(url, {
    method: "POST",
    ...(body === undefined
      ? {}
      : {
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        }),
  });
  if (!response.ok) throw await readApiError(response);
  return response;
};
