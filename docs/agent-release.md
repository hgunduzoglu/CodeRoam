# Verify and install a CodeRoam agent release

CodeRoam publishes agent binaries only from an intentional `agent-vX.Y.Z` release request whose
source commit is already on `main`. A `repository_dispatch` event forces GitHub to source the
workflow from the default branch; the trusted workflow validates the unused tag name and exact
source commit before checking out its code. A read-only job builds `linux/amd64` and `linux/arm64`
with `CGO_ENABLED=0`; a separate publishing job creates the protected tag, receives the build
output, records its exact source and workflow commits, and creates GitHub/Sigstore provenance for
every released file. Source code never runs with release or OIDC permissions. The workflow never
publishes from a pull request and never overwrites a release.

The M3 branch is still a draft. Until its bootstrap and pairing lifecycle is complete, these steps
define and test the distribution contract; they do not make the starter `run` command a production
service.

## Download without executing

Install a current GitHub CLI, authenticate it, choose the released version, and download all four
assets into a new empty directory:

```bash
VERSION=0.1.0
REPOSITORY=hgunduzoglu/CodeRoam
DOWNLOAD_DIR="$(mktemp -d "coderoam-agent-${VERSION}.XXXXXX")"
cd "$DOWNLOAD_DIR"
gh release download "agent-v${VERSION}" \
  --repo "$REPOSITORY" \
  --pattern "coderoam-agent_${VERSION}_linux_*" \
  --pattern SHA256SUMS \
  --pattern RELEASE-METADATA
```

Do not run either binary yet. Select the local architecture explicitly:

```bash
case "$(uname -m)" in
  x86_64) ARCH=amd64 ;;
  aarch64 | arm64) ARCH=arm64 ;;
  *) echo "unsupported architecture" >&2; exit 1 ;;
esac
ARTIFACT="coderoam-agent_${VERSION}_linux_${ARCH}"
```

Require the exact four regular, non-symlink assets before trusting any of their contents:

```bash
test "$(find . -mindepth 1 -maxdepth 1 | wc -l | tr -d ' ')" = 4
for file in \
  "coderoam-agent_${VERSION}_linux_amd64" \
  "coderoam-agent_${VERSION}_linux_arm64" \
  SHA256SUMS RELEASE-METADATA; do
  test -f "$file"
  test ! -L "$file"
done
```

## Verify provenance and checksums

First verify the signed release metadata against CodeRoam's `main`-branch workflow, then read its
bounded values without executing or sourcing the file:

```bash
gh attestation verify RELEASE-METADATA \
  --repo "$REPOSITORY" \
  --signer-workflow "$REPOSITORY/.github/workflows/release-agent.yml" \
  --source-ref refs/heads/main

SOURCE_TAG="$(sed -n 's/^SOURCE_TAG=//p' RELEASE-METADATA)"
SOURCE_COMMIT="$(sed -n 's/^SOURCE_COMMIT=//p' RELEASE-METADATA)"
WORKFLOW_COMMIT="$(sed -n 's/^WORKFLOW_COMMIT=//p' RELEASE-METADATA)"
ARTIFACT_DIGEST="$(sed -n 's/^ARTIFACT_DIGEST=//p' RELEASE-METADATA)"
test "$(wc -l <RELEASE-METADATA | tr -d ' ')" = 4
test "$SOURCE_TAG" = "agent-v${VERSION}"
[[ "$SOURCE_COMMIT" =~ ^[0-9a-f]{40}$ ]]
[[ "$WORKFLOW_COMMIT" =~ ^[0-9a-f]{40}$ ]]
[[ "$ARTIFACT_DIGEST" =~ ^[0-9a-f]{64}$ ]]
test "$(gh api "repos/${REPOSITORY}/commits/${SOURCE_TAG}" --jq .sha)" = "$SOURCE_COMMIT"
```

Now bind every asset to both the exact source and signer-workflow commit recorded in that attested
metadata. The attestation source is `main` because the privileged workflow is deliberately sourced
from the trusted default branch, while `SOURCE_COMMIT` identifies the separately validated agent
code:

```bash
for file in coderoam-agent_${VERSION}_linux_* SHA256SUMS RELEASE-METADATA; do
  gh attestation verify "$file" \
    --repo "$REPOSITORY" \
    --signer-workflow "$REPOSITORY/.github/workflows/release-agent.yml" \
    --source-ref refs/heads/main \
    --source-digest "$WORKFLOW_COMMIT" \
    --signer-digest "$WORKFLOW_COMMIT"
done
```

Then verify both binary digests. On Linux:

```bash
sha256sum --check SHA256SUMS
```

On macOS, for verification before copying the chosen Linux artifact to its host:

```bash
shasum -a 256 --check SHA256SUMS
```

Any missing or duplicate metadata value, tag/commit mismatch, missing attestation, signer/ref/digest
mismatch, checksum failure, unexpected filename, or unsupported architecture is terminal. Do not
install the artifact and do not fall back to an unverified binary.

## Install for a non-root agent user

On a Linux host, create the dedicated account once using the host's normal administration process.
For distributions with `useradd`, the following creates a non-login service account and an
owner-only state directory while keeping the executable root-owned:

```bash
sudo useradd --system --create-home --home-dir /var/lib/coderoam \
  --shell /usr/sbin/nologin coderoam
sudo install -o root -g root -m 0755 "$ARTIFACT" /usr/local/bin/coderoam-agent
sudo install -d -o coderoam -g coderoam -m 0700 /var/lib/coderoam
sudo -u coderoam env HOME=/var/lib/coderoam /usr/local/bin/coderoam-agent version
```

The final command must print the selected semantic version and the tag commit SHA. Do not run the
agent as root, give the agent user sudo access, make its identity directory group/world-readable,
or enable autonomous self-update. Service supervision and the bounded outbound bootstrap command
are added in their own reviewed M3 slices.

## Maintainer release rule

Releases remain disabled until the repository has two active tag rulesets whose only include is
`refs/tags/agent-v*` and whose exclusions are empty:

1. A creation-only ruleset with exactly the GitHub Actions App as its sole `always` bypass actor.
2. An update-and-deletion ruleset with an empty bypass list, including for administrators.

Record their numeric IDs in `AGENT_RELEASE_CREATION_RULESET_ID` and
`AGENT_RELEASE_IMMUTABLE_RULESET_ID`. Record the GitHub Actions App's numeric actor ID in
`AGENT_RELEASE_CREATOR_ACTOR_ID`, and the only login allowed to request a release in
`AGENT_RELEASE_DISPATCH_ACTOR`. The workflow fetches and validates both live rules and the caller
before any requested source code is built. A creation bypass in the first rule therefore cannot
bypass immutability in the second rule.

Choose an unused stable tag and an exact commit already on `main`, then send the bounded dispatch
event. Do not create or push the tag yourself; the trusted publish job creates it. A
`repository_dispatch` uses the workflow definition and commit from the default branch, so the
caller cannot select another workflow revision:

```bash
RELEASE_TAG=agent-v0.1.0
REPOSITORY=hgunduzoglu/CodeRoam
SOURCE_COMMIT="$(git rev-parse main)"
gh api --method POST "repos/${REPOSITORY}/dispatches" \
  -f event_type=agent-release \
  -f "client_payload[release_tag]=$RELEASE_TAG" \
  -f "client_payload[source_commit]=$SOURCE_COMMIT"
```

The workflow rejects an unauthorized dispatcher, unstable or existing tag names, branch-only
commits, a non-`main` invocation, missing or incomplete protection rulesets, dirty or untracked
build inputs, and existing releases. It creates a lightweight tag at the verified source commit and
reads that remote tag immediately before and after publication. Only its final job receives release
and OIDC permissions; external actions are pinned to immutable commit SHAs. If publication fails
after immutable tag creation, reconcile the workflow and release state but never move or delete the
tag; fix the cause and publish a new patch version if no release exists.
