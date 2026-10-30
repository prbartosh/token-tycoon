package sim

type EventKind uint8

const (
	EvCycle EventKind = iota + 1
	EvCrit
	EvPurchase
	EvManager
	EvUpgrade
	EvMilestoneSpeed
	EvMilestoneProfit
	EvMilestoneGlobal
	EvUnlock
	EvUndo
	EvRejected
	EvStart
	EvCombo
	EvComboEnd
)

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

type Event struct {
	Kind     EventKind
	Business int
	Index    int
	Count    int
	Amount   float64
	Reason   Reject
	Cmd      CommandKind
}
