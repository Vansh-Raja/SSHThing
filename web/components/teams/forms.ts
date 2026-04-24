import type { HostFormState, PersonalCredentialFormState } from "./types";

export const blankHostForm: HostFormState = {
  label: "",
  hostname: "",
  username: "",
  port: "22",
  group: "",
  tags: "",
  notes: "",
  credentialMode: "shared",
  credentialType: "none",
  sharedCredential: "",
};

export const blankPersonalCredentialForm: PersonalCredentialFormState = {
  username: "",
  credentialType: "password",
  secret: "",
};
