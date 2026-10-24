import { createItemsStore } from "./store";
import { session, clearSession } from "../../user/session";

export const listStore = createItemsStore({
  fetch: (...args) => fetch(...args),
  onUnauthorized: clearSession,
});

session.subscribe((account) => listStore.setAccount(account?.id));
