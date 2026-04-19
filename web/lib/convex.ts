import { ConvexHttpClient } from "convex/browser";
import { anyApi } from "convex/server";

import { getRequiredEnv } from "./env";

function client(): ConvexHttpClient {
  return new ConvexHttpClient(getRequiredEnv("NEXT_PUBLIC_CONVEX_URL"));
}

export async function convexQuery<T>(reference: unknown, args: Record<string, any> = {}): Promise<T> {
  return client().query(reference as never, args as never);
}

export async function convexMutation<T>(reference: unknown, args: Record<string, any> = {}): Promise<T> {
  return client().mutation(reference as never, args as never);
}

export const convexApi = anyApi;
