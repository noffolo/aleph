package graphbuilder

import "math"

type TrendReport struct {
	FundingConcentration []FundingConcentration `json:"funding_concentration"`
	PartyFundingShift    []PartyFundingShift     `json:"party_funding_shift"`
	DonorAnomalies       []DonorAnomaly          `json:"donor_anomalies"`
	VoteFundingRatio     []VoteFundingRatio      `json:"vote_funding_ratio"`
}

type FundingConcentration struct {
	Year       int     `json:"year"`
	DonorType  string  `json:"donor_type"`
	Amount     float64 `json:"amount"`
	Percentage float64 `json:"percentage"`
}

type PartyFundingShift struct {
	Party          string   `json:"party"`
	Year           int      `json:"year"`
	Amount         float64  `json:"amount"`
	ChangeFromPrev *float64 `json:"change_from_prev,omitempty"`
}

type DonorAnomaly struct {
	DonorName   string   `json:"donor_name"`
	Count       int      `json:"count"`
	TotalAmount float64  `json:"total_amount"`
	PartyCount  int      `json:"party_count"`
	Flag        string   `json:"flag"`
}

type VoteFundingRatio struct {
	Party        string  `json:"party"`
	Year         int     `json:"year"`
	CostPerVote  float64 `json:"cost_per_vote"`
	TotalFunding float64 `json:"total_funding"`
}

func (b *PoliticalGraphBuilder) AnalyzeTrends() (*TrendReport, error) {
	report := &TrendReport{}

	fc, err := b.analyzeFundingConcentration()
	if err != nil {
		return nil, err
	}
	report.FundingConcentration = fc

	pfs, err := b.analyzePartyFundingShift()
	if err != nil {
		return nil, err
	}
	report.PartyFundingShift = pfs

	da, err := b.analyzeDonorAnomalies()
	if err != nil {
		return nil, err
	}
	report.DonorAnomalies = da

	vfr, err := b.analyzeVoteFundingRatio()
	if err != nil {
		return nil, err
	}
	report.VoteFundingRatio = vfr

	return report, nil
}

func (b *PoliticalGraphBuilder) analyzeFundingConcentration() ([]FundingConcentration, error) {
	rows, err := b.db.Query(`SELECT donation_year, donor_type, COALESCE(SUM(donation_amount), 0) as amount
		FROM party_funding GROUP BY donation_year, donor_type ORDER BY donation_year, donor_type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var totalByYear map[int]float64 = make(map[int]float64)
	var results []FundingConcentration
	for rows.Next() {
		var year int
		var donorType string
		var amount float64
		if err := rows.Scan(&year, &donorType, &amount); err != nil {
			return nil, err
		}
		totalByYear[year] += amount
		results = append(results, FundingConcentration{
			Year:      year,
			DonorType: donorType,
			Amount:    amount,
		})
	}
	for i := range results {
		if total := totalByYear[results[i].Year]; total > 0 {
			results[i].Percentage = (results[i].Amount / total) * 100
		}
	}
	return results, nil
}

func (b *PoliticalGraphBuilder) analyzePartyFundingShift() ([]PartyFundingShift, error) {
	rows, err := b.db.Query(`SELECT recipient_party, donation_year, COALESCE(SUM(donation_amount), 0) as amount
		FROM party_funding GROUP BY recipient_party, donation_year ORDER BY recipient_party, donation_year`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []PartyFundingShift
	prevAmount := make(map[string]float64)
	for rows.Next() {
		var party string
		var year int
		var amount float64
		if err := rows.Scan(&party, &year, &amount); err != nil {
			return nil, err
		}
		entry := PartyFundingShift{Party: party, Year: year, Amount: amount}
		if prev, ok := prevAmount[party]; ok && prev > 0 {
			change := ((amount - prev) / prev) * 100
			entry.ChangeFromPrev = &change
		}
		prevAmount[party] = amount
		results = append(results, entry)
	}
	return results, nil
}

func (b *PoliticalGraphBuilder) analyzeDonorAnomalies() ([]DonorAnomaly, error) {
	rows, err := b.db.Query(`SELECT donor_name, COUNT(*) as cnt, COALESCE(SUM(donation_amount), 0) as total,
		COUNT(DISTINCT recipient_party) as parties FROM party_funding
		GROUP BY donor_name HAVING COUNT(DISTINCT recipient_party) >= 5`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DonorAnomaly
	for rows.Next() {
		var donorName string
		var count int
		var total float64
		var partyCount int
		if err := rows.Scan(&donorName, &count, &total, &partyCount); err != nil {
			return nil, err
		}
		flag := ""
		if partyCount > 1 {
			flag = "cross_party"
		} else if count > 10 {
			flag = "high_frequency"
		}
		results = append(results, DonorAnomaly{
			DonorName:   donorName,
			Count:       count,
			TotalAmount: total,
			PartyCount:  partyCount,
			Flag:        flag,
		})
	}
	return results, nil
}

func (b *PoliticalGraphBuilder) analyzeVoteFundingRatio() ([]VoteFundingRatio, error) {
	rows, err := b.db.Query(`
		SELECT
			LOWER(TRIM(descrizione)) as party,
			COALESCE(SUM(voti_validi), 0) as total_votes,
			COALESCE(SUM(perc), 0) as total_perc
		FROM election_results_2022_camera
		GROUP BY LOWER(TRIM(descrizione))`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type voteInfo struct {
		totalVotes float64
		totalPerc  float64
	}
	voteMap := make(map[string]voteInfo)
	for rows.Next() {
		var party string
		var votes, perc float64
		if err := rows.Scan(&party, &votes, &perc); err != nil {
			return nil, err
		}
		voteMap[party] = voteInfo{totalVotes: votes, totalPerc: perc}
	}

	rows2, err := b.db.Query(`SELECT LOWER(TRIM(recipient_party)) as party, donation_year, COALESCE(SUM(donation_amount), 0) as total
		FROM party_funding GROUP BY LOWER(TRIM(recipient_party)), donation_year`)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()

	var results []VoteFundingRatio
	for rows2.Next() {
		var party string
		var year int
		var funding float64
		if err := rows2.Scan(&party, &year, &funding); err != nil {
			return nil, err
		}
		vi, ok := voteMap[party]
		if !ok || vi.totalVotes == 0 {
			continue
		}
		costPerVote := funding / vi.totalVotes
		results = append(results, VoteFundingRatio{
			Party:        party,
			Year:         year,
			CostPerVote:  math.Round(costPerVote*10000) / 10000,
			TotalFunding: funding,
		})
	}
	return results, nil
}
