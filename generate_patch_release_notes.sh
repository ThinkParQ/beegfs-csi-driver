#!/bin/bash

# Copyright 2023 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


# Usage: generate_patch_release_notes.sh
#
# Generates and creates PRs for kubernetes-csi patch releases.
#
# Required environment variables
# CSI_RELEASE_TOKEN: Github token needed for generating release notes
# GITHUB_USER: Github username to create PRs with
#
# Instructions:
# 1. Login with "gh auth login"
# 2. Copy this script to the kubernetes-csi directory (one directory above the
# repos)
# 3. Update the repos and versions in the $releases array
# 4. Set environment variables
# 5. Run script from the kubernetes-csi directory
#
# Caveats:
# - This script doesn't handle regenerating and updating existing PRs yet.
#   It might work if you comment out the PR creation line

set -e
set -x

releases=(
#  "external-attacher 4.4.1"
#  "external-provisioner 3.6.1"
#  "external-snapshotter 6.2.3"
)

function gen_patch_relnotes() {
  rm out.md || true
  rm -rf /tmp/k8s-repo || true
  GITHUB_TOKEN="$CSI_RELEASE_TOKEN" \
  release-notes --discover=patch-to-latest --branch="$2" \
    --org=kubernetes-csi --repo="$1" \
    --required-author="" --markdown-links --output out.md
}

for rel in "${releases[@]}"; do
  read -r repo version <<< "$rel"

  # Parse minor version
  minorPattern="(^[[:digit:]]+\.[[:digit:]]+)\."
  [[ "$version" =~ $minorPattern ]]
  minor="${BASH_REMATCH[1]}"

  echo "$repo" "$version" "$minor"

  pushd "$repo/CHANGELOG"

  git fetch upstream

  # Create branch
  branch="changelog-release-$minor"
  git checkout master
  git branch -D "$branch" || true
  git checkout --track "upstream/release-$minor" -b "$branch"

  # Generate release notes
  gen_patch_relnotes "$repo" "release-$minor"
  cat > tmp.md <<EOF
# Release notes for v$version

[Documentation](https://kubernetes-csi.github.io)

EOF

  cat out.md >> tmp.md
  echo >> tmp.md

  file="CHANGELOG-$minor.md"
  cat "$file" >> tmp.md
  mv tmp.md "$file"

  git add -u
  git commit -m "Add changelog for $version"
  git push -f origin "$branch"

  # Create PR
prbody=$(cat <<EOF
\`\`\`release-note
NONE
\`\`\`
EOF
)
  gh pr create --title="Changelog for v$version" --body "$prbody"  --head "$GITHUB_USER:$branch" --base "release-$minor" --repo="kubernetes-csi/$repo"

  popd
done
