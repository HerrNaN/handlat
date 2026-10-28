import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { get } from "svelte/store";
import { queryClient } from "../query";
import { clearSession, refreshSession, session } from "./session";

const fetch = vi.fn<typeof globalThis.fetch>();

beforeEach(() => {
  fetch.mockReset();
  vi.stubGlobal("fetch", fetch);
  clearSession();
  queryClient.clear();
});

afterEach(() => {
  clearSession();
  queryClient.clear();
  vi.unstubAllGlobals();
});

it("loads the account through the session query", async () => {
  const account = { id: "owner", name: "Owner" };
  fetch.mockResolvedValueOnce(new Response(JSON.stringify(account)));
  await refreshSession();
  expect(get(session)).toEqual(account);
  expect(queryClient.getQueryData(["session"])).toEqual(account);
});

it("does not restore a session cleared while its query was pending", async () => {
  let resolve!: (response: Response) => void;
  fetch.mockImplementationOnce(() => new Promise((done) => { resolve = done; }));
  const pending = refreshSession();
  clearSession();
  resolve(new Response(JSON.stringify({ id: "owner", name: "Owner" })));
  await pending;
  expect(get(session)).toBeNull();
  expect(queryClient.getQueryData(["session"])).toBeUndefined();
});

it("clears all queries and mutations when the session is cleared", () => {
  queryClient.setQueryData(["private-lists", "owner", "summaries"], [{ id: "list-1", name: "Private" }]);
  queryClient.setQueryData(["session"], { id: "owner", name: "Owner" });
  queryClient.getMutationCache().build(queryClient, { mutationKey: ["private-lists", "write"] });
  queryClient.getMutationCache().build(queryClient, { mutationKey: ["auth", "sign-in"] });

  clearSession();
  expect(get(session)).toBeNull();
  expect(queryClient.getQueryCache().getAll()).toEqual([]);
  expect(queryClient.getMutationCache().getAll()).toEqual([]);
});

it("clears private data when a session refresh is unauthorized", async () => {
  fetch.mockResolvedValueOnce(new Response(JSON.stringify({ id: "owner", name: "Owner" })));
  await refreshSession();
  queryClient.setQueryData(["private-lists", "owner", "summaries"], [{ id: "list-1", name: "Private" }]);

  fetch.mockResolvedValueOnce(new Response("Unauthorized", { status: 401 }));
  await refreshSession();
  expect(get(session)).toBeNull();
  expect(queryClient.getQueryCache().findAll({ queryKey: ["private-lists"] })).toEqual([]);
});
