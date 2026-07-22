CREATE TABLE auth.oidc_identities (
  issuer text COLLATE "C" NOT NULL,
  subject text COLLATE "C" NOT NULL,
  user_id text NOT NULL REFERENCES auth.users(id) ON DELETE RESTRICT,
  linked_at timestamptz NOT NULL,
  PRIMARY KEY (issuer, subject),
  CONSTRAINT oidc_identities_issuer_length
    CHECK (octet_length(issuer) BETWEEN 1 AND 2048),
  CONSTRAINT oidc_identities_subject_length
    CHECK (octet_length(subject) BETWEEN 1 AND 255),
  CONSTRAINT oidc_identities_linked_at_finite
    CHECK (linked_at > '-infinity'::timestamptz AND linked_at < 'infinity'::timestamptz)
);

CREATE INDEX oidc_identities_user_id_idx ON auth.oidc_identities (user_id);
