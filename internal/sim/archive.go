// Model limit and archive (DECISIONS 13b). A player-started training that
// finishes with every slot taken waits in Pending for the player's choice;
// offline runs, policy runs and the AGI model keep the better model and
// archive the other one. No model leaves the game without a player decision.
package sim

import (
	"github.com/prbartosh/token-tycoon/internal/balance"
	"github.com/prbartosh/token-tycoon/internal/market"
)

const (
	CmdResolveRelease CommandKind = 200 + iota
	CmdSwapArchived
	CmdDeleteArchived
)

const (
	EvReleasePending EventKind = 200 + iota
	EvModelArchived
	EvArchiveSwap
	EvArchiveDeleted
)

const (
	ArchivedNew = iota
	ArchivedDisplaced
)

func ManualTrain(params, tokens, share float64) Command {
	c := Train(params, tokens, share)
	c.On = true
	return c
}

func ResolveRelease(retire int) Command {
	return Command{Kind: CmdResolveRelease, Business: -1, Index: retire}
}

func SwapArchived(archived, active int) Command {
	return Command{Kind: CmdSwapArchived, Business: -1, Index: archived, Qty: active}
}

func DeleteArchived(i int) Command { return Command{Kind: CmdDeleteArchived, Business: -1, Index: i} }

func PendingRelease(s *State) (Model, bool) {
	if len(s.Training.Pending) == 0 {
		return Model{}, false
	}
	return s.Training.Pending[0], true
}

func WeakestModel(s *State) int {
	w := -1
	for k := range s.Training.Models {
		if w < 0 || s.Training.Models[k].Benchmark < s.Training.Models[w].Benchmark {
			w = k
		}
	}
	return w
}

func placeModel(s *State, b *balance.Balance, md Model, ask bool, out []Event) []Event {
	tr := &s.Training
	finishModel(s, &md)
	if len(tr.Models) < MaxModels(s, b) {
		return enterModel(s, b, md, out)
	}
	if ask {
		tr.Pending = append(tr.Pending, md)
		return append(out, Event{Kind: EvReleasePending, Business: -1, Index: len(tr.Pending) - 1, Value: md.Benchmark})
	}
	if w := WeakestModel(s); w >= 0 && md.Benchmark > tr.Models[w].Benchmark {
		out = archiveActive(s, w, out)
		return enterModel(s, b, md, out)
	}
	return archiveModel(s, md, ArchivedNew, out)
}

func archiveModel(s *State, md Model, why int, out []Event) []Event {
	md.Segment = ""
	s.Training.Archive = append(s.Training.Archive, md)
	return append(out, Event{Kind: EvModelArchived, Business: -1, Index: len(s.Training.Archive) - 1, Count: why, Value: md.Benchmark})
}

func archiveActive(s *State, i int, out []Event) []Event {
	tr := &s.Training
	md := tr.Models[i]
	tr.Models = append(tr.Models[:i], tr.Models[i+1:]...)
	return archiveModel(s, md, ArchivedDisplaced, out)
}

func applyArchive(s *State, b *balance.Balance, c *Command, out []Event) []Event {
	tr := &s.Training
	switch c.Kind {
	case CmdResolveRelease:
		if len(tr.Pending) == 0 {
			return reject(out, c, RejectInvalid, 0)
		}
		md := tr.Pending[0]
		md.ReleasedS = s.Clock
		tr.Pending = append(tr.Pending[:0], tr.Pending[1:]...)
		switch {
		case len(tr.Models) < MaxModels(s, b):
			out = enterModel(s, b, md, out)
		case c.Index >= 0 && c.Index < len(tr.Models):
			out = archiveActive(s, c.Index, out)
			out = enterModel(s, b, md, out)
		default:
			out = archiveModel(s, md, ArchivedNew, out)
		}
	case CmdSwapArchived:
		i := c.Index
		if i < 0 || i >= len(tr.Archive) {
			return reject(out, c, RejectInvalid, 0)
		}
		in := tr.Archive[i]
		j := c.Qty
		if len(tr.Models) < MaxModels(s, b) {
			tr.Archive = append(tr.Archive[:i], tr.Archive[i+1:]...)
			tr.Models = append(tr.Models, in)
			out = append(out, Event{Kind: EvArchiveSwap, Business: -1, Index: len(tr.Models) - 1, Count: -1, Value: in.Benchmark})
			break
		}
		if j < 0 || j >= len(tr.Models) {
			return reject(out, c, RejectTeamFull, 0)
		}
		gone := tr.Models[j]
		in.Segment = gone.Segment
		if in.Segment == market.SegmentNames[market.SegEnterprise] && in.Benchmark < b.Market.Segments.EnterpriseMinBenchmark {
			in.Segment = ""
		}
		gone.Segment = ""
		tr.Models[j], tr.Archive[i] = in, gone
		out = append(out, Event{Kind: EvArchiveSwap, Business: -1, Index: j, Count: i, Value: in.Benchmark})
	case CmdDeleteArchived:
		i := c.Index
		if i < 0 || i >= len(tr.Archive) {
			return reject(out, c, RejectInvalid, 0)
		}
		tr.Archive = append(tr.Archive[:i], tr.Archive[i+1:]...)
		out = append(out, Event{Kind: EvArchiveDeleted, Business: -1, Index: i})
	default:
		return reject(out, c, RejectInvalid, 0)
	}
	Recompute(s, b)
	return out
}

func normalizeArchive(s *State, b *balance.Balance) {
	tr := &s.Training
	lim := MaxModels(s, b)
	if extra := len(tr.Models) - lim; extra > 0 {
		for k := range extra {
			md := tr.Models[k]
			md.Segment = ""
			tr.Archive = append(tr.Archive, md)
		}
		tr.Models = append(tr.Models[:0], tr.Models[extra:]...)
	}
	for len(tr.Pending) > 0 && len(tr.Models) < lim {
		tr.Models = append(tr.Models, tr.Pending[0])
		tr.Pending = append(tr.Pending[:0], tr.Pending[1:]...)
	}
}
