package az

import (
	"testing"
	"time"
)

func TestInferenceMetadataMatchesExplicitRequest(t *testing.T) {
	base := InferenceMetadata{
		ModelID: "random-test", ModelConfig: "large", Device: "mps",
		DType: "float32", Parameters: 29_114_114, TorchThreads: 4,
	}
	tests := []struct {
		name    string
		options PythonEvaluatorOptions
		wantErr bool
	}{
		{name: "exact", options: PythonEvaluatorOptions{ModelConfig: "large", Device: "mps", TorchThreads: 4}},
		{name: "auto_resolves", options: PythonEvaluatorOptions{ModelConfig: "auto", Device: "auto", TorchThreads: 0}},
		{name: "wrong_config", options: PythonEvaluatorOptions{ModelConfig: "debug", Device: "mps", TorchThreads: 4}, wantErr: true},
		{name: "wrong_device", options: PythonEvaluatorOptions{ModelConfig: "large", Device: "cpu", TorchThreads: 4}, wantErr: true},
		{name: "wrong_threads", options: PythonEvaluatorOptions{ModelConfig: "large", Device: "mps", TorchThreads: 1}, wantErr: true},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := validateInferenceMetadata(base, testCase.options)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("validation error = %v, wantErr=%v", err, testCase.wantErr)
			}
		})
	}
}

func TestInferenceDurationWindowIsBoundedAndRecent(t *testing.T) {
	var window durationWindow
	for index := 0; index < inferenceDurationWindowSize+904; index++ {
		window.observe(time.Duration(index) * time.Millisecond)
	}
	samples := window.samples()
	if len(samples) != inferenceDurationWindowSize {
		t.Fatalf("sample count = %d, want %d", len(samples), inferenceDurationWindowSize)
	}
	if got := durationPercentile(samples, 0); got != 904*time.Millisecond {
		t.Fatalf("oldest retained latency = %v, want 904ms", got)
	}
	if got := durationPercentile(samples, 1); got != 4999*time.Millisecond {
		t.Fatalf("newest retained latency = %v, want 4999ms", got)
	}
}
