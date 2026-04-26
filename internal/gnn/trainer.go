package gnn

import (
	"math"
	"math/rand"
)

// TrainingResult captures the output of a training run.
type TrainingResult struct {
	LossHistory []float64 // loss per epoch
	FinalLoss   float64   // loss at the final epoch
	EpochsRun   int
}

// Trainer runs BPR (Bayesian Personalized Ranking) optimization on a
// LightGCN model for link prediction.
//
// BPR loss for a triple (u, v+, v-):
//
//	loss = -log(sigmoid(s(u,v+) - s(u,v-)))
//
// where s(u,v) = dot(H[u], H[v]) is the link prediction score.
type Trainer struct {
	Model     *GNNModel
	LR        float64 // initial learning rate
	MinLR     float64 // floor for learning rate decay
	Decay     float64 // multiplicative decay per epoch (e.g. 0.99)
	BatchSize int
	rng       *rand.Rand
}

// NewTrainer creates a Trainer with sensible defaults.
func NewTrainer(model *GNNModel, lr float64) *Trainer {
	return &Trainer{
		Model:     model,
		LR:        lr,
		MinLR:     1e-6,
		Decay:     0.99,
		BatchSize: 32,
		rng:       rand.New(rand.NewSource(42)),
	}
}

// Train runs the full training loop for the given number of epochs.
// It shuffles edges each epoch and processes them in mini-batches.
func (t *Trainer) Train(posEdges, negEdges [][2]int, epochs int) TrainingResult {
	lossHistory := make([]float64, epochs)
	numPos := len(posEdges)
	numNeg := len(negEdges)

	maxLen := numPos
	if numNeg > maxLen {
		maxLen = numNeg
	}

	copyPos := make([][2]int, numPos)
	copyNeg := make([][2]int, numNeg)
	copy(copyPos, posEdges)
	copy(copyNeg, negEdges)

	for ep := 0; ep < epochs; ep++ {
		t.rng.Shuffle(len(copyPos), func(i, j int) { copyPos[i], copyPos[j] = copyPos[j], copyPos[i] })
		t.rng.Shuffle(len(copyNeg), func(i, j int) { copyNeg[i], copyNeg[j] = copyNeg[j], copyNeg[i] })

		var epochLoss float64
		var batches int

		for start := 0; start < maxLen; start += t.BatchSize {
			end := start + t.BatchSize
			if end > maxLen {
				end = maxLen
			}
			batchSize := end - start

			posBatch := make([][2]int, batchSize)
			negBatch := make([][2]int, batchSize)
			for i := 0; i < batchSize; i++ {
				posIdx := (start + i) % numPos
				negIdx := (start + i) % numNeg
				posBatch[i] = copyPos[posIdx]
				negBatch[i] = copyNeg[negIdx]
			}

			loss := t.step(posBatch, negBatch)
			epochLoss += loss
			batches++
		}

		if batches > 0 {
			lossHistory[ep] = epochLoss / float64(batches)
		}

		t.LR = math.Max(t.MinLR, t.LR*t.Decay)
	}

	return TrainingResult{
		LossHistory: lossHistory,
		FinalLoss:   lossHistory[epochs-1],
		EpochsRun:   epochs,
	}
}

// step performs one gradient update on a mini-batch.
// Computes BPR loss, backpropagates through the GNN, and applies SGD.
func (t *Trainer) step(posBatch, negBatch [][2]int) float64 {
	n := t.Model.NumNodes
	d := t.Model.Dim
	batchSize := len(posBatch)

	emb := t.Model.Forward()

	gradFinal := make([][]float64, n)
	for i := range gradFinal {
		gradFinal[i] = make([]float64, d)
	}

	var totalLoss float64

	for k := 0; k < batchSize; k++ {
		u := posBatch[k][0]
		v := posBatch[k][1]
		nv := negBatch[k][0]

		posScore := dotProduct(emb[u], emb[v])
		negScore := dotProduct(emb[u], emb[nv])
		x := posScore - negScore

		sigx := sigmoid(x)
		if sigx <= 0 {
			sigx = 1e-15
		}
		totalLoss -= math.Log(sigx)

		dLdx := -(1.0 - sigx)

		for j := 0; j < d; j++ {
			gradFinal[u][j] += dLdx * (emb[v][j] - emb[nv][j])
			gradFinal[v][j] += dLdx * emb[u][j]
			gradFinal[nv][j] -= dLdx * emb[u][j]
		}
	}

	invBatch := 1.0 / float64(batchSize)
	for i := range gradFinal {
		for j := range gradFinal[i] {
			gradFinal[i][j] *= invBatch
		}
	}

	gradEmb := t.Model.Backward(gradFinal)

	lr := t.LR
	for i := range t.Model.Embeddings {
		for j := range t.Model.Embeddings[i] {
			t.Model.Embeddings[i][j] -= lr * gradEmb[i][j]
		}
	}

	return totalLoss / float64(batchSize)
}
