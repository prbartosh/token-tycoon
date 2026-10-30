package sim

// EventKind is the type of an event emitted by Step for the UI animation
// layer and the simulator.
type EventKind uint8

const (
	// EvCycle: Business completed Count rolled cycles worth Amount $ (only
	// emitted for rolled cycles, so at most draw_cap_per_s per business).
	EvCycle EventKind = iota + 1
	// EvCrit: a crit on Business added Amount $ in total for that cycle.
	EvCrit
	// EvPurchase: Count units of Business bought for Amount $.
	EvPurchase
	// EvManager: manager of Business hired for Amount $.
	EvManager
	// EvUpgrade: upgrade Index bought for Amount $.
	EvUpgrade
	// EvMilestoneSpeed / EvMilestoneProfit: Business reached Index units;
	// Amount is the new total multiplier of that kind.
	EvMilestoneSpeed
	EvMilestoneProfit
	// EvMilestoneGlobal: all businesses reached Index units (the threshold of
	// the highest global milestone now reached); Amount is the new global
	// multiplier.
	EvMilestoneGlobal
	// EvUnlock: Unlock(Index) opened.
	EvUnlock
	// EvUndo: the last purchase was reverted, Amount $ refunded.
	EvUndo
	// EvRejected: a command failed; Reason says why, Amount is the shortfall
	// for RejectCannotAfford or the threshold for RejectLocked.
	EvRejected
	// EvStart: a manual cycle of Business started.
	EvStart
	// EvCombo: Space pressed, Amount is the new combo multiplier.
	EvCombo
	// EvComboEnd: the combo decayed back to 1.
	EvComboEnd
)

// Reject is why a command was rejected.
type Reject uint8

const (
	RejectNone Reject = iota
	RejectCannotAfford
	RejectLocked
	RejectAlreadyOwned
	RejectNotOwned
	RejectNothingToUndo
	RejectInvalid
	RejectBusy
)

// Event is a flat value type so event slices can be reused without
// allocations.
type Event struct {
	Kind     EventKind
	Business int // -1 when not about a business
	Index    int
	Count    int
	Amount   float64
	Reason   Reject
	Cmd      CommandKind // for EvRejected
}
