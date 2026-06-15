package services

import (
	"math"

	"github.com/tpt-nz/tpt-voter-portal-nz/internal/models"
)

// ComputeIRV runs an Instant Runoff Voting count over a set of ranked ballots.
//
// Each ballot is a slice of choice indices in preference order (most preferred
// first). Ballots that are entirely exhausted (all preferences eliminated) are
// excluded from the active count but counted in total.
//
// The algorithm:
//  1. Count first active preference for each ballot.
//  2. If any candidate has strictly more than 50 % of active votes, they win.
//  3. Eliminate the candidate(s) with the fewest votes.
//  4. Repeat until one candidate remains or all remaining candidates tie exactly.
func ComputeIRV(rankings [][]int, numOptions int) models.IRVResult {
	active := make(map[int]bool, numOptions)
	for i := 0; i < numOptions; i++ {
		active[i] = true
	}

	var rounds []models.IRVRound

	for len(active) > 1 {
		counts := make(map[int]int, len(active))
		for i := range active {
			counts[i] = 0
		}
		activeVotes := 0

		for _, prefs := range rankings {
			for _, choice := range prefs {
				if active[choice] {
					counts[choice]++
					activeVotes++
					break
				}
			}
		}

		// Majority check
		for candidate, count := range counts {
			if count*2 > activeVotes {
				c := candidate
				rounds = append(rounds, models.IRVRound{Counts: counts, TotalActive: activeVotes})
				return models.IRVResult{Winner: &c, Rounds: rounds}
			}
		}

		// Find minimum vote count
		minVotes := math.MaxInt
		for _, count := range counts {
			if count < minVotes {
				minVotes = count
			}
		}

		// Eliminate all candidates at the minimum (handles ties at the bottom)
		var eliminated []int
		for candidate, count := range counts {
			if count == minVotes {
				eliminated = append(eliminated, candidate)
				delete(active, candidate)
			}
		}

		rounds = append(rounds, models.IRVRound{
			Counts:      counts,
			Eliminated:  eliminated,
			TotalActive: activeVotes,
		})

		// If we eliminated everyone, it's an all-tie
		if len(active) == 0 {
			break
		}
	}

	// One candidate remains — declare winner
	var result models.IRVResult
	result.Rounds = rounds
	for candidate := range active {
		c := candidate
		result.Winner = &c
		break
	}
	return result
}
