#!/bin/bash

set -e

# Usage function
usage() {
  echo "Usage: $0 [major|minor|patch]"
  exit 1
}

# Check for exactly one argument
if [[ $# -ne 1 ]]; then
  usage
fi

INCREMENT=$1

# Validate argument
if [[ "$INCREMENT" != "major" && "$INCREMENT" != "minor" && "$INCREMENT" != "patch" ]]; then
  usage
fi

# Fetch all tags from remote to ensure up-to-date
git fetch --tags

# Get the latest tag reachable from HEAD
CURRENT_TAG=$(git describe --abbrev=0 --tags 2>/dev/null || echo "v0.0.0")

# Validate tag format
if [[ ! $CURRENT_TAG =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
  echo "Current tag '$CURRENT_TAG' does not match semantic versioning vMAJOR.MINOR.PATCH"
  exit 1
fi

# Extract major, minor, and patch numbers
MAJOR=${BASH_REMATCH[1]}
MINOR=${BASH_REMATCH[2]}
PATCH=${BASH_REMATCH[3]}

# Increment version based on argument
case $INCREMENT in
  major)
    MAJOR=$((MAJOR + 1))
    MINOR=0
    PATCH=0
    ;;
  minor)
    MINOR=$((MINOR + 1))
    PATCH=0
    ;;
  patch)
    PATCH=$((PATCH + 1))
    ;;
esac

NEW_TAG="v${MAJOR}.${MINOR}.${PATCH}"
echo "New tag will be: $NEW_TAG"

# Create new tag and push it
git tag "$NEW_TAG"
git push origin "$NEW_TAG"
git push

echo "Tagged current commit with new version: $NEW_TAG"
