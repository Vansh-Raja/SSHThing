package teamsattach

import "testing"

func TestStoreUpsertReplaceAndDelete(t *testing.T) {
	s := Store{}
	s.Upsert(Attachment{TeamHostID: "h1", AttachmentType: AttachmentTypeExistingPersonalHost, PersonalHostID: 7})
	s.Upsert(Attachment{TeamHostID: "h1", AttachmentType: AttachmentTypeExistingPersonalHost, PersonalHostID: 9})

	a, ok := s.Find("h1")
	if !ok || a.PersonalHostID != 9 {
		t.Fatalf("expected updated attachment, got %+v", a)
	}

	s.Delete("h1")
	if _, ok := s.Find("h1"); ok {
		t.Fatalf("expected attachment to be deleted")
	}
}
