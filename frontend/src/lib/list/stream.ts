import { get } from "svelte/store";

import { queryClient } from "../query";
import { session } from "../user/session";
import { ListSchema } from "./api";
import { listKey, summariesKey } from "./queries";

export const subscribeToList = (account: string, listID: string, onDeleted: () => void) => {
  const events = new EventSource(`/api/lists/${encodeURIComponent(listID)}/events`);
  let closed = false;
  const active = () => !closed && account === get(session)?.id;
  events.onmessage = (event) => {
    if (!active()) return;
    const result = ListSchema.safeParse(JSON.parse(event.data));
    if (!result.success || result.data.id !== listID) return;
    queryClient.setQueryData(listKey(account, listID), result.data);
    void queryClient.invalidateQueries({ queryKey: summariesKey(account) });
  };
  events.onerror = () => {
    if (!active() || events.readyState !== EventSource.CLOSED) return;
    queryClient
      .getQueryCache()
      .find({ queryKey: listKey(account, listID), exact: true })
      ?.setState({
        status: "error",
        error: new Error("Unable to open the list stream."),
      });
  };
  events.addEventListener("deleted", () => {
    events.close();
    if (!active()) return;
    queryClient.removeQueries({ queryKey: listKey(account, listID), exact: true });
    void queryClient.invalidateQueries({ queryKey: summariesKey(account) });
    onDeleted();
  });
  return () => {
    closed = true;
    events.close();
  };
};
