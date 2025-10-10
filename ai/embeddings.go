package ai

import (
	"fmt"
	"image"
	"math"

	"github.com/anthonynsimon/bild/transform"
	ort "github.com/yalue/onnxruntime_go"
)

// TileEmbedding represents a tile's learned features
type TileEmbedding struct {
	Features []float32
	TileIdx  int
}

// ONNXEmbedder uses a pre-trained ONNX model for embeddings
type ONNXEmbedder struct {
	session      *ort.AdvancedSession
	inputTensor  *ort.Tensor[float32]
	outputTensor *ort.Tensor[float32]
	inputShape   []int64
	outputShape  []int64
	inputName    string
	outputName   string
	embeddingDim int
	meanNorm     []float32
	stdNorm      []float32
}

// NewONNXEmbedder creates an embedder from an ONNX model file
func NewONNXEmbedder(modelPath string) (*ONNXEmbedder, error) {
	// Initialize ONNX Runtime
	ort.SetSharedLibraryPath("lib/libonnxruntime.so")
	err := ort.InitializeEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ONNX Runtime: %w", err)
	}

	// Get input shape (typically [1, 3, 224, 224] for SqueezeNet)
	inputShape := []int64{1, 3, 224, 224}
	outputShape := []int64{1, 1000}

	// Create input/output tensors
	inputTensor, err := ort.NewEmptyTensor[float32](ort.NewShape(inputShape...))
	if err != nil {
		return nil, fmt.Errorf("failed to create input tensor: %w", err)
	}

	outputTensor, err := ort.NewEmptyTensor[float32](ort.NewShape(outputShape...))
	if err != nil {
		inputTensor.Destroy()
		return nil, fmt.Errorf("failed to create output tensor: %w", err)
	}

	// Create session options
	sessionOptions, err := ort.NewSessionOptions()
	if err != nil {
		inputTensor.Destroy()
		outputTensor.Destroy()
		return nil, fmt.Errorf("failed to create session options: %w", err)
	}
	defer sessionOptions.Destroy()

	// Set optimization level
	err = sessionOptions.SetIntraOpNumThreads(4)
	if err != nil {
		inputTensor.Destroy()
		outputTensor.Destroy()
		return nil, fmt.Errorf("failed to set thread count: %w", err)
	}

	// Load the model
	session, err := ort.NewAdvancedSession(modelPath, 
		[]string{"data"}, // Input name for SqueezeNet
		[]string{"squeezenet0_flatten0_reshape0"}, // Output name for SqueezeNet 1.1
		[]ort.Value{inputTensor}, 
		[]ort.Value{outputTensor},
		sessionOptions)
	if err != nil {
		inputTensor.Destroy()
		outputTensor.Destroy()
		return nil, fmt.Errorf("failed to create ONNX session: %w", err)
	}

	// ImageNet normalization (standard for pre-trained models)
	meanNorm := []float32{0.485, 0.456, 0.406}
	stdNorm := []float32{0.229, 0.224, 0.225}

	return &ONNXEmbedder{
		session:      session,
		inputTensor:  inputTensor,
		outputTensor: outputTensor,
		inputShape:   inputShape,
		outputShape:  outputShape,
		inputName:    "data",
		outputName:   "squeezenet0_flatten0_reshape0",
		embeddingDim: 1000, // SqueezeNet outputs 1000 classes, we'll use as embedding
		meanNorm:     meanNorm,
		stdNorm:      stdNorm,
	}, nil
}

// Close releases ONNX resources
func (e *ONNXEmbedder) Close() error {
	if e.session != nil {
		e.session.Destroy()
	}
	return nil
}

// preprocessImage converts an image to the format expected by the model
func (e *ONNXEmbedder) preprocessImage(img image.Image) []float32 {
	// Get input dimensions
	height := int(e.inputShape[2])
	width := int(e.inputShape[3])

	// Resize image to model input size
	resized := transform.Resize(img, width, height, transform.Linear)

	// Convert to float32 tensor in CHW format (channels, height, width)
	tensor := make([]float32, 3*height*width)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := resized.At(x, y).RGBA()

			// Convert to 0-1 range
			rf := float32(r) / 65535.0
			gf := float32(g) / 65535.0
			bf := float32(b) / 65535.0

			// Apply ImageNet normalization
			rf = (rf - e.meanNorm[0]) / e.stdNorm[0]
			gf = (gf - e.meanNorm[1]) / e.stdNorm[1]
			bf = (bf - e.meanNorm[2]) / e.stdNorm[2]

			// Store in CHW format
			idx := y*width + x
			tensor[0*height*width+idx] = rf // R channel
			tensor[1*height*width+idx] = gf // G channel
			tensor[2*height*width+idx] = bf // B channel
		}
	}

	return tensor
}

// ComputeEmbedding generates an embedding vector for an image
func (e *ONNXEmbedder) ComputeEmbedding(img image.Image) ([]float32, error) {
	// Preprocess image
	inputData := e.preprocessImage(img)
	
	// Copy data to input tensor
	copy(e.inputTensor.GetData(), inputData)

	// Run inference
	err := e.session.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run inference: %w", err)
	}

	// Copy output data
	outputData := e.outputTensor.GetData()
	embedding := make([]float32, len(outputData))
	copy(embedding, outputData)

	// Normalize embedding to unit length
	return normalizeVectorONNX(embedding), nil
}

// normalizeVector normalizes a vector to unit length
func normalizeVectorONNX(v []float32) []float32 {
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

// CosineSimilarity computes cosine similarity between two embeddings
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}
	
	dot := float32(0)
	for i := range a {
		dot += a[i] * b[i]
	}
	
	return dot // Already normalized vectors
}
