type ResourcePayload = {
  id: string;
  label: string;
  group?: string;
  tags?: string[];
  hostname?: string;
  username?: string;
  port?: number;
  shareMode: string;
  notes?: string[];
  createdAt?: number;
  updatedAt?: number;
};

export function filterResourceForRole(
  role: string,
  resource: ResourcePayload,
  options?: { viewerCanSeeSensitiveMetadata?: boolean },
) {
  const normalized = role.toLowerCase();
  if (
    normalized === "owner" ||
    normalized === "admin" ||
    normalized === "vault_admin" ||
    normalized === "operator"
  ) {
    return {
      ...resource,
      visibilityMode: "full",
    };
  }

  if (normalized === "viewer" && options?.viewerCanSeeSensitiveMetadata) {
    return {
      ...resource,
      visibilityMode: "full",
    };
  }

  if (normalized === "restricted_operator") {
    return {
      id: resource.id,
      label: resource.label,
      group: resource.group ?? "",
      tags: resource.tags ?? [],
      port: resource.port ?? 22,
      shareMode: resource.shareMode,
      notes: (resource.notes ?? []).slice(0, 1),
      visibilityMode: "masked",
    };
  }

  return {
    id: resource.id,
    label: resource.label,
    group: resource.group ?? "",
    tags: [],
    port: resource.port ?? 22,
    shareMode: resource.shareMode,
    notes: [],
    visibilityMode: "masked",
  };
}
