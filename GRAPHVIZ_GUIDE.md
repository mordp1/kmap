# Graphviz DOT Export Guide

## Overview

kmap can export Topic-Consumer relationships as Graphviz DOT files, which can be converted to high-quality images suitable for large clusters.

## Why Use DOT Export?

### HTML Report Limitations
- Focused on tables and broker metrics
- No visualization for large clusters
- Not suitable for external documentation

### DOT Format Advantages
✅ **Scalable** - SVG output scales infinitely
✅ **High-resolution** - Generate 300+ DPI images
✅ **Professional** - Automatic layout optimization
✅ **Flexible** - Convert to PNG, SVG, PDF
✅ **Tool-compatible** - Open in many visualization tools

## Quick Start

### 1. Generate DOT File

```bash
kmap -brokers kafka:9092 -dot kafka-topology.dot
```

### 2. Convert to Image

**Using helper script:**
```bash
./dot-to-image.sh kafka-topology.dot
```

This creates:
- `kafka-topology.png` - High-res PNG (300 DPI)
- `kafka-topology.svg` - Scalable vector graphics
- `kafka-topology.pdf` - Print-ready PDF

**Manual conversion:**
```bash
# PNG - Good for embedding in documents
dot -Tpng -Gdpi=300 kafka-topology.dot -o diagram.png

# SVG - Best for web, infinite zoom
dot -Tsvg kafka-topology.dot -o diagram.svg

# PDF - Best for printing
dot -Tpdf kafka-topology.dot -o diagram.pdf
```

## Installation

### Graphviz

**macOS:**
```bash
brew install graphviz
```

**Ubuntu/Debian:**
```bash
sudo apt-get update
sudo apt-get install graphviz
```

**RHEL/CentOS:**
```bash
sudo yum install graphviz
```

**Windows:**
```bash
# With Chocolatey
choco install graphviz

# Or download installer from:
# https://graphviz.org/download/
```

**Verify installation:**
```bash
dot -V
# Should output: dot - graphviz version X.X.X
```

## Viewing Options

### 1. Command-Line Tools (Best)

```bash
# Convert and view in one step
dot -Tpng -Gdpi=300 kafka-topology.dot | open -f -a Preview  # macOS
dot -Tsvg kafka-topology.dot | xdg-open                      # Linux
```

### 2. Online Viewers (No Installation)

**Graphviz Online:**
- https://dreampuf.github.io/GraphvizOnline/
- Paste DOT file content
- Instant preview with pan/zoom

**Edotor:**
- https://edotor.net/
- Split view with live preview
- Download as PNG/SVG

**GraphvizOnline by sketchviz:**
- https://sketchviz.com/
- Hand-drawn style option
- Export to multiple formats

### 3. Desktop Applications

**yEd Graph Editor (Free)**
- Download: https://www.yworks.com/products/yed
- Import → File → Import → DOT
- Advanced layout algorithms
- Export to many formats

**Gephi (Free)**
- Download: https://gephi.org/
- Network analysis features
- Statistics and metrics
- Beautiful exports

**VS Code Extensions**
- "Graphviz Preview" by João Pinto
- "Graphviz (dot) language" by Stephan van Stekelenburg
- Live preview in editor

### 4. Programming Libraries

**Python:**
```python
import graphviz
g = graphviz.Source.from_file('kafka-topology.dot')
g.render('output', format='png', dpi=300)
```

**Node.js:**
```javascript
const viz = require('viz.js');
const fs = require('fs');

const dot = fs.readFileSync('kafka-topology.dot', 'utf8');
viz(dot).then(svg => {
  fs.writeFileSync('output.svg', svg);
});
```

## Advanced Customization

### Layout Algorithms

Change the layout engine for different styles:

```bash
# Default: dot (hierarchical, left-to-right)
dot -Tpng kafka-topology.dot -o diagram.png

# Circular layout
circo -Tpng kafka-topology.dot -o diagram.png

# Force-directed (spring model)
neato -Tpng kafka-topology.dot -o diagram.png

# Radial layout
twopi -Tpng kafka-topology.dot -o diagram.png

# Force-directed with edge concentrators
fdp -Tpng kafka-topology.dot -o diagram.png

# Scalable Force-Directed (large graphs)
sfdp -Tpng kafka-topology.dot -o diagram.png
```

### Image Quality

```bash
# Standard (72 DPI)
dot -Tpng kafka-topology.dot -o diagram-standard.png

# High quality (300 DPI) - print ready
dot -Tpng -Gdpi=300 kafka-topology.dot -o diagram-print.png

# Ultra high (600 DPI) - posters
dot -Tpng -Gdpi=600 kafka-topology.dot -o diagram-poster.png

# Transparent background
dot -Tpng -Gbgcolor=transparent kafka-topology.dot -o diagram-transparent.png
```

### Size Control

```bash
# Limit width (in inches)
dot -Tpng -Gsize="10,20!" kafka-topology.dot -o diagram-sized.png

# Fit to specific dimensions
dot -Tpng -Gsize="8.5,11!" -Gratio=fill kafka-topology.dot -o diagram-letter.png
```

## Large Cluster Tips

For clusters with 200+ topics/consumers:

### 1. Use sfdp Layout
```bash
sfdp -Tsvg kafka-topology.dot -o diagram.svg
```
Better handles large graphs with many edges.

### 2. Increase DPI for Clarity
```bash
dot -Tpng -Gdpi=600 kafka-topology.dot -o diagram.png
```

### 3. Generate SVG First
```bash
dot -Tsvg kafka-topology.dot -o diagram.svg
```
Then convert to PNG/PDF as needed. SVG allows infinite zoom.

### 4. Filter Before Export

Edit DOT file to show only relevant topics:
```bash
# Extract specific topics/consumers
grep -A 1000 "topic_payments\|consumer_processor" kafka-topology.dot > filtered.dot
```

## Automation

### Generate Reports Automatically

```bash
#!/bin/bash
# daily-kafka-report.sh

DATE=$(date +%Y-%m-%d)
OUTPUT_DIR="reports/$DATE"
mkdir -p "$OUTPUT_DIR"

# Generate data
kmap -brokers kafka:9092 \
  -output "$OUTPUT_DIR/cluster.json" \
  -html "$OUTPUT_DIR/report.html" \
  -dot "$OUTPUT_DIR/topology.dot"

# Convert to images
cd "$OUTPUT_DIR"
dot -Tpng -Gdpi=300 topology.dot -o topology.png
dot -Tsvg topology.dot -o topology.svg
dot -Tpdf topology.dot -o topology.pdf

echo "Report generated: $OUTPUT_DIR"
```

### CI/CD Integration

**GitHub Actions:**
```yaml
- name: Generate Kafka Topology
  run: |
    sudo apt-get install -y graphviz
    ./kmap -brokers $KAFKA_BROKERS -dot topology.dot
    dot -Tpng -Gdpi=300 topology.dot -o topology.png
    
- name: Upload Diagram
  uses: actions/upload-artifact@v3
  with:
    name: kafka-topology
    path: topology.png
```

## Troubleshooting

### "command not found: dot"
Graphviz not installed. See Installation section above.

### "Error: Could not parse input as a dot file"
The DOT file is malformed. Regenerate with kmap.

### Graph too large / out of memory
Try:
```bash
# Use sfdp for large graphs
sfdp -Tsvg kafka-topology.dot -o diagram.svg

# Or split into smaller subgraphs
grep -A 100 "cluster_topics" kafka-topology.dot > topics-only.dot
```

### Text overlapping in output
Increase DPI or use SVG:
```bash
dot -Tpng -Gdpi=600 kafka-topology.dot -o diagram.png
# Or
dot -Tsvg kafka-topology.dot -o diagram.svg
```

## Examples

### Small Cluster (< 50 topics)
```bash
kmap -brokers localhost:9092 -dot cluster.dot
dot -Tpng -Gdpi=300 cluster.dot -o cluster.png
open cluster.png
```

### Medium Cluster (50-200 topics)
```bash
kmap -brokers kafka:9092 -dot cluster.dot
dot -Tsvg cluster.dot -o cluster.svg
open cluster.svg  # Better for panning/zooming
```

### Large Cluster (200+ topics)
```bash
kmap -brokers kafka:9092 -dot cluster.dot
sfdp -Tsvg cluster.dot -o cluster.svg
# Or generate ultra-high-res PNG
dot -Tpng -Gdpi=600 cluster.dot -o cluster-uhd.png
```

## Resources

- **Graphviz Official:** https://graphviz.org/
- **Gallery:** https://graphviz.org/gallery/
- **DOT Language:** https://graphviz.org/doc/info/lang.html
- **Online Viewer:** https://dreampuf.github.io/GraphvizOnline/
