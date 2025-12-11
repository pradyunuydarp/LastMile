#!/bin/bash

# render_report.sh
# Renders PlantUML diagrams to PNG and compiles LaTeX report to PDF

set -e  # Exit on error

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPORT_DIR="$SCRIPT_DIR/docs/report"
DIAGRAMS_DIR="$REPORT_DIR/diagrams"
IMAGES_DIR="$REPORT_DIR/images"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}════════════════════════════════════════${NC}"
echo -e "${GREEN}  LastMile Report Renderer${NC}"
echo -e "${GREEN}════════════════════════════════════════${NC}"
echo ""

# Check dependencies
check_dependency() {
    if ! command -v "$1" &> /dev/null; then
        echo -e "${RED}Error: $1 is not installed${NC}"
        echo -e "${YELLOW}Install with: $2${NC}"
        exit 1
    fi
}

echo -e "${YELLOW}Checking dependencies...${NC}"
check_dependency "plantuml" "brew install plantuml"
check_dependency "tectonic" "brew install tectonic"

echo -e "${GREEN}✓ All dependencies found${NC}"
echo ""

# Create output directories
mkdir -p "$DIAGRAMS_DIR"
mkdir -p "$IMAGES_DIR"

# Render PlantUML diagrams
echo -e "${YELLOW}Rendering PlantUML diagrams...${NC}"

PUML_FILES=(
    "$DIAGRAMS_DIR/system_architecture.puml"
    "$DIAGRAMS_DIR/deployment_architecture.puml"
    "$DIAGRAMS_DIR/load_balancing.puml"
    "$DIAGRAMS_DIR/matching_flow.puml"
)

for puml_file in "${PUML_FILES[@]}"; do
    if [ -f "$puml_file" ]; then
        filename=$(basename "$puml_file" .puml)
        echo -e "  → Rendering ${filename}.png"
        
        # Change to diagrams directory and render there
        (cd "$DIAGRAMS_DIR" && plantuml -tpng "$(basename "$puml_file")")
        
        if [ $? -eq 0 ]; then
            echo -e "    ${GREEN}✓ Success${NC}"
        else
            echo -e "    ${RED}✗ Failed${NC}"
            exit 1
        fi
    else
        echo -e "  ${YELLOW}⚠ File not found: $puml_file${NC}"
    fi
done

echo ""
echo -e "${GREEN}✓ All diagrams rendered successfully${NC}"
echo ""

# Compile LaTeX report
echo -e "${YELLOW}Compiling LaTeX report...${NC}"

TEX_FILE="$REPORT_DIR/LastMile_Report.tex"

if [ ! -f "$TEX_FILE" ]; then
    echo -e "${RED}Error: $TEX_FILE not found${NC}"
    exit 1
fi

cd "$REPORT_DIR"

echo -e "  → Running tectonic (may take a moment)..."

# Run tectonic to compile PDF (run twice for references)
tectonic -X compile "LastMile_Report.tex"

if [ $? -eq 0 ]; then
    # Run second pass for references
    tectonic -X compile "LastMile_Report.tex"
fi

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ PDF compiled successfully${NC}"
    echo ""
    echo -e "${GREEN}════════════════════════════════════════${NC}"
    echo -e "${GREEN}  Report Generation Complete!${NC}"
    echo -e "${GREEN}════════════════════════════════════════${NC}"
    echo ""
    echo -e "Output files:"
    echo -e "  📄 PDF: ${GREEN}$REPORT_DIR/LastMile_Report.pdf${NC}"
    echo -e "  🖼️  Diagrams: ${GREEN}$DIAGRAMS_DIR/*.png${NC}"
    echo ""
    
    # Open PDF if on macOS
    if [[ "$OSTYPE" == "darwin"* ]]; then
        echo -e "${YELLOW}Opening PDF...${NC}"
        open "$REPORT_DIR/LastMile_Report.pdf"
    fi
else
    echo -e "${RED}✗ PDF compilation failed${NC}"
    echo -e "${YELLOW}Check the LaTeX log for errors${NC}"
    exit 1
fi
