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
	EvRack
	EvRackSold
	EvEnergy
	EvPPAExpired
	EvCloud
	EvStarter
	EvGPUGen
	EvPrestige
	EvCard
	EvCardSet
	EvEventStart
	EvEventEnd
	EvReact
	EvReactMiss
	EvGrant
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
	RejectNoEvent
	RejectAlreadyClaimed
	RejectBadDate
)

const (
	EnergyGrid = iota
	EnergyPPA
	EnergySMR
	EnergyOrbital
)

type Event struct {
	Kind     EventKind
	Business int
	Index    int
	Count    int
	Amount   float64
	Value    float64
	Reason   Reject
	Cmd      CommandKind
	Purchase PurchaseKind
}
