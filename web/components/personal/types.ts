export type PersonalVaultSummary = {
  vaultId: string;
  schemaVersion: number;
  encryptionVersion: string;
  kdf: {
    name: string;
    iterations: number;
    salt: string;
  };
  updatedAt: number;
};

export type PersonalVaultItem = {
  itemType: string;
  syncId: string;
  ciphertext: string;
  nonce: string;
  updatedAt: number;
  deletedAt?: number | null;
  schemaVersion: number;
};

export type PersonalHost = {
  id?: number;
  sync_id: string;
  label: string;
  group_name?: string;
  tags?: string[];
  hostname: string;
  username: string;
  port: number;
  key_data?: string;
  secret?: string;
  key_type: string;
  created_at: string;
  updated_at: string;
  last_connected?: string;
};

export type PersonalGroup = {
  sync_id: string;
  name: string;
  created_at: string;
  updated_at: string;
  deleted_at?: string;
};

export type PersonalTokenHost = {
  host_id: number;
  display_label: string;
};

export type PersonalTokenDef = {
  token_id: string;
  name: string;
  created_at: string;
  updated_at: string;
  revoked_at?: string | null;
  deleted_at?: string | null;
  expires_at?: string | null;
  max_uses?: number;
  sync_enabled?: boolean;
  hosts: PersonalTokenHost[];
};

export type PersonalActivityEvent = {
  source: string;
  action: string;
  itemType?: string | null;
  itemCount?: number | null;
  createdAt: number;
};
