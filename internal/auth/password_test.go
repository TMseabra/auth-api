package auth

import "testing"

func TestHashAndVerify(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}

	ok, err := VerifyPassword("correct horse battery staple", hash)
	if err != nil || !ok {
		t.Fatalf("expected match, ok=%v err=%v", ok, err)
	}

	ok, err = VerifyPassword("wrong password", hash)
	if err != nil || ok {
		t.Fatalf("expected mismatch, ok=%v err=%v", ok, err)
	}
}

func TestHashIsSalted(t *testing.T) {
	a, _ := HashPassword("same password")
	b, _ := HashPassword("same password")
	if a == b {
		t.Fatal("two hashes of the same password must differ")
	}
}

func TestVerifyRejectsMalformedHash(t *testing.T) {
	for _, h := range []string{"", "plaintext", "$argon2i$v=19$m=1,t=1,p=1$YQ$YQ", "$argon2id$v=19$bad$YQ$YQ"} {
		if _, err := VerifyPassword("x", h); err == nil {
			t.Errorf("expected error for %q", h)
		}
	}
}

func TestHashHandlesUnicodePasswords(t *testing.T) {
	const pw = "pässwörd-日本語-🔐-long-enough"
	hash, err := HashPassword(pw)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := VerifyPassword(pw, hash); err != nil || !ok {
		t.Fatalf("unicode password did not verify: ok=%v err=%v", ok, err)
	}
	if ok, _ := VerifyPassword("passwörd-日本語-🔐-long-enough", hash); ok {
		t.Fatal("different password verified")
	}
}

func TestVerifyRejectsTruncatedHashParts(t *testing.T) {
	hash, _ := HashPassword("some password")
	for _, bad := range []string{
		hash[:len(hash)-5] + "!!!!!", // invalid base64 in the key
		"$argon2id$v=19$m=65536,t=3,p=2$!!!$YQ",
		"$argon2id$v=18$m=65536,t=3,p=2$YQ$YQ",
	} {
		if _, err := VerifyPassword("some password", bad); err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}
}
