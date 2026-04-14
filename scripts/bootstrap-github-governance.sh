#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
LABELS_FILE="${ROOT_DIR}/.github/labels.json"

if ! command -v gh >/dev/null 2>&1; then
  echo "gh CLI is required"
  exit 1
fi

repo_full_name=${1:-$(gh repo view --json nameWithOwner -q .nameWithOwner)}
owner=${repo_full_name%/*}
repo=${repo_full_name#*/}
default_branch=$(gh api "repos/${owner}/${repo}" --jq .default_branch)

echo "Configuring GitHub governance for ${owner}/${repo} on branch ${default_branch}"

python3 - "$LABELS_FILE" "$owner" "$repo" <<'PY'
import json
import subprocess
import sys

labels_file, owner, repo = sys.argv[1:]
with open(labels_file, "r", encoding="utf-8") as handle:
    labels = json.load(handle)

for label in labels:
    name = label["name"]
    color = label["color"]
    description = label["description"]
    get = subprocess.run(
        ["gh", "api", f"repos/{owner}/{repo}/labels/{name}"],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    if get.returncode == 0:
        subprocess.run(
            [
                "gh", "api", "-X", "PATCH",
                f"repos/{owner}/{repo}/labels/{name}",
                "-f", f"new_name={name}",
                "-f", f"color={color}",
                "-f", f"description={description}",
            ],
            check=True,
            stdout=subprocess.DEVNULL,
        )
    else:
        subprocess.run(
            [
                "gh", "api", "-X", "POST",
                f"repos/{owner}/{repo}/labels",
                "-f", f"name={name}",
                "-f", f"color={color}",
                "-f", f"description={description}",
            ],
            check=True,
            stdout=subprocess.DEVNULL,
        )
PY

gh api -X PATCH "repos/${owner}/${repo}" \
  -F allow_squash_merge=true \
  -F allow_merge_commit=false \
  -F allow_rebase_merge=false \
  -F delete_branch_on_merge=true \
  -F allow_auto_merge=true \
  -F web_commit_signoff_required=true >/dev/null

cat > /tmp/axiom-branch-protection.json <<'EOF'
{
  "required_status_checks": {
    "strict": true,
    "contexts": [
      "Backend Tests",
      "Frontend Tests",
      "Build Container Image",
      "Docker Compose Smoke Test",
      "Container Vulnerability Scan",
      "Go Security Analysis",
      "Secret Detection",
      "Infrastructure as Code Security",
      "Dependency Vulnerability Check",
      "License Compliance Check"
    ]
  },
  "enforce_admins": true,
  "required_pull_request_reviews": {
    "dismiss_stale_reviews": true,
    "require_code_owner_reviews": true,
    "required_approving_review_count": 1,
    "require_last_push_approval": true
  },
  "restrictions": null,
  "required_linear_history": true,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "block_creations": false,
  "required_conversation_resolution": true,
  "lock_branch": false,
  "allow_fork_syncing": true
}
EOF

gh api -X PUT "repos/${owner}/${repo}/branches/${default_branch}/protection" --input /tmp/axiom-branch-protection.json >/dev/null

rm -f /tmp/axiom-branch-protection.json

echo "GitHub governance configured for ${owner}/${repo}"
