#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
TEST_PROJECTS_DIR="$ROOT_DIR/test_projects"

echo "Setting up E2E test projects in $TEST_PROJECTS_DIR..."

if [ ! -d "$TEST_PROJECTS_DIR" ]; then
  echo "Error: test_projects directory not found."
  exit 1
fi

for project in "$TEST_PROJECTS_DIR"/*; do
  if [ -d "$project" ]; then
    proj_name="$(basename "$project")"
    echo "=========================================="
    echo "Bootstrapping project: $proj_name"
    echo "=========================================="

    if [ "$proj_name" = "monorepo_pnpm" ]; then
      if command -v pnpm &> /dev/null; then
        echo "Running pnpm install in $proj_name..."
        (cd "$project" && pnpm install)
      else
        echo "pnpm not found, falling back to npm install in $proj_name subdirectories..."
        for pkg in "$project"/packages/*; do
          if [ -d "$pkg" ]; then
            echo "Running npm install in $(basename "$pkg")..."
            (cd "$pkg" && npm install)
          fi
        done
      fi
    else
      echo "Running npm install in $proj_name..."
      (cd "$project" && npm install)
    fi
  fi
done

echo "=========================================="
echo "All E2E test projects setup successfully!"
echo "=========================================="
