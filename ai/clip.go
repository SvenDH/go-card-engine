package ai

import (
	"encoding/json"
	"fmt"
	"image"
	"math"
	"os"
	"strings"

	"github.com/anthonynsimon/bild/transform"
	ort "github.com/yalue/onnxruntime_go"
)

// CLIPModel provides text and vision encoding using CLIP
type CLIPModel struct {
	textSession   *ort.AdvancedSession
	visionSession *ort.AdvancedSession
	tokenizer     *CLIPTokenizer
	textInput     *ort.Tensor[int64]
	textOutput    *ort.Tensor[float32]
	visionInput   *ort.Tensor[float32]
	visionOutput  *ort.Tensor[float32]
	embeddingDim  int
}

// CLIPTokenizer handles CLIP text tokenization
type CLIPTokenizer struct {
	vocab      map[string]int
	merges     [][]string
	startToken int
	endToken   int
	maxLength  int
}

// NewCLIPModel creates a new CLIP model from ONNX files
func NewCLIPModel(textModelPath, visionModelPath, tokenizerPath string) (*CLIPModel, error) {
	// Initialize ONNX Runtime
	ort.SetSharedLibraryPath("lib/libonnxruntime.so")
	err := ort.InitializeEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ONNX Runtime: %w", err)
	}

	// Load tokenizer
	tokenizer, err := LoadCLIPTokenizer(tokenizerPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load tokenizer: %w", err)
	}

	// Create text encoder session
	textInput, err := ort.NewEmptyTensor[int64](ort.NewShape(1, 77)) // CLIP text length is 77
	if err != nil {
		return nil, fmt.Errorf("failed to create text input tensor: %w", err)
	}

	textOutput, err := ort.NewEmptyTensor[float32](ort.NewShape(1, 512)) // CLIP ViT-B/32 embedding dim is 512
	if err != nil {
		textInput.Destroy()
		return nil, fmt.Errorf("failed to create text output tensor: %w", err)
	}

	sessionOptions, err := ort.NewSessionOptions()
	if err != nil {
		textInput.Destroy()
		textOutput.Destroy()
		return nil, fmt.Errorf("failed to create session options: %w", err)
	}
	defer sessionOptions.Destroy()

	sessionOptions.SetIntraOpNumThreads(4)

	// CLIP text encoder typically uses "input_ids" as input and "text_embeds" or "last_hidden_state" as output
	textSession, err := ort.NewAdvancedSession(textModelPath,
		[]string{"input_ids"},
		[]string{"text_embeds"},
		[]ort.Value{textInput},
		[]ort.Value{textOutput},
		sessionOptions)
	if err != nil {
		textInput.Destroy()
		textOutput.Destroy()
		return nil, fmt.Errorf("failed to create text session: %w", err)
	}

	// Create vision encoder session
	visionInput, err := ort.NewEmptyTensor[float32](ort.NewShape(1, 3, 224, 224))
	if err != nil {
		textSession.Destroy()
		return nil, fmt.Errorf("failed to create vision input tensor: %w", err)
	}

	visionOutput, err := ort.NewEmptyTensor[float32](ort.NewShape(1, 512))
	if err != nil {
		textSession.Destroy()
		visionInput.Destroy()
		return nil, fmt.Errorf("failed to create vision output tensor: %w", err)
	}

	sessionOptions2, err := ort.NewSessionOptions()
	if err != nil {
		textSession.Destroy()
		visionInput.Destroy()
		visionOutput.Destroy()
		return nil, fmt.Errorf("failed to create session options: %w", err)
	}
	defer sessionOptions2.Destroy()

	sessionOptions2.SetIntraOpNumThreads(4)

	// CLIP vision encoder typically uses "pixel_values" as input and "image_embeds" as output
	visionSession, err := ort.NewAdvancedSession(visionModelPath,
		[]string{"pixel_values"},
		[]string{"image_embeds"},
		[]ort.Value{visionInput},
		[]ort.Value{visionOutput},
		sessionOptions2)
	if err != nil {
		textSession.Destroy()
		visionInput.Destroy()
		visionOutput.Destroy()
		return nil, fmt.Errorf("failed to create vision session: %w", err)
	}

	return &CLIPModel{
		textSession:   textSession,
		visionSession: visionSession,
		tokenizer:     tokenizer,
		textInput:     textInput,
		textOutput:    textOutput,
		visionInput:   visionInput,
		visionOutput:  visionOutput,
		embeddingDim:  512,
	}, nil
}

// Close releases CLIP resources
func (c *CLIPModel) Close() error {
	if c.textSession != nil {
		c.textSession.Destroy()
	}
	if c.visionSession != nil {
		c.visionSession.Destroy()
	}
	return nil
}

// EncodeText encodes text into a CLIP embedding
func (c *CLIPModel) EncodeText(text string) ([]float32, error) {
	// Tokenize text
	tokens := c.tokenizer.Encode(text)

	// Copy tokens to input tensor
	inputData := c.textInput.GetData()
	for i := range inputData {
		if i < len(tokens) {
			inputData[i] = int64(tokens[i])
		} else {
			inputData[i] = 0 // Padding
		}
	}

	// Run inference
	err := c.textSession.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run text inference: %w", err)
	}

	// Get output
	outputData := c.textOutput.GetData()
	embedding := make([]float32, len(outputData))
	copy(embedding, outputData)

	// Normalize to unit length
	return normalizeVectorCLIP(embedding), nil
}

// EncodeImage encodes an image into a CLIP embedding
func (c *CLIPModel) EncodeImage(img image.Image) ([]float32, error) {
	// Preprocess image (resize to 224x224, normalize)
	inputData := c.preprocessImage(img)

	// Copy to input tensor
	copy(c.visionInput.GetData(), inputData)

	// Run inference
	err := c.visionSession.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run vision inference: %w", err)
	}

	// Get output
	outputData := c.visionOutput.GetData()
	embedding := make([]float32, len(outputData))
	copy(embedding, outputData)

	// Normalize to unit length
	return normalizeVectorCLIP(embedding), nil
}

// preprocessImage prepares an image for CLIP vision encoder
func (c *CLIPModel) preprocessImage(img image.Image) []float32 {
	// Resize to 224x224
	resized := transform.Resize(img, 224, 224, transform.Linear)

	// CLIP uses different normalization than ImageNet
	meanNorm := []float32{0.48145466, 0.4578275, 0.40821073}
	stdNorm := []float32{0.26862954, 0.26130258, 0.27577711}

	tensor := make([]float32, 3*224*224)

	for y := 0; y < 224; y++ {
		for x := 0; x < 224; x++ {
			r, g, b, _ := resized.At(x, y).RGBA()

			// Convert to 0-1 range
			rf := float32(r) / 65535.0
			gf := float32(g) / 65535.0
			bf := float32(b) / 65535.0

			// Apply CLIP normalization
			rf = (rf - meanNorm[0]) / stdNorm[0]
			gf = (gf - meanNorm[1]) / stdNorm[1]
			bf = (bf - meanNorm[2]) / stdNorm[2]

			// Store in CHW format
			idx := y*224 + x
			tensor[0*224*224+idx] = rf
			tensor[1*224*224+idx] = gf
			tensor[2*224*224+idx] = bf
		}
	}

	return tensor
}

// LoadCLIPTokenizer loads the CLIP tokenizer from JSON
func LoadCLIPTokenizer(path string) (*CLIPTokenizer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read tokenizer file: %w", err)
	}

	var tokenizerData struct {
		Model struct {
			Vocab map[string]int `json:"vocab"`
		} `json:"model"`
		AddedTokens []struct {
			ID      int    `json:"id"`
			Content string `json:"content"`
		} `json:"added_tokens"`
	}

	err = json.Unmarshal(data, &tokenizerData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tokenizer JSON: %w", err)
	}

	return &CLIPTokenizer{
		vocab:      tokenizerData.Model.Vocab,
		startToken: 49406, // CLIP start token
		endToken:   49407, // CLIP end token
		maxLength:  77,    // CLIP context length
	}, nil
}

// Encode tokenizes text into token IDs
func (t *CLIPTokenizer) Encode(text string) []int {
	// Basic BPE tokenization (simplified)
	text = strings.ToLower(strings.TrimSpace(text))

	tokens := make([]int, t.maxLength)
	tokens[0] = t.startToken

	// Split into words and tokenize
	words := strings.Fields(text)
	tokenIdx := 1

	for _, word := range words {
		// Try to find word in vocab
		word = word + "</w>" // BPE end-of-word marker
		if id, ok := t.vocab[word]; ok && tokenIdx < t.maxLength-1 {
			tokens[tokenIdx] = id
			tokenIdx++
		} else {
			// Fallback: character-level tokenization
			for _, ch := range word {
				charStr := string(ch)
				if id, ok := t.vocab[charStr]; ok && tokenIdx < t.maxLength-1 {
					tokens[tokenIdx] = id
					tokenIdx++
				}
			}
		}
	}

	// Add end token
	if tokenIdx < t.maxLength {
		tokens[tokenIdx] = t.endToken
	}

	return tokens
}

// normalizeVectorCLIP normalizes a vector to unit length
func normalizeVectorCLIP(v []float32) []float32 {
	norm := float32(0)
	for _, val := range v {
		norm += val * val
	}
	norm = float32(math.Sqrt(float64(norm)))

	if norm > 0 {
		for i := range v {
			v[i] /= norm
		}
	}

	return v
}

// CLIPSimilarity computes cosine similarity between two CLIP embeddings
func CLIPSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	dot := float32(0)
	for i := range a {
		dot += a[i] * b[i]
	}

	return dot // Already normalized, so dot product = cosine similarity
}
