#!/bin/zsh

# migrate_structure.zsh
# Migrates existing backups to the flattened structure:
# FROM: snapshot/ID/raw/Takeout/Google Photos/...
# TO:   snapshot/Google Photos/...

set -e

BACKUP_ROOT="${1:-./backup}"

if [[ ! -d "$BACKUP_ROOT" ]]; then
    echo "❌ Backup root not found: $BACKUP_ROOT"
    echo "Usage: ./migrate_structure.zsh <path_to_backup_root>"
    exit 1
fi

echo "📂 Scanning backups in: $BACKUP_ROOT"

# Loop through each snapshot (directories with timestamp pattern)
for snapshot in "$BACKUP_ROOT"/*; do
    if [[ -d "$snapshot" ]]; then
        dirname=$(basename "$snapshot")
        # Check if it looks like a timestamp (simple regex)
        if [[ "$dirname" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}-[0-9]{6}$ ]]; then
            echo "🔍 Processing snapshot: $dirname"
            
            # Destination root for Google Photos in this snapshot
            DEST_GP="$snapshot/Google Photos"
            
            # Find all Export IDs (UUID-like folders)
            # Assumption: Any folder that is NOT "Google Photos" and NOT "backup_log.jsonl"
            for export_dir in "$snapshot"/*; do
                if [[ -d "$export_dir" ]]; then
                    export_name=$(basename "$export_dir")
                    
                    # Skip if it is the target "Google Photos" already
                    if [[ "$export_name" == "Google Photos" ]]; then
                        continue
                    fi
                    
                    echo "   Processing export dir: $export_name"
                    
                    # Check for structure: export_dir/raw/Takeout/Google Photos
                    SRC_GP="$export_dir/raw/Takeout/Google Photos"
                    ALT_GP="$export_dir/raw/Google Photos"
                    
                    TARGET_SRC=""
                    if [[ -d "$SRC_GP" ]]; then
                        TARGET_SRC="$SRC_GP"
                    elif [[ -d "$ALT_GP" ]]; then
                        TARGET_SRC="$ALT_GP"
                    fi
                    
                    if [[ -n "$TARGET_SRC" ]]; then
                        echo "   Found Google Photos content in: $TARGET_SRC"
                        mkdir -p "$DEST_GP"
                        
                        # Move content using rsync (preserve hardlinks -H, archive -a)
                        # Remove source files after transfer
                        echo "   Moving content..."
                        rsync -aH --remove-source-files "$TARGET_SRC/" "$DEST_GP/"
                        
                        # Clean up structure
                        # Remove source dir (now empty hopefully)
                        rmdir "$TARGET_SRC" 2>/dev/null || true
                        # Go up taking out parents
                        rmdir "$(dirname "$TARGET_SRC")" 2>/dev/null || true # Takeout or raw
                        rmdir "$export_dir/raw" 2>/dev/null || true
                        
                        # Move metadata files
                        echo "   Moving metadata files..."
                        # processing_index.json, state.json
                        for json_file in "$export_dir"/*.json; do
                            if [[ -f "$json_file" ]]; then
                                fname=$(basename "$json_file")
                                # Rename: filename_EXPORTID.json to avoid collision
                                new_name="${fname%.*}_$export_name.json"
                                mv "$json_file" "$snapshot/$new_name"
                            fi
                        done
                        
                        # Remove export dir itself
                        rmdir "$export_dir" 2>/dev/null || true
                        echo "   ✅ Migrated $export_name"
                    else
                        echo "   ⚠️  Could not find 'Google Photos' inside $export_name. Skipping."
                    fi
                fi
            done
        fi
    fi
done

echo "🎉 Migration Complete."
