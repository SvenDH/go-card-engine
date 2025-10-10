# go-card-engine

A card game engine with a card text parser for a made up card game, featuring AI-powered tilemap generation.

The parser uses [Participle](https://github.com/alecthomas/participle) for defining the card grammar and creating the AST.

## Features

- 🎴 **Card Game Engine** - Custom card game logic and rules
- 🎨 **AI-Powered Tilemap Generation** - Convert images to tilemaps with neural networks
- 🧬 **Evolutionary Algorithm** - Global optimization of tile placement
- 🤖 **Semantic Matching** - Use pre-trained ONNX models (SqueezeNet, etc.)
- 🔄 **Live Preview** - Real-time visualization during generation

## Quick Start: AI Tilemap Generation

### 1. Setup

```bash
# Install dependencies
go mod download

# Download ONNX Runtime and SqueezeNet model
./setup_onnx.sh
```

### 2. Generate a Tilemap

**Basic generation:**
```bash
go run . genlive -W 14 -H 20 assets/wizard.jpg -c 6
```

**With AI embeddings (best quality):**
```bash
go run . genlive -W 14 -H 20 assets/wizard.jpg \
  --use-embeddings \
  --model models/squeezenet1.1-7.onnx \
  --embedding-weight 0.6 \
  -c 6
```

**With evolution (highest quality):**
```bash
go run . genlive -W 14 -H 20 assets/wizard.jpg \
  --use-embeddings \
  --model models/squeezenet1.1-7.onnx \
  --evolve \
  --pop-size 50 \
  --generations 100 \
  -c 6
```

## Documentation

- **[Pretrained Models](docs/PRETRAINED_MODELS.md)** - Using ONNX models for semantic matching
- **[Patch Embeddings](docs/PATCH_EMBEDDINGS.md)** - How patch-based embeddings work
- **[Evolutionary Algorithm](docs/EVOLUTIONARY_ALGORITHM.md)** - Genetic algorithm optimization
- **[Warm Start](docs/WARM_START.md)** - Evolution with greedy initialization
- **[ONNX Runtime Setup](lib/README.md)** - Installing ONNX Runtime library

## Requirements

- Go 1.21+
- ONNX Runtime 1.20+ (for AI features)
- Pre-trained ONNX models (e.g., SqueezeNet)

## Status

Project is WIP - actively developing AI-powered tilemap generation features.
