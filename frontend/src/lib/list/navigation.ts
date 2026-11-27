import { get } from "svelte/store";

import { navigate, route } from "../../router";
import { queryClient } from "../query";
import { session } from "../user/session";
import { preferredList, summariesQuery } from "./queries";

export const navigateToDefaultList = async () => {
  const account = get(session)?.id;
  const pathname = route.pathname;
  if (!account) {
    return;
  }

  const lists = await queryClient.fetchQuery(summariesQuery(account)).catch(() => []);
  if (account !== get(session)?.id || pathname !== route.pathname) {
    return;
  }

  const defaultList = preferredList(lists);
  if (defaultList) {
    return navigate("/lists/:id", {
      params: { id: defaultList.id },
    });
  }
  return navigate("/lists", { replace: true });
};
