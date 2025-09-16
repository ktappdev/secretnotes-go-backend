#!/usr/bin/env bash
set -euo pipefail

# SecretNotes CLI release helper
# Usage:
#   scripts/release.sh v0.1.1
# Or run without arg and it will prompt for a version.
#
# What it does:
# - Verifies a clean (or confirmed) working tree
# - Creates an annotated tag (e.g., v0.1.1)
# - Pushes the tag to origin to trigger the GitHub Release workflow
#
# Notes:
# - You can push commits before or after running this script.
# - If your working tree has changes, you'll be prompted to continue or abort.
# - Requires: git

RED="\033[31m"; GREEN="\033[32m"; YELLOW="\033[33m"; BLUE="\033[34m"; RESET="\033[0m"

# Ensure we're in the repo root
if ! git rev-parse --show-toplevel >/dev/null 2>&1; then
  echo -e "${RED}Error:${RESET} Not inside a git repository" >&2
  exit 1
fi
REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

# Parse args
VERSION="${1:-}"
NONINTERACTIVE="false"
ALLOW_DIRTY="false"

for arg in "$@"; do
  case "$arg" in
    -y|--yes) NONINTERACTIVE="true" ;;
    --allow-dirty) ALLOW_DIRTY="true" ;;
  esac
done

if [[ -z "$VERSION" || "$VERSION" =~ ^- ]]; then
  read -rp "Enter version tag (e.g., v0.1.1): " VERSION
fi

# Basic semver-ish validation: must start with 'v'
if [[ ! "$VERSION" =~ ^v[0-9]+(\.[0-9]+){1,2}([A-Za-z0-9.-]*)?$ ]]; then
  echo -e "${RED}Error:${RESET} Version must look like v0.1.1" >&2
  exit 1
fi

# Check if tag already exists
if git rev-parse -q --verify "refs/tags/$VERSION" >/dev/null; then
  echo -e "${RED}Error:${RESET} Tag $VERSION already exists" >&2
  exit 1
fi

# Check working tree cleanliness
if [[ "$ALLOW_DIRTY" != "true" ]]; then
  if [[ -n $(git status --porcelain) ]]; then
    if [[ "$NONINTERACTIVE" == "true" ]]; then
      echo -e "${RED}Error:${RESET} Working tree has uncommitted changes. Commit or pass --allow-dirty." >&2
      exit 1
    fi
    echo -e "${YELLOW}Warning:${RESET} You have uncommitted changes." \
         "Tagging will proceed with current workspace state in HEAD." \
         "(they won't be included unless committed)."
    read -rp "Continue anyway? [y/N]: " ans
    ans=${ans:-N}
    if [[ ! "$ans" =~ ^[yY]$ ]]; then
      echo "Aborted."; exit 1
    fi
  fi
fi

# Confirm remote exists
if ! git remote | grep -q '^origin$'; then
  echo -e "${RED}Error:${RESET} No 'origin' remote configured" >&2
  exit 1
fi

CURRENT_BRANCH="$(git rev-parse --abbrev-ref HEAD)"

# Optionally push current branch first if not up to date
LOCAL_AHEAD=false
if ! git remote show origin | grep -q "local out of date"; then
  # Determine if local commits not on origin
  if [[ -n $(git rev-list --left-only --count "$CURRENT_BRANCH"..."origin/$CURRENT_BRANCH" 2>/dev/null || echo) ]]; then
    LOCAL_AHEAD=true
  fi
fi

if [[ "$LOCAL_AHEAD" == "true" && "$NONINTERACTIVE" != "true" ]]; then
  echo -e "${YELLOW}Notice:${RESET} Your branch '$CURRENT_BRANCH' has local commits not pushed to origin."
  read -rp "Push '$CURRENT_BRANCH' before tagging? [Y/n]: " push_ans
  push_ans=${push_ans:-Y}
  if [[ "$push_ans" =~ ^[yY]$ ]]; then
    echo -e "${BLUE}Pushing branch...${RESET}"
    git push origin "$CURRENT_BRANCH"
  fi
fi

# Create annotated tag
echo -e "${BLUE}Creating tag ${VERSION}...${RESET}"
git tag -a "$VERSION" -m "Release $VERSION"

echo -e "${BLUE}Pushing tag ${VERSION} to origin...${RESET}"
git push origin "$VERSION"

echo -e "${GREEN}Done!${RESET} GitHub Actions will now build and publish release assets."
echo -e "View releases: ${BLUE}https://github.com/$(git config --get remote.origin.url | sed -E 's#.*github.com[:/](.*)\.git#\1#')/releases${RESET}"

# Optional: user's convenience command to stage, commit, and push
# Only run if available in PATH
if command -v gitcomm >/dev/null 2>&1; then
  echo -e "${BLUE}Running user command:${RESET} gitcomm -sa -ap"
  gitcomm -sa -ap || true
else
  echo -e "${YELLOW}Note:${RESET} 'gitcomm' not found; skipping convenience commit/push."
fi
