# LastMile Report - README

## Overview

This directory contains the LaTeX source and PlantUML diagrams for the comprehensive LastMile project report.

## Contents

- **`LastMile_Report.tex`** - Main LaTeX report document
- **`diagrams/`** - PlantUML source files and rendered PNG diagrams
  - `system_architecture.puml` - High-level system architecture
  - `deployment_architecture.puml` - Kubernetes deployment architecture
  - `load_balancing.puml` - gRPC client-side load balancing sequence
  - `matching_flow.puml` - Rider-driver matching flow
- **`images/`** - Additional images and screenshots (if any)

## Prerequisites

Install the required tools:

```bash
# macOS
brew install plantuml tectonic

# Ubuntu/Debian
sudo apt-get install plantuml
curl --proto '=https' --tlsv1.2 -fsSL https://drop-sh.fullyjustified.net | sh
```

## Building the Report

From the project root, run:

```bash
./render_report.sh
```

This script will:
1. ✅ Check for required dependencies (plantuml, tectonic)
2. 🖼️ Render all `.puml` files to PNG diagrams
3. 📄 Compile the LaTeX document to PDF
4. 🚀 Open the PDF automatically (macOS only)

## Output

The generated PDF will be located at:
```
docs/report/LastMile_Report.pdf
```

Rendered diagrams:
```
docs/report/diagrams/*.png
```

## Report Sections

1. **Executive Summary** - Project overview and key achievements
2. **System Architecture** - High-level architecture and service communication
3. **Microservices Design** - Detailed breakdown of all 8 services
4. **Deployment Architecture** - Kubernetes resources and configuration
5. **Scalability & Load Balancing** - HPA and client-side load balancing
6. **Technology Stack** - Tools, frameworks, and languages used
7. **AI-Assisted Development** - Role of Google Gemini in accelerating delivery
8. **Challenges & Solutions** - Technical obstacles and resolutions
9. **Future Enhancements** - Planned improvements
10. **Conclusion** - Summary and lessons learned
11. **References** - Citations and links
12. **Appendix** - Deployment commands and API examples

## Manual Rendering

If you prefer to run commands manually:

```bash
# Render PlantUML diagrams
cd docs/report/diagrams
plantuml -tpng system_architecture.puml
plantuml -tpng deployment_architecture.puml
plantuml -tpng load_balancing.puml
plantuml -tpng matching_flow.puml

# Compile LaTeX
cd ..
tectonic -X compile LastMile_Report.tex
```

## Troubleshooting

### PlantUML Rendering Issues

If diagrams don't render properly:
```bash
# Check PlantUML version
plantuml -version

# Test individual file
plantuml -tpng diagrams/system_architecture.puml
```

### LaTeX Compilation Errors

If tectonic fails:
```bash
# Run with verbose output
tectonic -X compile --keep-logs LastMile_Report.tex

# Check the log
cat LastMile_Report.log
```

### Missing Kubernetes Sprites

The deployment diagram uses Kubernetes PlantUML sprites. If they fail to load, the diagram will still render but without icons.

## Customization

### Adding New Diagrams

1. Create a new `.puml` file in `docs/report/diagrams/`
2. Add the file to the `PUML_FILES` array in `render_report.sh`
3. Reference the diagram in the LaTeX file:
   ```latex
   \begin{figure}[H]
       \centering
       \includegraphics[width=\textwidth]{diagrams/your_diagram.png}
       \caption{Your Diagram Caption}
       \label{fig:your_label}
   \end{figure}
   ```

### Modifying Report Content

Edit `LastMile_Report.tex` directly. The document uses standard LaTeX commands and packages.

## License

This report is part of the LastMile project. See the main repository LICENSE file.

## Contact

**Author:** Pradyun Devarakonda  
**Repository:** https://github.com/pradyunuydarp/LastMile
