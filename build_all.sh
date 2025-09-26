#!/bin/bash

# ========================================
# Ultimate Go Multi-command Cross-platform Build Script
# Features:
#   - Support building specific commands: ./build_all.sh cmd1 cmd2
#   - Auto-scan cmd/* (when no args)
#   - UPX compression (optional)
#   - Build for all platforms
#   - Generate unified release.zip + version notes
# Environment: Windows Git Bash / Linux / macOS
# ========================================

# ---------- Config ----------
PROJECT_NAME="${PROJECT_NAME:-apusic}"
CMD_DIR="cmd"
BUILD_DIR="pkg"
RELEASE_DIR="release"
BUILD_TIME=$(date -u "+%Y-%m-%d %H:%M:%S UTC")
GIT_COMMIT=$(git rev-parse --short=8 HEAD 2>/dev/null || echo "unknown")
GIT_TAG=$(git describe --tags --exact-match 2>/dev/null || echo "dev")
VERSION="${GIT_TAG}-${GIT_COMMIT}"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log()     { echo -e "${BLUE}📦 $*${NC}"; }
success() { echo -e "${GREEN}✅ $*${NC}"; }
warn()    { echo -e "${YELLOW}⚠️  $*${NC}"; }
error()   { echo -e "${RED}❌ $*${NC}"; }

has_upx() { command -v upx >/dev/null 2>&1 && upx --version >/dev/null 2>&1; }

# ---------- Main Function ----------
main() {
    log "Starting build system"
    log "Project: $PROJECT_NAME | Version: $VERSION"

    # ========== Parse CLI Args ==========
    if [ $# -eq 0 ]; then
        # Auto-scan cmd/*
        if [ ! -d "$CMD_DIR" ]; then
            error "Directory '$CMD_DIR' does not exist, and no commands specified"
            exit 1
        fi

        # Compatible with old macOS bash (no mapfile)
        if command -v mapfile >/dev/null 2>&1; then
            mapfile -t CMD_LIST < <(find "$CMD_DIR" -mindepth 1 -maxdepth 1 -type d -exec basename {} \; | sort)
        else
            CMD_LIST=()
            while IFS= read -r dir; do
                CMD_LIST+=("$(basename "$dir")")
            done < <(find "$CMD_DIR" -mindepth 1 -maxdepth 1 -type d | sort)
        fi

        if [ ${#CMD_LIST[@]} -eq 0 ]; then
            error "No command directories found under '$CMD_DIR'"
            exit 1
        fi
        log "Auto-discovered commands: ${CMD_LIST[*]}"
    else
        # Use specified args
        CMD_LIST=("$@")
        log "Building specified commands: ${CMD_LIST[*]}"
    fi

    # Validate each command directory
    for CMD in "${CMD_LIST[@]}"; do
        if [ ! -d "$CMD_DIR/$CMD" ]; then
            error "Command directory '$CMD_DIR/$CMD' does not exist!"
            exit 1
        fi
        if ! find "$CMD_DIR/$CMD" -name "*.go" | grep -q "."; then
            error "No Go source files found in '$CMD'"
            exit 1
        fi
    done

    # ========== Clean & Prepare ==========
    rm -rf "$BUILD_DIR" "$RELEASE_DIR"
    mkdir -p "$BUILD_DIR" "$RELEASE_DIR"

    # ========== UPX Detection ==========
    if has_upx; then
        UPX_AVAILABLE=1
        success "UPX enabled"
    else
        UPX_AVAILABLE=0
        warn "UPX not installed, skipping compression"
    fi

    # ========== Check zip availability ==========
    if ! command -v zip >/dev/null 2>&1; then
        error "zip command not found. Please install zip first."
        exit 1
    fi

    # ========== Target Platforms ==========
    TARGETS=(
        "windows amd64 1"
        "windows arm64 1"
        "linux   amd64 0"
        "linux   arm64 0"
        "darwin  amd64 0"
        "darwin  arm64 0"
    )

    # ========== Build Each Command ==========
    for CMD in "${CMD_LIST[@]}"; do
        echo
        CMD_PATH="$CMD_DIR/$CMD"
        log "========== Building command: $CMD | Path: $CMD_PATH =========="

        for TARGET in "${TARGETS[@]}"; do
            set -- $TARGET
            GOOS=$1
            GOARCH=$2
            IS_WINDOWS=$3

            OUTPUT_NAME="$CMD-$VERSION-$GOOS-$GOARCH"
            BINARY_NAME="$OUTPUT_NAME"
            [ $IS_WINDOWS -eq 1 ] && BINARY_NAME="$OUTPUT_NAME.exe"

            OUTPUT_PATH="$BUILD_DIR/$BINARY_NAME"

            log "→ $GOOS/$GOARCH"

            export GOOS GOARCH CGO_ENABLED=0

            go build \
                -ldflags "-s -w \
                -X 'main.version=$VERSION' \
                -X 'main.buildTime=$BUILD_TIME' \
                -X 'main.gitCommit=$GIT_COMMIT' \
                -X 'main.buildBy=$(uname -s)' \
                -X 'main.command=$CMD'" \
                -o "$OUTPUT_PATH" \
                "./$CMD_PATH"

            if [ $? -ne 0 ]; then
                error "Build failed: $CMD ($GOOS/$GOARCH)"
                exit 1
            fi
            success "✔ $BINARY_NAME"

            # UPX Compression (skip for darwin)
            if [ $UPX_AVAILABLE -eq 1 ] && [ "$GOOS" != "darwin" ]; then
                upx --best --quiet "$OUTPUT_PATH"
                COMPRESSED=$(upx -q -l "$OUTPUT_PATH" | tail -1 | awk '{print $6}')
                success "⚡ Compressed: $COMPRESSED"
            fi
        done
    done

    # ========== Create Unified Release ==========
    log "📦 Creating unified release package..."

    # Create per-platform directories
    for TARGET in "${TARGETS[@]}"; do
        set -- $TARGET
        GOOS=$1
        GOARCH=$2
        IS_WINDOWS=$3

        PLATFORM_DIR="$RELEASE_DIR/$GOOS-$GOARCH"
        mkdir -p "$PLATFORM_DIR"

        for CMD in "${CMD_LIST[@]}"; do
            OUTPUT_NAME="$CMD-$VERSION-$GOOS-$GOARCH"
            BINARY_NAME="$OUTPUT_NAME"
            [ $IS_WINDOWS -eq 1 ] && BINARY_NAME="$OUTPUT_NAME.exe"

            cp "$BUILD_DIR/$BINARY_NAME" "$PLATFORM_DIR/" || {
                error "Copy failed: $BINARY_NAME"
                exit 1
            }
        done
    done

    # Add version file
    cat > "$RELEASE_DIR/VERSION.txt" << EOF
Project: $PROJECT_NAME
Version: $VERSION
Git Commit: $GIT_COMMIT
Build Time: $BUILD_TIME
Commands: ${CMD_LIST[*]}
Built on: $(uname -s)
EOF

    # Add release notes
    cat > "$RELEASE_DIR/RELEASE_NOTES.md" << EOF
# Release: $VERSION

- **Project**: $PROJECT_NAME
- **Build Time**: $BUILD_TIME
- **Git Commit**: \`$GIT_COMMIT\`
- **Commands**: ${CMD_LIST[*]}

## File List

EOF

    find "$RELEASE_DIR" -type f -not -name "RELEASE_NOTES.md" | sort | while read file; do
        echo "- \$(basename "$file")" >> "$RELEASE_DIR/RELEASE_NOTES.md"
    done

    # Package into release.zip
    RELEASE_ZIP="$PROJECT_NAME-release-$VERSION.zip"
    (cd "$RELEASE_DIR" && zip -rq "../$BUILD_DIR/$RELEASE_ZIP" .)

    success "🎉 Unified release package created: ./$BUILD_DIR/$RELEASE_ZIP"
    success "Release content located in: ./$RELEASE_DIR/"

    # List all build artifacts
    echo
    success "✅ All builds completed! Output files:"
    ls -lh "$BUILD_DIR/" | grep -E "\.(exe|zip|gz)$" | awk '{print "  " $9}'

    echo
    success "💡 Usage examples:"
    success "  ./build_all.sh                    # Build all commands"
    success "  ./build_all.sh server agent       # Build only server and agent"
}

# ---------- Entry ----------
main "$@"
