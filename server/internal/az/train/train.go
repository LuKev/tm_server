package train

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/lukev/tm_server/internal/az/model"
	"github.com/lukev/tm_server/internal/az/selfplay"
)

type aggregate struct {
	policy map[string]float64
	value  float64
	count  int
}

// TrainFile trains the bootstrap table evaluator from self-play JSONL.
func TrainFile(path string) (*model.TableModel, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	buckets := make(map[string]*aggregate)
	globalPolicy := make(map[string]float64)
	globalCount := 0
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*1024), 32*1024*1024)
	for scanner.Scan() {
		var record selfplay.Record
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return nil, err
		}
		key := model.EncodingKey(record.Encoding)
		bucket := buckets[key]
		if bucket == nil {
			bucket = &aggregate{policy: make(map[string]float64)}
			buckets[key] = bucket
		}
		bucket.count++
		bucket.value += record.Outcome
		for actionID, prob := range record.Policy {
			bucket.policy[actionID] += prob
			globalPolicy[actionID] += prob
		}
		globalCount++
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if globalCount == 0 {
		return nil, fmt.Errorf("no records in %s", path)
	}
	out := &model.TableModel{
		Buckets:      make(map[string]model.TableBucket, len(buckets)),
		GlobalPolicy: make(map[string]float64, len(globalPolicy)),
	}
	for key, bucket := range buckets {
		policy := make(map[string]float64, len(bucket.policy))
		for actionID, sum := range bucket.policy {
			policy[actionID] = sum / float64(bucket.count)
		}
		out.Buckets[key] = model.TableBucket{
			Policy: policy,
			Value:  bucket.value / float64(bucket.count),
			Count:  bucket.count,
		}
	}
	for actionID, sum := range globalPolicy {
		out.GlobalPolicy[actionID] = sum / float64(globalCount)
	}
	return out, nil
}
