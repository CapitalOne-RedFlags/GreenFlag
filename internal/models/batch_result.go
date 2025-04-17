package models

type BatchResult struct {
	BatchItemFailures []BatchItemFailure `json:"BatchItemFailures"`
}

type BatchItemFailure struct {
	ItemIdentifier string `json:"ItemIdentifier"`
}

func (br BatchResult) GetRids() []string {
	rids := []string{}

	for _, batchItemFailure := range br.BatchItemFailures {
		rids = append(rids, batchItemFailure.ItemIdentifier)
	}

	return rids
}
