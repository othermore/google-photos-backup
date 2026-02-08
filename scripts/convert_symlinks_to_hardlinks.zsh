#!/bin/zsh
# scripts/convert_symlinks_to_hardlinks.zsh

# Converts all symlinks in the directory (recursive) to hardlinks
# WARNING: Only works if target is on the same filesystem.

if [[ -z "$1" ]]; then
  echo "Usage: $0 <directory>"
  echo "Example: $0 ./backup/output"
  exit 1
fi

TARGET_DIR="$1"

if [[ ! -d "$TARGET_DIR" ]]; then
  echo "Error: Directory '$TARGET_DIR' not found."
  exit 1
fi

echo "🔍 Scanning for symlinks in: $TARGET_DIR"
count=0

# Loop through all symlinks
find "$TARGET_DIR" -type l -print0 | while IFS= read -r -d '' link; do
  # Get absolute target path
  # Handling relative symlinks requires care. readlink -f resolves to absolute.
  target=$(readlink -f "$link")
  
  if [[ -f "$target" ]]; then
    # Create hardlink (force overwrite of symlink)
    # Check if they are on same filesystem first? ln will fail if not.
    if ln -f "$target" "$link" 2>/dev/null; then
      echo "✅ Converted: $link -> $target"
      ((count++))
    else
      echo "❌ Failed to convert (cross-device?): $link -> $target"
    fi
  else
    echo "⚠️  Skipping broken link or non-file target: $link"
  fi
done

echo "🎉 Finished. Converted $count symlinks to hardlinks."
